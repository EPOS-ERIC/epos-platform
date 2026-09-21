-- +goose Up

-- Application code checks authorization. These triggers also reject rows that
-- would break core ownership, permission-scope, or lifecycle invariants.

-- +goose StatementBegin
-- Role and permission scopes are fixed by policy. Foreign keys confirm
-- catalog entries exist, but cannot confirm an entry belongs in this table.
CREATE FUNCTION enforce_fixed_authorization_scopes() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    row_data jsonb := to_jsonb(NEW);
BEGIN
    IF TG_TABLE_NAME = 'user_global_roles' AND row_data->>'role_key' <> 'ADMIN' THEN
        RAISE EXCEPTION 'global roles must be ADMIN';
    END IF;

    IF TG_TABLE_NAME = 'group_membership_roles'
        AND row_data->>'role_key' NOT IN ('EDITOR', 'REVIEWER', 'GROUP_ADMIN') THEN
        RAISE EXCEPTION 'invalid group role';
    END IF;

    IF TG_TABLE_NAME = 'role_permissions'
        AND NOT EXISTS (
            SELECT 1
            FROM roles AS role
            JOIN permissions AS permission ON permission.key = row_data->>'permission_key'
            WHERE role.key = row_data->>'role_key'
                AND role.scope = permission.scope
        ) THEN
        RAISE EXCEPTION 'role and permission scopes must match';
    END IF;

    IF TG_TABLE_NAME IN ('group_membership_roles', 'group_permission_grants')
        AND NOT EXISTS (
            SELECT 1
            FROM group_memberships
            WHERE user_id = row_data->>'user_id'
                AND group_id = (row_data->>'group_id')::uuid
                AND status = 'ACTIVE'
        ) THEN
        RAISE EXCEPTION 'group roles and permissions require active membership';
    END IF;

    IF TG_TABLE_NAME = 'group_memberships'
        AND row_data->>'status' <> 'ACTIVE'
        AND (
            EXISTS (
                SELECT 1
                FROM group_membership_roles
                WHERE user_id = row_data->>'user_id'
                    AND group_id = (row_data->>'group_id')::uuid
            )
            OR EXISTS (
                SELECT 1
                FROM group_permission_grants
                WHERE user_id = row_data->>'user_id'
                    AND group_id = (row_data->>'group_id')::uuid
            )
        ) THEN
        RAISE EXCEPTION 'non-active memberships cannot retain group roles or permissions';
    END IF;

    IF TG_TABLE_NAME = 'global_permission_grants'
        AND NOT EXISTS (
            SELECT 1
            FROM permissions
            WHERE key = row_data->>'permission_key' AND scope = 'GLOBAL'
        ) THEN
        RAISE EXCEPTION 'permission must be global';
    END IF;

    IF TG_TABLE_NAME = 'group_permission_grants'
        AND NOT EXISTS (
            SELECT 1
            FROM permissions
            WHERE key = row_data->>'permission_key' AND scope = 'GROUP'
        ) THEN
        RAISE EXCEPTION 'permission must be group scoped';
    END IF;

    RETURN NEW;
END;
$$;

-- A self-referencing foreign key prevents a missing parent, not an indirect
-- cycle. Cycles make category traversal and deletion ambiguous.
CREATE FUNCTION enforce_category_acyclic() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.parent_category_id IS NULL THEN
        RETURN NEW;
    END IF;

    IF EXISTS (
        WITH RECURSIVE ancestors AS (
            SELECT parent_category_id
            FROM categories
            WHERE id = NEW.parent_category_id

            UNION ALL

            SELECT category.parent_category_id
            FROM categories AS category
            JOIN ancestors ON category.id = ancestors.parent_category_id
            WHERE category.parent_category_id IS NOT NULL
        )
        SELECT 1 FROM ancestors WHERE parent_category_id = NEW.id
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'category hierarchy cycle';
    END IF;

    RETURN NEW;
END;
$$;

-- A parent link must not make an organization its own ancestor.
CREATE FUNCTION enforce_organization_acyclic() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.child_organization_id = NEW.parent_organization_id THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'organization hierarchy cycle';
    END IF;

    IF EXISTS (
        WITH RECURSIVE ancestors(id) AS (
            SELECT NEW.parent_organization_id

            UNION ALL

            SELECT membership.parent_organization_id
            FROM organization_memberships AS membership
            JOIN ancestors ON membership.child_organization_id = ancestors.id
        )
        SELECT 1 FROM ancestors WHERE id = NEW.child_organization_id
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'organization hierarchy cycle';
    END IF;

    RETURN NEW;
END;
$$;

-- The composite category foreign key proves a category belongs to supplied
-- group. This additionally prevents callers from supplying a group other than
-- the immutable group of the resource version being categorized.
CREATE FUNCTION enforce_category_resource_group() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    -- NEW has a different row type for each link table. Field access must stay
    -- inside its branch so PL/pgSQL does not resolve absent columns.
    IF TG_TABLE_NAME = 'dataset_version_categories' THEN
        IF NOT EXISTS (
            SELECT 1
            FROM dataset_versions AS version
            JOIN datasets AS dataset ON dataset.id = version.dataset_id
            WHERE version.id = NEW.dataset_version_id AND dataset.group_id = NEW.group_id
        ) THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = 'category group must match dataset group';
        END IF;
    ELSIF TG_TABLE_NAME = 'software_application_version_categories' THEN
        IF NOT EXISTS (
            SELECT 1
            FROM software_application_versions AS version
            JOIN software_applications AS application ON application.id = version.software_application_id
            WHERE version.id = NEW.software_application_version_id AND application.group_id = NEW.group_id
        ) THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = 'category group must match software application group';
        END IF;
    ELSIF TG_TABLE_NAME = 'software_source_code_version_categories' THEN
        IF NOT EXISTS (
            SELECT 1
            FROM software_source_code_versions AS version
            JOIN software_source_codes AS source_code ON source_code.id = version.software_source_code_id
            WHERE version.id = NEW.software_source_code_version_id AND source_code.group_id = NEW.group_id
        ) THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = 'category group must match source code group';
        END IF;
    ELSE
        RAISE EXCEPTION 'unsupported category resource table: %', TG_TABLE_NAME;
    END IF;

    RETURN NEW;
END;
$$;

-- Some records have no application-level update path. Keep modification times
-- accurate regardless of which repository or maintenance operation updates them.
CREATE FUNCTION set_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;

-- Initial installation needs an administrator. Storing this in global roles,
-- rather than a users flag, keeps one source of authority.
CREATE FUNCTION assign_first_user_admin() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    -- Bootstrap first account without retaining a second super-admin authority path.
    IF (SELECT count(*) FROM users) = 1 THEN
        INSERT INTO user_global_roles (user_id, role_key, assigned_by)
        VALUES (NEW.id, 'ADMIN', NEW.id);
    END IF;

    RETURN NEW;
END;
$$;

-- Moving a resource would change its authorization boundary.
CREATE FUNCTION enforce_immutable_logical_resource() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.group_id <> OLD.group_id THEN
        RAISE EXCEPTION 'logical resource group is immutable';
    END IF;

    RETURN NEW;
END;
$$;

-- A category's group identifies canonical shared metadata. Moving it would
-- silently change every resource version that references it.
CREATE FUNCTION enforce_immutable_category() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.group_id <> OLD.group_id THEN
        RAISE EXCEPTION 'category group is immutable';
    END IF;

    RETURN NEW;
END;
$$;

-- Identifiers are copied for a new owner instead of transferred. Otherwise an
-- edit to one resource could silently alter another resource's identifier.
CREATE FUNCTION enforce_immutable_identifier_owner() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.dataset_version_id IS DISTINCT FROM OLD.dataset_version_id
        OR NEW.organization_id IS DISTINCT FROM OLD.organization_id
        OR NEW.software_application_version_id IS DISTINCT FROM OLD.software_application_version_id
        OR NEW.software_source_code_version_id IS DISTINCT FROM OLD.software_source_code_version_id THEN
        RAISE EXCEPTION 'identifier owner is immutable';
    END IF;

    RETURN NEW;
END;
$$;
-- Version rows are mutable only while DRAFT. Lifecycle transitions may update
-- their own attribution fields but may not change version metadata.
CREATE FUNCTION enforce_version_lifecycle_mutability() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    old_data jsonb := to_jsonb(OLD);
    new_data jsonb := to_jsonb(NEW);
    allowed_fields text[];
BEGIN
    IF OLD.status = NEW.status THEN
        IF OLD.status = 'DRAFT' THEN
            RETURN NEW;
        END IF;

        IF old_data - ARRAY['updated_at'] IS DISTINCT FROM new_data - ARRAY['updated_at'] THEN
            RAISE EXCEPTION 'non-draft version is immutable';
        END IF;
        RETURN NEW;
    END IF;

    IF OLD.status = 'DRAFT' AND NEW.status = 'SUBMITTED' THEN
        allowed_fields := ARRAY['status', 'submitted_by', 'submitted_at', 'updated_at'];
    ELSIF OLD.status = 'SUBMITTED' AND NEW.status = 'DRAFT' THEN
        allowed_fields := ARRAY['status', 'updated_at'];
    ELSIF OLD.status = 'SUBMITTED' AND NEW.status = 'REJECTED' THEN
        allowed_fields := ARRAY['status', 'rejected_by', 'rejected_at', 'rejection_note', 'updated_at'];
    ELSIF OLD.status = 'REJECTED' AND NEW.status = 'DRAFT' THEN
        allowed_fields := ARRAY['status', 'updated_at'];
    ELSIF OLD.status = 'SUBMITTED' AND NEW.status = 'PUBLISHED' THEN
        allowed_fields := ARRAY['status', 'published_by', 'published_at', 'rejected_by', 'rejected_at', 'rejection_note', 'updated_at'];
    ELSIF OLD.status = 'PUBLISHED' AND NEW.status = 'ARCHIVED' THEN
        allowed_fields := ARRAY['status', 'archived_by', 'archived_at', 'updated_at'];
    ELSE
        RAISE EXCEPTION 'invalid version lifecycle transition from % to %', OLD.status, NEW.status;
    END IF;

    IF old_data - allowed_fields IS DISTINCT FROM new_data - allowed_fields THEN
        RAISE EXCEPTION 'lifecycle transition cannot change version metadata';
    END IF;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- Run hierarchy validation before the changed parent relation is stored.
CREATE TRIGGER categories_acyclic
BEFORE INSERT OR UPDATE OF parent_category_id ON categories
FOR EACH ROW EXECUTE FUNCTION enforce_category_acyclic();

CREATE TRIGGER dataset_version_categories_group
BEFORE INSERT OR UPDATE ON dataset_version_categories
FOR EACH ROW EXECUTE FUNCTION enforce_category_resource_group();
CREATE TRIGGER software_application_version_categories_group
BEFORE INSERT OR UPDATE ON software_application_version_categories
FOR EACH ROW EXECUTE FUNCTION enforce_category_resource_group();
CREATE TRIGGER software_source_code_version_categories_group
BEFORE INSERT OR UPDATE ON software_source_code_version_categories
FOR EACH ROW EXECUTE FUNCTION enforce_category_resource_group();

-- Keep scope validation adjacent to every table that could otherwise accept a
-- valid catalog key in the wrong authorization scope.
CREATE TRIGGER user_global_roles_scope
BEFORE INSERT OR UPDATE ON user_global_roles
FOR EACH ROW EXECUTE FUNCTION enforce_fixed_authorization_scopes();
CREATE TRIGGER role_permissions_scope
BEFORE INSERT OR UPDATE ON role_permissions
FOR EACH ROW EXECUTE FUNCTION enforce_fixed_authorization_scopes();
CREATE TRIGGER group_membership_roles_scope
BEFORE INSERT OR UPDATE ON group_membership_roles
FOR EACH ROW EXECUTE FUNCTION enforce_fixed_authorization_scopes();
CREATE TRIGGER global_permission_grants_scope
BEFORE INSERT OR UPDATE ON global_permission_grants
FOR EACH ROW EXECUTE FUNCTION enforce_fixed_authorization_scopes();
CREATE TRIGGER group_permission_grants_scope
BEFORE INSERT OR UPDATE ON group_permission_grants
FOR EACH ROW EXECUTE FUNCTION enforce_fixed_authorization_scopes();
CREATE TRIGGER group_memberships_active_assignments
BEFORE UPDATE OF status ON group_memberships
FOR EACH ROW EXECUTE FUNCTION enforce_fixed_authorization_scopes();

-- Record modification time even when change does not use a dedicated handler.
CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER groups_set_updated_at BEFORE UPDATE ON groups FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER categories_set_updated_at BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER group_memberships_set_updated_at BEFORE UPDATE ON group_memberships FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER dataset_versions_set_updated_at BEFORE UPDATE ON dataset_versions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER software_application_versions_set_updated_at BEFORE UPDATE ON software_application_versions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER software_source_code_versions_set_updated_at BEFORE UPDATE ON software_source_code_versions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- Version-owned link mutation guards are deferred until confirmed canonical
-- deletion can perform its documented cross-version unlink transaction.
CREATE TRIGGER dataset_versions_lifecycle_mutability
BEFORE UPDATE ON dataset_versions
FOR EACH ROW EXECUTE FUNCTION enforce_version_lifecycle_mutability();
CREATE TRIGGER software_application_versions_lifecycle_mutability
BEFORE UPDATE ON software_application_versions
FOR EACH ROW EXECUTE FUNCTION enforce_version_lifecycle_mutability();
CREATE TRIGGER software_source_code_versions_lifecycle_mutability
BEFORE UPDATE ON software_source_code_versions
FOR EACH ROW EXECUTE FUNCTION enforce_version_lifecycle_mutability();

-- Establish initial authority only after the user exists for its role foreign key.
CREATE TRIGGER users_assign_first_admin AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION assign_first_user_admin();
-- Prevent authorization/identity fields from being changed after references exist.
CREATE TRIGGER datasets_immutable BEFORE UPDATE ON datasets FOR EACH ROW EXECUTE FUNCTION enforce_immutable_logical_resource();
CREATE TRIGGER software_applications_immutable BEFORE UPDATE ON software_applications FOR EACH ROW EXECUTE FUNCTION enforce_immutable_logical_resource();
CREATE TRIGGER software_source_codes_immutable BEFORE UPDATE ON software_source_codes FOR EACH ROW EXECUTE FUNCTION enforce_immutable_logical_resource();
CREATE TRIGGER categories_immutable BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION enforce_immutable_category();
CREATE TRIGGER organization_memberships_acyclic
BEFORE INSERT OR UPDATE ON organization_memberships
FOR EACH ROW EXECUTE FUNCTION enforce_organization_acyclic();
CREATE TRIGGER identifiers_immutable_owner BEFORE UPDATE ON identifiers FOR EACH ROW EXECUTE FUNCTION enforce_immutable_identifier_owner();
-- +goose Down

DROP TRIGGER IF EXISTS identifiers_immutable_owner ON identifiers;
DROP TRIGGER IF EXISTS categories_immutable ON categories;
DROP TRIGGER IF EXISTS organization_memberships_acyclic ON organization_memberships;
DROP TRIGGER IF EXISTS software_source_codes_immutable ON software_source_codes;
DROP TRIGGER IF EXISTS software_applications_immutable ON software_applications;
DROP TRIGGER IF EXISTS datasets_immutable ON datasets;
DROP TRIGGER IF EXISTS users_assign_first_admin ON users;
DROP TRIGGER IF EXISTS software_source_code_versions_lifecycle_mutability ON software_source_code_versions;
DROP TRIGGER IF EXISTS software_application_versions_lifecycle_mutability ON software_application_versions;
DROP TRIGGER IF EXISTS dataset_versions_lifecycle_mutability ON dataset_versions;
DROP TRIGGER IF EXISTS software_source_code_versions_set_updated_at ON software_source_code_versions;
DROP TRIGGER IF EXISTS software_application_versions_set_updated_at ON software_application_versions;
DROP TRIGGER IF EXISTS dataset_versions_set_updated_at ON dataset_versions;
DROP TRIGGER IF EXISTS group_memberships_set_updated_at ON group_memberships;
DROP TRIGGER IF EXISTS categories_set_updated_at ON categories;
DROP TRIGGER IF EXISTS groups_set_updated_at ON groups;
DROP TRIGGER IF EXISTS users_set_updated_at ON users;
DROP TRIGGER IF EXISTS group_permission_grants_scope ON group_permission_grants;
DROP TRIGGER IF EXISTS global_permission_grants_scope ON global_permission_grants;
DROP TRIGGER IF EXISTS group_membership_roles_scope ON group_membership_roles;
DROP TRIGGER IF EXISTS role_permissions_scope ON role_permissions;
DROP TRIGGER IF EXISTS group_memberships_active_assignments ON group_memberships;
DROP TRIGGER IF EXISTS user_global_roles_scope ON user_global_roles;
DROP TRIGGER IF EXISTS software_source_code_version_categories_group ON software_source_code_version_categories;
DROP TRIGGER IF EXISTS software_application_version_categories_group ON software_application_version_categories;
DROP TRIGGER IF EXISTS dataset_version_categories_group ON dataset_version_categories;
DROP TRIGGER IF EXISTS categories_acyclic ON categories;
DROP FUNCTION IF EXISTS enforce_organization_acyclic(), enforce_version_lifecycle_mutability(), enforce_immutable_identifier_owner(), enforce_immutable_category(), enforce_immutable_logical_resource(), assign_first_user_admin(), set_updated_at(), enforce_category_resource_group(), enforce_category_acyclic(), enforce_fixed_authorization_scopes();
