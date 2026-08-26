-- +goose Up

CREATE TABLE categories (
    -- Canonical category identity and owning scheme
    id                 uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id           uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    parent_category_id uuid,

    -- Category metadata
    name               text NOT NULL,

    -- Metadata Lifecycle record timestamps
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, group_id),
    FOREIGN KEY (parent_category_id, group_id) REFERENCES categories(id, group_id) ON DELETE CASCADE,
    CHECK (parent_category_id IS NULL OR parent_category_id <> id),
    CHECK (length(trim(name)) > 0)
);

-- Store only display name. Expression indexes enforce trim/case-insensitive
-- sibling uniqueness in PostgreSQL, so concurrent writes cannot bypass it.
-- Separate partial indexes distinguish top-level names from sibling names.
CREATE UNIQUE INDEX categories_top_level_name_uq
    ON categories (group_id, lower(trim(name)))
    WHERE parent_category_id IS NULL;
CREATE UNIQUE INDEX categories_sibling_name_uq
    ON categories (group_id, parent_category_id, lower(trim(name)))
    WHERE parent_category_id IS NOT NULL;

CREATE TABLE organizations (
    -- Canonical organization identity
    id          uuid PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Schema.org organization metadata
    name        text NOT NULL UNIQUE,
    street      text,
    locality    text,
    postal_code text,
    country     text,
    logo_url    text,
    page_url    text,
    email       text,
    lei_code    text
);
CREATE TABLE organization_memberships (
    child_organization_id  uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    parent_organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    PRIMARY KEY (child_organization_id, parent_organization_id),
    CHECK (child_organization_id <> parent_organization_id)
);
CREATE TABLE contact_points (
    -- Canonical contact-point identity
    id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Contact-point metadata
    email      text NOT NULL,
    role       text NOT NULL,

    UNIQUE (email, role)
);
CREATE TABLE organization_contact_points (
    organization_id  uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    contact_point_id uuid NOT NULL REFERENCES contact_points(id) ON DELETE CASCADE,
    PRIMARY KEY (organization_id, contact_point_id)
);

-- +goose Down

DROP TABLE IF EXISTS organization_contact_points, contact_points, organization_memberships, organizations, categories;
