-- +goose Up

CREATE TYPE resource_status AS ENUM ('DRAFT', 'SUBMITTED', 'REJECTED', 'PUBLISHED', 'ARCHIVED');

CREATE TABLE datasets (
    id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id   uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    created_by text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE software_applications (
    id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id   uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    created_by text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE software_source_codes (
    id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id   uuid NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    created_by text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE dataset_versions (
    -- Identity and lifecycle concurrency
    id                  uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    dataset_id          uuid NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    status              resource_status NOT NULL DEFAULT 'DRAFT',
    revision            positive_int NOT NULL DEFAULT 1,

    -- Version-owned dataset metadata
    title               nonblank_text NOT NULL,
    description         nonblank_text NOT NULL,
    keywords            nonblank_text[] NOT NULL DEFAULT '{}',
    accrual_periodicity nonblank_text_opt,
    quality_annotation  nonblank_text_opt,
    access_right        nonblank_text_opt,
    -- Source timestamps describe upstream dataset metadata. They are distinct
    -- from metadata-lifecycle record and lifecycle timestamps below.
    source_created_at   timestamptz,
    source_updated_at   timestamptz,
    spatial_coverage    geometry(Geometry, 4326),
    temporal_start      timestamptz,
    temporal_end        timestamptz,
    geological_temporal_start_mya double precision,
    geological_temporal_end_mya   double precision,

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
    rejection_note      nonblank_text_opt,

    CHECK (temporal_start IS NULL OR temporal_end IS NULL OR temporal_start <= temporal_end),
    CHECK (
        geological_temporal_start_mya IS NULL
        OR (
            geological_temporal_start_mya >= 0
            AND geological_temporal_start_mya NOT IN (
                'NaN'::double precision,
                'Infinity'::double precision,
                '-Infinity'::double precision
            )
        )
    ),
    CHECK (
        geological_temporal_end_mya IS NULL
        OR (
            geological_temporal_end_mya >= 0
            AND geological_temporal_end_mya NOT IN (
                'NaN'::double precision,
                'Infinity'::double precision,
                '-Infinity'::double precision
            )
        )
    ),
    CHECK (
        geological_temporal_start_mya IS NULL
        OR geological_temporal_end_mya IS NULL
        OR geological_temporal_start_mya >= geological_temporal_end_mya
    ),
    CHECK (
        (temporal_start IS NULL AND temporal_end IS NULL)
        OR (
            geological_temporal_start_mya IS NULL
            AND geological_temporal_end_mya IS NULL
        )
    ),
    CHECK (array_position(keywords, NULL) IS NULL)
);
CREATE TABLE software_application_versions (
    -- Identity and lifecycle concurrency
    id                      uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    software_application_id uuid NOT NULL REFERENCES software_applications(id) ON DELETE CASCADE,
    status                  resource_status NOT NULL DEFAULT 'DRAFT',
    revision                positive_int NOT NULL DEFAULT 1,

    -- Version-owned application metadata
    name                    nonblank_text NOT NULL,
    description             nonblank_text NOT NULL,
    main_entity_of_page     nonblank_text_opt,
    license                 nonblank_text_opt,
    version                 nonblank_text_opt,
    keywords                nonblank_text[] NOT NULL DEFAULT '{}',
    download_url            nonblank_text_opt,
    requirements            nonblank_text_opt,
    operating_system        nonblank_text_opt,

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
    rejection_note          nonblank_text_opt,
    CHECK (array_position(keywords, NULL) IS NULL)
);
CREATE TABLE software_source_code_versions (
    -- Identity and lifecycle concurrency
    id                      uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    software_source_code_id uuid NOT NULL REFERENCES software_source_codes(id) ON DELETE CASCADE,
    status                  resource_status NOT NULL DEFAULT 'DRAFT',
    revision                positive_int NOT NULL DEFAULT 1,

    -- Version-owned source-code metadata
    name                    nonblank_text NOT NULL,
    description             nonblank_text NOT NULL,
    programming_language    nonblank_text_opt,
    keywords                nonblank_text[] NOT NULL DEFAULT '{}',
    repository              nonblank_text_opt,
    license                 nonblank_text_opt,
    software_requirements   nonblank_text_opt,
    runtime_platform        nonblank_text_opt,
    main_entity_of_page     nonblank_text_opt,
    version                 nonblank_text_opt,

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
    rejection_note          nonblank_text_opt,
    CHECK (array_position(keywords, NULL) IS NULL)
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
