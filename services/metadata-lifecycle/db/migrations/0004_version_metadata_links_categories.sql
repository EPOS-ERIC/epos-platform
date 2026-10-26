-- +goose Up

CREATE TABLE identifiers (
    -- Identifier identity and RDF value metadata
    id                              uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    type                            nonblank_text NOT NULL,
    value                           nonblank_text NOT NULL,

    -- Exactly one immutable owner
    dataset_version_id              uuid REFERENCES dataset_versions(id) ON DELETE CASCADE,
    organization_id                 uuid REFERENCES organizations(id) ON DELETE CASCADE,
    software_application_version_id uuid REFERENCES software_application_versions(id) ON DELETE CASCADE,
    software_source_code_version_id uuid REFERENCES software_source_code_versions(id) ON DELETE CASCADE,

    -- Identifiers are owned values, never shared canonical records.
    CHECK (num_nonnulls(
        dataset_version_id,
        organization_id,
        software_application_version_id,
        software_source_code_version_id
    ) = 1)
);
CREATE UNIQUE INDEX identifiers_dataset_version_content_uq ON identifiers (dataset_version_id, type, value) WHERE dataset_version_id IS NOT NULL;
CREATE UNIQUE INDEX identifiers_organization_content_uq ON identifiers (organization_id, type, value) WHERE organization_id IS NOT NULL;
CREATE UNIQUE INDEX identifiers_software_application_version_content_uq ON identifiers (software_application_version_id, type, value) WHERE software_application_version_id IS NOT NULL;
CREATE UNIQUE INDEX identifiers_software_source_code_version_content_uq ON identifiers (software_source_code_version_id, type, value) WHERE software_source_code_version_id IS NOT NULL;

CREATE TABLE dataset_version_providers (
    dataset_version_id uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,
    organization_id    uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    PRIMARY KEY (dataset_version_id, organization_id)
);
CREATE TABLE software_application_version_creators (
    software_application_version_id uuid NOT NULL REFERENCES software_application_versions(id) ON DELETE CASCADE,
    organization_id                 uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    PRIMARY KEY (software_application_version_id, organization_id)
);
CREATE TABLE software_source_code_version_creators (
    software_source_code_version_id uuid NOT NULL REFERENCES software_source_code_versions(id) ON DELETE CASCADE,
    organization_id                 uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    PRIMARY KEY (software_source_code_version_id, organization_id)
);
CREATE TABLE software_application_version_contact_points (
    software_application_version_id uuid NOT NULL REFERENCES software_application_versions(id) ON DELETE CASCADE,
    contact_point_id               uuid NOT NULL REFERENCES contact_points(id) ON DELETE CASCADE,
    PRIMARY KEY (software_application_version_id, contact_point_id)
);
CREATE TABLE software_source_code_version_contact_points (
    software_source_code_version_id uuid NOT NULL REFERENCES software_source_code_versions(id) ON DELETE CASCADE,
    contact_point_id                uuid NOT NULL REFERENCES contact_points(id) ON DELETE CASCADE,
    PRIMARY KEY (software_source_code_version_id, contact_point_id)
);
CREATE TABLE dataset_version_contact_points (
    dataset_version_id uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,
    contact_point_id   uuid NOT NULL REFERENCES contact_points(id) ON DELETE CASCADE,
    PRIMARY KEY (dataset_version_id, contact_point_id)
);

-- group_id forms a composite foreign key with category_id. Migration 0008 adds
-- a trigger that verifies it matches owning logical resource's immutable group.
CREATE TABLE dataset_version_categories (
    dataset_version_id uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,
    group_id           uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    category_id        uuid NOT NULL,
    PRIMARY KEY (dataset_version_id, category_id),
    FOREIGN KEY (category_id, group_id) REFERENCES categories(id, group_id) ON DELETE CASCADE
);
CREATE TABLE software_application_version_categories (
    software_application_version_id uuid NOT NULL REFERENCES software_application_versions(id) ON DELETE CASCADE,
    group_id                        uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    category_id                     uuid NOT NULL,
    PRIMARY KEY (software_application_version_id, category_id),
    FOREIGN KEY (category_id, group_id) REFERENCES categories(id, group_id) ON DELETE CASCADE
);
CREATE TABLE software_source_code_version_categories (
    software_source_code_version_id uuid NOT NULL REFERENCES software_source_code_versions(id) ON DELETE CASCADE,
    group_id                        uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    category_id                     uuid NOT NULL,
    PRIMARY KEY (software_source_code_version_id, category_id),
    FOREIGN KEY (category_id, group_id) REFERENCES categories(id, group_id) ON DELETE CASCADE
);

-- +goose Down

DROP TABLE IF EXISTS software_source_code_version_categories, software_application_version_categories, dataset_version_categories, dataset_version_contact_points, software_source_code_version_contact_points, software_application_version_contact_points, software_source_code_version_creators, software_application_version_creators, dataset_version_providers, identifiers;
