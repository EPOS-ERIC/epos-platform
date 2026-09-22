-- +goose Up

ALTER TABLE retrievals ALTER COLUMN payload_size_bytes TYPE BIGINT;

-- +goose Down

ALTER TABLE retrievals ALTER COLUMN payload_size_bytes TYPE INTEGER;
