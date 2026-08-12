-- +goose Up

-- Application code checks authorization. These triggers also reject rows that
-- would break core ownership, permission-scope, or lifecycle invariants.

-- +goose StatementBegin
-- Role and permission scopes are fixed by policy. Foreign keys confirm
-- catalog entries exist, but cannot confirm an entry belongs in this table.
CREATE FUNCTION enforce_fixed_authorization_scopes() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_TABLE_NAME = 'user_global_roles' AND NEW.role_key <> 'ADMIN' THEN
        RAISE EXCEPTION 'global roles must be ADMIN';
    END IF;

    IF TG_TABLE_NAME = 'group_membership_roles'
        AND NEW.role_key NOT IN ('EDITOR', 'REVIEWER', 'GROUP_ADMIN') THEN
        RAISE EXCEPTION 'invalid group role';
    END IF;

    IF TG_TABLE_NAME = 'global_permission_grants'
        AND NOT EXISTS (SELECT 1 FROM permissions WHERE key = NEW.permission_key AND scope = 'GLOBAL') THEN
        RAISE EXCEPTION 'permission must be global';
    END IF;

    IF TG_TABLE_NAME = 'group_permission_grants'
        AND NOT EXISTS (SELECT 1 FROM permissions WHERE key = NEW.permission_key AND scope = 'GROUP') THEN
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
        RAISE EXCEPTION 'category hierarchy cycle';
    END IF;

    RETURN NEW;
END;
$$;

-- The composite category foreign key proves a category belongs to supplied
-- group. This additionally prevents callers from supplying a group other than
-- the immutable group of the resource version being categorized.
CREATE FUNCTION enforce_category_resource_group() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_TABLE_NAME = 'dataset_version_categories' AND NOT EXISTS (
        SELECT 1
        FROM dataset_versions AS version
        JOIN datasets AS dataset ON dataset.id = version.dataset_id
        WHERE version.id = NEW.dataset_version_id AND dataset.group_id = NEW.group_id
    ) THEN
        RAISE EXCEPTION 'category group must match dataset group';
    END IF;

    IF TG_TABLE_NAME = 'software_application_version_categories' AND NOT EXISTS (
        SELECT 1
        FROM software_application_versions AS version
        JOIN software_applications AS application ON application.id = version.software_application_id
        WHERE version.id = NEW.software_application_version_id AND application.group_id = NEW.group_id
    ) THEN
        RAISE EXCEPTION 'category group must match software application group';
    END IF;

    IF TG_TABLE_NAME = 'software_source_code_version_categories' AND NOT EXISTS (
        SELECT 1
        FROM software_source_code_versions AS version
        JOIN software_source_codes AS source_code ON source_code.id = version.software_source_code_id
        WHERE version.id = NEW.software_source_code_version_id AND source_code.group_id = NEW.group_id
    ) THEN
        RAISE EXCEPTION 'category group must match source code group';
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

-- Moving a resource would change its authorization boundary; changing source
-- IRI would change the stable identity shared by every version.
CREATE FUNCTION enforce_immutable_logical_resource() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.source_iri <> OLD.source_iri OR NEW.group_id <> OLD.group_id THEN
        RAISE EXCEPTION 'logical resource source IRI and group are immutable';
    END IF;

    RETURN NEW;
END;
$$;

-- A category's source IRI and group identify canonical shared metadata. Moving
-- either would silently change every resource version that references it.
CREATE FUNCTION enforce_immutable_category() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.source_iri <> OLD.source_iri OR NEW.group_id <> OLD.group_id THEN
        RAISE EXCEPTION 'category source IRI and group are immutable';
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
CREATE FUNCTION enforce_resource_source_iri_global_unique() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    -- PostgreSQL cannot express uniqueness across three tables with an index.
    IF TG_TABLE_NAME <> 'datasets'
        AND EXISTS (SELECT 1 FROM datasets WHERE source_iri = NEW.source_iri) THEN
        RAISE EXCEPTION 'resource source IRI must be globally unique';
    END IF;

    IF TG_TABLE_NAME <> 'software_applications'
        AND EXISTS (SELECT 1 FROM software_applications WHERE source_iri = NEW.source_iri) THEN
        RAISE EXCEPTION 'resource source IRI must be globally unique';
    END IF;

    IF TG_TABLE_NAME <> 'software_source_codes'
        AND EXISTS (SELECT 1 FROM software_source_codes WHERE source_iri = NEW.source_iri) THEN
        RAISE EXCEPTION 'resource source IRI must be globally unique';
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
CREATE TRIGGER group_membership_roles_scope
BEFORE INSERT OR UPDATE ON group_membership_roles
FOR EACH ROW EXECUTE FUNCTION enforce_fixed_authorization_scopes();
CREATE TRIGGER global_permission_grants_scope
BEFORE INSERT OR UPDATE ON global_permission_grants
FOR EACH ROW EXECUTE FUNCTION enforce_fixed_authorization_scopes();
CREATE TRIGGER group_permission_grants_scope
BEFORE INSERT OR UPDATE ON group_permission_grants
FOR EACH ROW EXECUTE FUNCTION enforce_fixed_authorization_scopes();

-- Record modification time even when change does not use a dedicated handler.
CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER groups_set_updated_at BEFORE UPDATE ON groups FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER categories_set_updated_at BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER group_memberships_set_updated_at BEFORE UPDATE ON group_memberships FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER dataset_versions_set_updated_at BEFORE UPDATE ON dataset_versions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER software_application_versions_set_updated_at BEFORE UPDATE ON software_application_versions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER software_source_code_versions_set_updated_at BEFORE UPDATE ON software_source_code_versions FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Establish initial authority only after the user exists for its role foreign key.
CREATE TRIGGER users_assign_first_admin AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION assign_first_user_admin();
-- Prevent authorization/identity fields from being changed after references exist.
CREATE TRIGGER datasets_immutable BEFORE UPDATE ON datasets FOR EACH ROW EXECUTE FUNCTION enforce_immutable_logical_resource();
CREATE TRIGGER software_applications_immutable BEFORE UPDATE ON software_applications FOR EACH ROW EXECUTE FUNCTION enforce_immutable_logical_resource();
CREATE TRIGGER software_source_codes_immutable BEFORE UPDATE ON software_source_codes FOR EACH ROW EXECUTE FUNCTION enforce_immutable_logical_resource();
CREATE TRIGGER categories_immutable BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION enforce_immutable_category();
CREATE TRIGGER identifiers_immutable_owner BEFORE UPDATE ON identifiers FOR EACH ROW EXECUTE FUNCTION enforce_immutable_identifier_owner();
-- Enforce source IRI uniqueness across logical resource tables, which a normal
-- per-table unique constraint cannot express.
CREATE TRIGGER datasets_source_iri_global_unique BEFORE INSERT OR UPDATE OF source_iri ON datasets FOR EACH ROW EXECUTE FUNCTION enforce_resource_source_iri_global_unique();
CREATE TRIGGER software_applications_source_iri_global_unique BEFORE INSERT OR UPDATE OF source_iri ON software_applications FOR EACH ROW EXECUTE FUNCTION enforce_resource_source_iri_global_unique();
CREATE TRIGGER software_source_codes_source_iri_global_unique BEFORE INSERT OR UPDATE OF source_iri ON software_source_codes FOR EACH ROW EXECUTE FUNCTION enforce_resource_source_iri_global_unique();

-- +goose Down

DROP TRIGGER IF EXISTS software_source_codes_source_iri_global_unique ON software_source_codes;
DROP TRIGGER IF EXISTS software_applications_source_iri_global_unique ON software_applications;
DROP TRIGGER IF EXISTS datasets_source_iri_global_unique ON datasets;
DROP TRIGGER IF EXISTS identifiers_immutable_owner ON identifiers;
DROP TRIGGER IF EXISTS categories_immutable ON categories;
DROP TRIGGER IF EXISTS software_source_codes_immutable ON software_source_codes;
DROP TRIGGER IF EXISTS software_applications_immutable ON software_applications;
DROP TRIGGER IF EXISTS datasets_immutable ON datasets;
DROP TRIGGER IF EXISTS users_assign_first_admin ON users;
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
DROP TRIGGER IF EXISTS user_global_roles_scope ON user_global_roles;
DROP TRIGGER IF EXISTS software_source_code_version_categories_group ON software_source_code_version_categories;
DROP TRIGGER IF EXISTS software_application_version_categories_group ON software_application_version_categories;
DROP TRIGGER IF EXISTS dataset_version_categories_group ON dataset_version_categories;
DROP TRIGGER IF EXISTS categories_acyclic ON categories;
DROP FUNCTION IF EXISTS enforce_resource_source_iri_global_unique(), enforce_immutable_identifier_owner(), enforce_immutable_category(), enforce_immutable_logical_resource(), assign_first_user_admin(), set_updated_at(), enforce_category_resource_group(), enforce_category_acyclic(), enforce_fixed_authorization_scopes();
