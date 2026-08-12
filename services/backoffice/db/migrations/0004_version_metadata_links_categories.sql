-- +goose Up

CREATE TYPE parameter_kind AS ENUM ('STRING', 'BOOLEAN', 'INTEGER', 'FLOAT', 'DATETIME');

CREATE TABLE identifiers (
    -- Identifier identity and RDF value metadata
    id                              uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_iri                      text NOT NULL,
    type                            text NOT NULL,
    value                           text NOT NULL,

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
CREATE UNIQUE INDEX identifiers_dataset_version_content_uq ON identifiers (dataset_version_id, source_iri, type, value) WHERE dataset_version_id IS NOT NULL;
CREATE UNIQUE INDEX identifiers_organization_content_uq ON identifiers (organization_id, source_iri, type, value) WHERE organization_id IS NOT NULL;
CREATE UNIQUE INDEX identifiers_software_application_version_content_uq ON identifiers (software_application_version_id, source_iri, type, value) WHERE software_application_version_id IS NOT NULL;
CREATE UNIQUE INDEX identifiers_software_source_code_version_content_uq ON identifiers (software_source_code_version_id, source_iri, type, value) WHERE software_source_code_version_id IS NOT NULL;

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
CREATE TABLE dataset_version_contact_points (
    dataset_version_id uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,
    contact_point_id   uuid NOT NULL REFERENCES contact_points(id) ON DELETE CASCADE,
    PRIMARY KEY (dataset_version_id, contact_point_id)
);

CREATE TABLE distributions (
    -- Version-owned distribution identity
    id                       uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    dataset_version_id       uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,

    -- Distribution metadata
    source_iri               text NOT NULL,
    title                    text NOT NULL,
    description              text NOT NULL DEFAULT '',
    license                  text,
    source_issued_at         timestamptz,
    source_modified_at       timestamptz,
    byte_size                integer,

    -- Delivery method. Constraint below requires one complete representation.
    type                     text NOT NULL,
    downloadable_file_url    text,
    downloadable_file_format text,
    web_service_id           uuid REFERENCES web_services(id) ON DELETE CASCADE,
    UNIQUE (dataset_version_id, source_iri),
    CHECK (
        (type = 'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE'
            AND downloadable_file_url IS NULL
            AND downloadable_file_format IS NULL
            AND web_service_id IS NOT NULL)
        OR
        (type = 'http://publications.europa.eu/resource/authority/distribution-type/DOWNLOADABLE_FILE'
            AND downloadable_file_url IS NOT NULL
            AND downloadable_file_format IS NOT NULL
            AND web_service_id IS NULL)
    )
);
CREATE TABLE web_services (
    -- Version-owned web service identity
    id                        uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    dataset_version_id        uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,

    -- Web service metadata
    source_iri                text NOT NULL,
    name                      text NOT NULL,
    description               text NOT NULL DEFAULT '',
    template_url              text NOT NULL,
    format                    text NOT NULL,
    provider_organization_id  uuid REFERENCES organizations(id) ON DELETE SET NULL,
    documentation_title       text,
    documentation_description text,
    documentation_uri         text,
    spatial_point             geometry(Point, 4326),
    spatial_polygon           geometry(Polygon, 4326),
    temporal_start            timestamptz,
    temporal_end              timestamptz,

    -- A coverage is either point or polygon; both may be absent.
    CHECK (spatial_point IS NULL OR spatial_polygon IS NULL),
    CHECK (temporal_start IS NULL OR temporal_end IS NULL OR temporal_start <= temporal_end),
    CHECK ((documentation_title IS NULL AND documentation_description IS NULL AND documentation_uri IS NULL) OR documentation_uri IS NOT NULL)
);
CREATE TABLE web_service_contact_points (
    web_service_id   uuid NOT NULL REFERENCES web_services(id) ON DELETE CASCADE,
    contact_point_id uuid NOT NULL REFERENCES contact_points(id) ON DELETE CASCADE,
    PRIMARY KEY (web_service_id, contact_point_id)
);
CREATE TABLE web_service_parameters (
    id              uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_iri      text NOT NULL,
    web_service_id  uuid NOT NULL REFERENCES web_services(id) ON DELETE CASCADE,
    kind            parameter_kind NOT NULL,
    variable        text NOT NULL,
    property        text NOT NULL,
    label           text NOT NULL,
    required        boolean NOT NULL DEFAULT false,
    multiple_values boolean NOT NULL DEFAULT false,
    read_only       boolean NOT NULL DEFAULT false,

	-- Parameter-specific fields require a JSON object.
    config          jsonb NOT NULL DEFAULT '{}'::jsonb,
    CHECK (jsonb_typeof(config) = 'object'),

    UNIQUE (web_service_id, variable)
);

-- group_id forms a composite foreign key with category_id. Migration 0006 adds
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

DROP TABLE IF EXISTS software_source_code_version_categories, software_application_version_categories, dataset_version_categories, web_service_parameters, web_service_contact_points, distributions, web_services, dataset_version_contact_points, software_source_code_version_creators, software_application_version_creators, dataset_version_providers, identifiers;
DROP TYPE IF EXISTS parameter_kind;
