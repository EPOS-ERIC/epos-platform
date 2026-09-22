-- +goose Up

DROP INDEX IF EXISTS retrievals_dedupe_idx;
DROP INDEX IF EXISTS retrievals_source_id_idx;
ALTER TABLE retrievals DROP COLUMN IF EXISTS source_id;
CREATE UNIQUE INDEX retrievals_dedupe_idx
  ON retrievals (resource_id, version_id, method, generated_url, request_body_hash);

-- +goose Down

DROP INDEX IF EXISTS retrievals_dedupe_idx;
ALTER TABLE retrievals ADD COLUMN IF NOT EXISTS source_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS retrievals_source_id_idx ON retrievals (source_id);
CREATE UNIQUE INDEX retrievals_dedupe_idx
  ON retrievals (resource_id, version_id, source_id, method, generated_url, request_body_hash);
