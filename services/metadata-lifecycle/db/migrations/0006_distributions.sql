-- +goose Up

CREATE TABLE distributions (
    -- Version-owned distribution identity
    id                       uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    dataset_version_id       uuid NOT NULL REFERENCES dataset_versions(id) ON DELETE CASCADE,

    -- Distribution metadata
    title                    nonblank_text NOT NULL,
    description              nonblank_text NOT NULL,
    license                  nonblank_text_opt,
    source_issued_at         timestamptz,
    source_modified_at       timestamptz,
    byte_size                integer,

    -- Delivery method. Constraint below requires one complete representation.
    type                     text NOT NULL,
    downloadable_file_url    nonblank_text_opt,
    downloadable_file_format nonblank_text_opt,
    web_service_id           uuid REFERENCES web_services(id) ON DELETE CASCADE,

    CHECK (
        (type = 'WEB_SERVICE'
            AND downloadable_file_url IS NULL
            AND downloadable_file_format IS NULL
            AND web_service_id IS NOT NULL)
        OR
        (type = 'DOWNLOADABLE_FILE'
            AND downloadable_file_url IS NOT NULL
            AND downloadable_file_format IS NOT NULL
            AND web_service_id IS NULL)
    )
);

-- +goose Down

DROP TABLE IF EXISTS distributions;
