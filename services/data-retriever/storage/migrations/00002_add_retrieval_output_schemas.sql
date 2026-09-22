-- +goose Up

ALTER TABLE retrievals
  ADD COLUMN IF NOT EXISTS resource_id UUID,
  ADD COLUMN IF NOT EXISTS version_id UUID;

-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'retrievals_resource_version_pair_check'
  ) THEN
    ALTER TABLE retrievals
      ADD CONSTRAINT retrievals_resource_version_pair_check
      CHECK ((resource_id IS NULL) = (version_id IS NULL));
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS retrievals_dedupe_idx;
CREATE UNIQUE INDEX retrievals_dedupe_idx
  ON retrievals (resource_id, version_id, source_id, method, generated_url, request_body_hash);

DROP INDEX IF EXISTS retrievals_search_vector_idx;
ALTER TABLE retrievals DROP COLUMN IF EXISTS search_vector;
ALTER TABLE retrievals DROP COLUMN IF EXISTS payload_text;

CREATE TABLE IF NOT EXISTS retrieval_output_schemas (
  resource_id UUID NOT NULL,
  version_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (resource_id, version_id)
);

CREATE TABLE IF NOT EXISTS retrieval_output_parameters (
  resource_id UUID NOT NULL,
  version_id UUID NOT NULL,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  pattern TEXT,
  PRIMARY KEY (resource_id, version_id, name),
  FOREIGN KEY (resource_id, version_id)
    REFERENCES retrieval_output_schemas (resource_id, version_id)
    ON DELETE CASCADE
);

-- +goose Down

DROP TABLE IF EXISTS retrieval_output_parameters;
DROP TABLE IF EXISTS retrieval_output_schemas;
ALTER TABLE retrievals DROP COLUMN IF EXISTS version_id;
ALTER TABLE retrievals DROP COLUMN IF EXISTS resource_id;
