-- +goose Up

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


-- ============================================================
-- Enums
-- ============================================================

CREATE TYPE user_status AS ENUM (
    'ACTIVE',
    'INACTIVE',
    'BLOCKED'
);

CREATE TYPE membership_role AS ENUM (
    'VIEWER',
    'EDITOR',
    'REVIEWER',
    'ADMIN'
);

CREATE TYPE membership_status AS ENUM (
    'PENDING',
    'ACTIVE',
    'REVOKED',
    'REJECTED'
);

CREATE TYPE permission_effect AS ENUM (
    'ALLOW',
    'DENY'
);

CREATE TYPE dataset_status AS ENUM (
    'PUBLISHED',
    'DRAFT',
    'SUBMITTED',
    'ARCHIVED',
    'REJECTED'
);

CREATE TYPE parameter_kind AS ENUM (
    'STRING',
    'BOOLEAN',
    'INTEGER',
    'FLOAT',
    'DATETIME'
);

CREATE TYPE top_level_concept AS ENUM (
    'DATA',
    'FACILITIES',
    'SOFTWARE'
);


-- ============================================================
-- Users and groups
-- ============================================================

-- Users
CREATE TABLE users (
    id              uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    auth_identifier text NOT NULL UNIQUE,
    family_name     text,
    given_name      text,
    email           text UNIQUE,
    is_super_admin  boolean NOT NULL DEFAULT false,
    status          user_status NOT NULL DEFAULT 'ACTIVE',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- First user becomes super admin so service bootstraps with one trusted account.
-- +goose StatementBegin
CREATE FUNCTION promote_first_user_as_super_admin()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM users) THEN
        NEW.is_super_admin := TRUE;
    END IF;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER users_promote_first_user_as_super_admin
BEFORE INSERT ON users
FOR EACH ROW
EXECUTE FUNCTION promote_first_user_as_super_admin();

-- Groups
CREATE TABLE groups (
    id          uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    slug        text NOT NULL UNIQUE,
    name        text NOT NULL,
    description text,
    is_system   boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- Group memberships
CREATE TABLE group_memberships (
    user_id    uuid NOT NULL
        REFERENCES users(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    group_id   uuid NOT NULL
        REFERENCES groups(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    role       membership_role NOT NULL,
    status     membership_status NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, group_id)
);

-- Group-scoped permission declarations.
CREATE TABLE group_permissions (
    id            uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    group_id      uuid NOT NULL
        REFERENCES groups(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    service       text NOT NULL,
    action        text NOT NULL,
    resource_type text,
    resource_id   text,
    effect        permission_effect NOT NULL DEFAULT 'ALLOW',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

INSERT INTO groups (slug, name, description, is_system)
VALUES ('all', 'ALL', 'Basic group', true)
ON CONFLICT (slug) DO NOTHING;


-- ============================================================
-- Datasets
-- ============================================================

CREATE TABLE datasets (
    id                      uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    source_iri              text NOT NULL UNIQUE,
    title                   text NOT NULL,
    description             text NOT NULL DEFAULT '',

    spatial_point           geometry(Point, 4326),
    spatial_polygon         geometry(Polygon, 4326),

    temporal_start          timestamptz,
    temporal_end            timestamptz,

    status                  dataset_status NOT NULL DEFAULT 'DRAFT',
    editor_user_id          uuid
        REFERENCES users(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,
    group_id                uuid
        REFERENCES groups(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,

    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    source_created_at       timestamptz,
    source_updated_at       timestamptz,

    original_dataset_id     uuid NOT NULL,

    CONSTRAINT datasets_temporal_range_check
        CHECK (
            temporal_start IS NULL
            OR temporal_end IS NULL
            OR temporal_start <= temporal_end
        ),

    CONSTRAINT datasets_original_dataset_fk
        FOREIGN KEY (original_dataset_id)
        REFERENCES datasets(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT
        DEFERRABLE INITIALLY IMMEDIATE
);


-- ============================================================
-- Identifiers
-- Shared by datasets and organizations
-- ============================================================

CREATE TABLE identifiers (
    id          uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    source_iri  text NOT NULL,
    type        text NOT NULL,
    value       text NOT NULL,

    CONSTRAINT identifiers_source_type_value_unique
        UNIQUE (source_iri, type, value)
);

CREATE TABLE dataset_identifiers (
    dataset_id      uuid NOT NULL
        REFERENCES datasets(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    identifier_id   uuid NOT NULL
        REFERENCES identifiers(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,

    PRIMARY KEY (dataset_id, identifier_id)
);


-- ============================================================
-- Organizations
-- ============================================================

CREATE TABLE organizations (
    id              uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    source_iri      text NOT NULL UNIQUE,
    name            text NOT NULL,

    street          text,
    locality        text,
    postal_code     text,
    country         text,

    logo_url        text,
    page_url        text,
    email           text,

    CONSTRAINT organizations_name_unique
        UNIQUE (name)
);

CREATE TABLE organization_identifiers (
    organization_id    uuid NOT NULL
        REFERENCES organizations(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    identifier_id     uuid NOT NULL
        REFERENCES identifiers(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,

    PRIMARY KEY (organization_id, identifier_id)
);

-- child_organization_id is member of parent_organization_id.
CREATE TABLE organization_memberships (
    child_organization_id    uuid NOT NULL
        REFERENCES organizations(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    parent_organization_id   uuid NOT NULL
        REFERENCES organizations(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    PRIMARY KEY (
        child_organization_id,
        parent_organization_id
    ),

    CONSTRAINT organization_memberships_no_self_reference
        CHECK (child_organization_id <> parent_organization_id)
);

CREATE TABLE dataset_providers (
    dataset_id        uuid NOT NULL
        REFERENCES datasets(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    organization_id   uuid NOT NULL
        REFERENCES organizations(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,

    PRIMARY KEY (dataset_id, organization_id)
);


-- ============================================================
-- Categories
-- ============================================================

CREATE TABLE category_schemes (
    id          uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    source_iri  text NOT NULL UNIQUE,
    name        text NOT NULL,
    concept     top_level_concept NOT NULL
);

CREATE TABLE categories (
    id              uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    source_iri      text NOT NULL UNIQUE,
    scheme_id       uuid NOT NULL
        REFERENCES category_schemes(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,
    name            text NOT NULL,

    CONSTRAINT categories_scheme_name_unique
        UNIQUE (scheme_id, name)
);

-- parent_category_id contains child_category_id.
CREATE TABLE category_children (
    parent_category_id   uuid NOT NULL
        REFERENCES categories(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    child_category_id    uuid NOT NULL
        REFERENCES categories(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    PRIMARY KEY (
        parent_category_id,
        child_category_id
    ),

    CONSTRAINT category_children_no_self_reference
        CHECK (parent_category_id <> child_category_id)
);

CREATE TABLE dataset_categories (
    dataset_id    uuid NOT NULL
        REFERENCES datasets(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    category_id   uuid NOT NULL
        REFERENCES categories(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,

    PRIMARY KEY (dataset_id, category_id)
);


-- ============================================================
-- Contact points
-- ============================================================

CREATE TABLE contact_points (
    id          uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    source_iri  text NOT NULL UNIQUE,
    email       text NOT NULL,
    role        text NOT NULL,

    CONSTRAINT contact_points_email_role_unique
        UNIQUE (email, role)
);

CREATE TABLE dataset_contact_points (
    dataset_id        uuid NOT NULL
        REFERENCES datasets(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    contact_point_id  uuid NOT NULL
        REFERENCES contact_points(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,

    PRIMARY KEY (dataset_id, contact_point_id)
);


-- ============================================================
-- Distributions
-- ============================================================

CREATE TABLE distributions (
    id                          uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    source_iri                  text NOT NULL,

    dataset_id                  uuid NOT NULL
        REFERENCES datasets(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    title                       text NOT NULL,
    description                 text NOT NULL DEFAULT '',
    license                     text,

    downloadable_file_url       text,
    downloadable_file_format    text,
    web_service_id              uuid,

    CONSTRAINT distributions_delivery_method_check
        CHECK (
            (
                downloadable_file_url IS NOT NULL
                AND downloadable_file_format IS NOT NULL
                AND web_service_id IS NULL
            )
            OR
            (
                downloadable_file_url IS NULL
                AND downloadable_file_format IS NULL
                AND web_service_id IS NOT NULL
            )
         ),

     CONSTRAINT distributions_source_iri_unique
         UNIQUE (dataset_id, source_iri)
 );


-- ============================================================
-- Web services
-- Independent service definitions referenced by distributions
-- ============================================================

CREATE TABLE web_services (
    id                          uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    source_iri                  text NOT NULL,

    name                        text NOT NULL,
    description                 text NOT NULL DEFAULT '',
    template_url                text NOT NULL,
    format                      text NOT NULL,

    documentation_title         text,
    documentation_description   text,
    documentation_uri           text,

    spatial_point               geometry(Point, 4326),
    spatial_polygon             geometry(Polygon, 4326),

    temporal_start              timestamptz,
    temporal_end                timestamptz,

    CONSTRAINT web_services_temporal_range_check
        CHECK (
            temporal_start IS NULL
            OR temporal_end IS NULL
            OR temporal_start <= temporal_end
        ),

    CONSTRAINT web_services_documentation_check
        CHECK (
            (
                documentation_title IS NULL
                AND documentation_description IS NULL
                AND documentation_uri IS NULL
            )
            OR documentation_uri IS NOT NULL
        )
    );

ALTER TABLE distributions
    ADD CONSTRAINT distributions_web_service_fk
    FOREIGN KEY (web_service_id)
    REFERENCES web_services(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE;

CREATE TABLE web_service_providers (
    web_service_id    uuid NOT NULL
        REFERENCES web_services(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    organization_id   uuid NOT NULL
        REFERENCES organizations(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,

    PRIMARY KEY (web_service_id, organization_id)
);

CREATE TABLE web_service_contact_points (
    web_service_id     uuid NOT NULL
        REFERENCES web_services(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    contact_point_id   uuid NOT NULL
        REFERENCES contact_points(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,

    PRIMARY KEY (web_service_id, contact_point_id)
);


-- ============================================================
-- Parameters
-- ============================================================

CREATE TABLE web_service_parameters (
    id                   uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    source_iri           text NOT NULL,

    web_service_id       uuid NOT NULL
        REFERENCES web_services(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    kind                 parameter_kind NOT NULL,

    variable             text NOT NULL,
    property             text NOT NULL,
    label                text NOT NULL,

    required             boolean NOT NULL DEFAULT false,
    multiple_values      boolean NOT NULL DEFAULT false,
    read_only            boolean NOT NULL DEFAULT false,

    -- Type-specific fields.
    --
    -- string:
    -- {
    --   "default": "value",
    --   "allowed_values": ["a", "b"],
    --   "pattern": "..."
    -- }
    --
    -- boolean:
    -- {
    --   "default": true
    -- }
    --
    -- integer:
    -- {
    --   "minimum": 0,
    --   "maximum": 100,
    --   "default": 10,
    --   "allowed_values": [10, 20]
    -- }
    --
    -- float:
    -- {
    --   "minimum": 0.5,
    --   "maximum": 10.5,
    --   "default": 1.5,
    --   "allowed_values": [1.5, 2.5]
    -- }
    --
    -- datetime:
    -- {
    --   "minimum": "2026-01-01T00:00:00Z",
    --   "maximum": "2026-12-31T23:59:59Z",
    --   "default": "2026-06-01T00:00:00Z",
    --   "pattern": "..."
    -- }
    config               jsonb NOT NULL DEFAULT '{}'::jsonb,

    CONSTRAINT web_service_parameters_config_object_check
        CHECK (jsonb_typeof(config) = 'object'),

    CONSTRAINT web_service_parameters_variable_unique
        UNIQUE (web_service_id, variable)
);


-- ============================================================
-- Dataset defaults and timestamps
-- ============================================================

-- +goose StatementBegin
CREATE FUNCTION set_dataset_original_id()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.original_dataset_id IS NULL THEN
        NEW.original_dataset_id := NEW.id;
    END IF;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER datasets_set_original_id
BEFORE INSERT ON datasets
FOR EACH ROW
EXECUTE FUNCTION set_dataset_original_id();


-- +goose StatementBegin
CREATE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER datasets_set_updated_at
BEFORE UPDATE ON datasets
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- ============================================================
-- Indexes
-- ============================================================

CREATE INDEX datasets_status_idx
    ON datasets(status);

CREATE INDEX datasets_editor_user_id_idx
    ON datasets(editor_user_id);

CREATE INDEX datasets_group_id_idx
    ON datasets(group_id);

CREATE INDEX users_auth_identifier_idx ON users (auth_identifier);
CREATE INDEX group_memberships_group_status_idx ON group_memberships (group_id, status);
CREATE UNIQUE INDEX group_permissions_scope_uq
    ON group_permissions (group_id, service, action, coalesce(resource_type, ''), coalesce(resource_id, ''), effect);
CREATE INDEX group_permissions_service_action_idx ON group_permissions (service, action);

CREATE INDEX datasets_original_dataset_id_idx
    ON datasets(original_dataset_id);

CREATE INDEX datasets_spatial_point_gist_idx
    ON datasets
    USING gist(spatial_point);

CREATE INDEX datasets_spatial_polygon_gist_idx
    ON datasets
    USING gist(spatial_polygon);

CREATE INDEX dataset_identifiers_identifier_id_idx
    ON dataset_identifiers(identifier_id);

CREATE INDEX organization_identifiers_identifier_id_idx
    ON organization_identifiers(identifier_id);

CREATE INDEX organization_memberships_parent_idx
    ON organization_memberships(parent_organization_id);

CREATE INDEX category_children_child_idx
    ON category_children(child_category_id);

CREATE INDEX distributions_dataset_id_idx
    ON distributions(dataset_id);

CREATE INDEX distributions_web_service_id_idx
    ON distributions(web_service_id);

CREATE INDEX web_services_spatial_point_gist_idx
    ON web_services
    USING gist(spatial_point);

CREATE INDEX web_services_spatial_polygon_gist_idx
    ON web_services
    USING gist(spatial_polygon);

CREATE INDEX web_service_parameters_web_service_id_idx
    ON web_service_parameters(web_service_id);

CREATE INDEX web_service_parameters_property_idx
    ON web_service_parameters(property);

CREATE INDEX web_service_parameters_config_gin_idx
    ON web_service_parameters
    USING gin(config);

-- +goose Down

DROP TABLE IF EXISTS web_service_parameters;
DROP TABLE IF EXISTS web_service_contact_points;
DROP TABLE IF EXISTS web_service_providers;
DROP TABLE IF EXISTS distributions;
DROP TABLE IF EXISTS web_services;
DROP TABLE IF EXISTS dataset_contact_points;
DROP TABLE IF EXISTS dataset_categories;
DROP TABLE IF EXISTS dataset_providers;
DROP TABLE IF EXISTS dataset_identifiers;
DROP TABLE IF EXISTS organization_identifiers;
DROP TABLE IF EXISTS organization_memberships;
DROP TABLE IF EXISTS category_children;
DROP TABLE IF EXISTS contact_points;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS category_schemes;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS identifiers;
DROP TABLE IF EXISTS datasets;

DROP FUNCTION IF EXISTS set_updated_at();
DROP FUNCTION IF EXISTS set_dataset_original_id();

DROP TABLE IF EXISTS group_permissions;
DROP TABLE IF EXISTS group_memberships;
DROP TABLE IF EXISTS groups;
DROP TRIGGER IF EXISTS users_promote_first_user_as_super_admin ON users;
DROP FUNCTION IF EXISTS promote_first_user_as_super_admin();
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS permission_effect;
DROP TYPE IF EXISTS membership_status;
DROP TYPE IF EXISTS membership_role;
DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS parameter_kind;
DROP TYPE IF EXISTS top_level_concept;
DROP TYPE IF EXISTS dataset_status;
