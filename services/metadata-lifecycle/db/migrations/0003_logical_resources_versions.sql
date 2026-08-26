-- +goose Up

CREATE TYPE resource_status AS ENUM ('DRAFT', 'SUBMITTED', 'REJECTED', 'PUBLISHED', 'ARCHIVED');

CREATE TABLE datasets (
    id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_iri text NOT NULL UNIQUE,
    group_id   uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    created_by text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE software_applications (
    id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_iri text NOT NULL UNIQUE,
    group_id   uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    created_by text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE software_source_codes (
    id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_iri text NOT NULL UNIQUE,
    group_id   uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    created_by text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE dataset_versions (
    -- Identity and lifecycle concurrency
    id                  uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    dataset_id          uuid NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    status              resource_status NOT NULL DEFAULT 'DRAFT',
    revision            integer NOT NULL DEFAULT 1 CHECK (revision > 0),

    -- Version-owned dataset metadata
    title               text NOT NULL,
    description         text NOT NULL DEFAULT '',
    keywords            text NOT NULL DEFAULT '',
    accrual_periodicity text,
    quality_annotation  text,
    access_right        text,
    -- Source timestamps describe upstream dataset metadata. They are distinct
    -- from metadata-lifecycle record and lifecycle timestamps below.
    source_created_at   timestamptz,
    source_updated_at   timestamptz,
    spatial_point       geometry(Point, 4326),
    spatial_polygon     geometry(Polygon, 4326),
    temporal_start      timestamptz,
    temporal_end        timestamptz,

    -- Metadata Lifecycle record timestamps
    created_by          text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),

    -- Lifecycle transition attribution; rejection stores latest decision only
    submitted_by        text REFERENCES users(id) ON DELETE RESTRICT,
    submitted_at        timestamptz,
    published_by        text REFERENCES users(id) ON DELETE RESTRICT,
    published_at        timestamptz,
    archived_by         text REFERENCES users(id) ON DELETE RESTRICT,
    archived_at         timestamptz,
    rejected_by         text REFERENCES users(id) ON DELETE RESTRICT,
    rejected_at         timestamptz,
    rejection_note      text,

    -- A coverage is either point or polygon; both may be absent.
    CHECK (spatial_point IS NULL OR spatial_polygon IS NULL),
    CHECK (temporal_start IS NULL OR temporal_end IS NULL OR temporal_start <= temporal_end),
    CHECK (rejection_note IS NULL OR length(trim(rejection_note)) > 0)
);
CREATE TABLE software_application_versions (
    -- Identity and lifecycle concurrency
    id                      uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    software_application_id uuid NOT NULL REFERENCES software_applications(id) ON DELETE CASCADE,
    status                  resource_status NOT NULL DEFAULT 'DRAFT',
    revision                integer NOT NULL DEFAULT 1 CHECK (revision > 0),

    -- Version-owned application metadata
    name                    text NOT NULL,
    description             text NOT NULL DEFAULT '',
    main_entity_of_page     text,
    license                 text,
    version                 text,
    keywords                text NOT NULL DEFAULT '',
    download_url            text,
    requirements            text,
    operating_system        text,

    -- Metadata Lifecycle record timestamps
    created_by              text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),

    -- Lifecycle transition attribution; rejection stores latest decision only
    submitted_by            text REFERENCES users(id) ON DELETE RESTRICT,
    submitted_at            timestamptz,
    published_by            text REFERENCES users(id) ON DELETE RESTRICT,
    published_at            timestamptz,
    archived_by             text REFERENCES users(id) ON DELETE RESTRICT,
    archived_at             timestamptz,
    rejected_by             text REFERENCES users(id) ON DELETE RESTRICT,
    rejected_at             timestamptz,
    rejection_note          text,
    CHECK (rejection_note IS NULL OR length(trim(rejection_note)) > 0)
);
CREATE TABLE software_source_code_versions (
    -- Identity and lifecycle concurrency
    id                      uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    software_source_code_id uuid NOT NULL REFERENCES software_source_codes(id) ON DELETE CASCADE,
    status                  resource_status NOT NULL DEFAULT 'DRAFT',
    revision                integer NOT NULL DEFAULT 1 CHECK (revision > 0),

    -- Version-owned source-code metadata
    name                    text NOT NULL,
    description             text NOT NULL DEFAULT '',
    programming_language    text,
    keywords                text NOT NULL DEFAULT '',
    repository              text,
    license                 text,
    software_requirements   text,
    runtime_platform        text,
    main_entity_of_page     text,
    version                 text,

    -- Metadata Lifecycle record timestamps
    created_by              text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),

    -- Lifecycle transition attribution; rejection stores latest decision only
    submitted_by            text REFERENCES users(id) ON DELETE RESTRICT,
    submitted_at            timestamptz,
    published_by            text REFERENCES users(id) ON DELETE RESTRICT,
    published_at            timestamptz,
    archived_by             text REFERENCES users(id) ON DELETE RESTRICT,
    archived_at             timestamptz,
    rejected_by             text REFERENCES users(id) ON DELETE RESTRICT,
    rejected_at             timestamptz,
    rejection_note          text,
    CHECK (rejection_note IS NULL OR length(trim(rejection_note)) > 0)
);

-- A resource has one pending change and one current public version at most.
CREATE UNIQUE INDEX dataset_versions_active_change_uq ON dataset_versions (dataset_id) WHERE status IN ('DRAFT', 'SUBMITTED', 'REJECTED');
CREATE UNIQUE INDEX dataset_versions_published_uq ON dataset_versions (dataset_id) WHERE status = 'PUBLISHED';
CREATE UNIQUE INDEX software_application_versions_active_change_uq ON software_application_versions (software_application_id) WHERE status IN ('DRAFT', 'SUBMITTED', 'REJECTED');
CREATE UNIQUE INDEX software_application_versions_published_uq ON software_application_versions (software_application_id) WHERE status = 'PUBLISHED';
CREATE UNIQUE INDEX software_source_code_versions_active_change_uq ON software_source_code_versions (software_source_code_id) WHERE status IN ('DRAFT', 'SUBMITTED', 'REJECTED');
CREATE UNIQUE INDEX software_source_code_versions_published_uq ON software_source_code_versions (software_source_code_id) WHERE status = 'PUBLISHED';

-- +goose Down

DROP TABLE IF EXISTS software_source_code_versions, software_application_versions, dataset_versions, software_source_codes, software_applications, datasets;
DROP TYPE IF EXISTS resource_status;
