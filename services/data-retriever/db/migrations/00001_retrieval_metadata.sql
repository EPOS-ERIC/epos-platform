-- +goose Up

CREATE TABLE IF NOT EXISTS retrievals (
  id                   BIGSERIAL PRIMARY KEY,
  source_id            TEXT,
  resource_id          UUID,
  version_id           UUID,
  method               TEXT NOT NULL,
  url_template         TEXT NOT NULL,
  generated_url        TEXT NOT NULL,
  path_params_json     JSONB NOT NULL,
  query_params_json    JSONB NOT NULL,
  headers_json         JSONB NOT NULL,
  request_body_text    TEXT,
  request_body_hash    TEXT NOT NULL,
  request_body_is_json BOOLEAN NOT NULL DEFAULT FALSE,
  http_status          INTEGER NOT NULL,
  content_type         TEXT NOT NULL,
  is_binary            BOOLEAN NOT NULL,
  payload_text         TEXT,
  payload_hash         TEXT NOT NULL,
  payload_size_bytes   BIGINT NOT NULL,
  used_auth            BOOLEAN NOT NULL DEFAULT FALSE,
  fetched_at           TIMESTAMPTZ NOT NULL,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  search_vector        TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', COALESCE(payload_text, ''))) STORED
);

ALTER TABLE retrievals ADD COLUMN IF NOT EXISTS source_id TEXT;
ALTER TABLE retrievals ALTER COLUMN source_id DROP NOT NULL;
ALTER TABLE retrievals ADD COLUMN IF NOT EXISTS resource_id UUID;
ALTER TABLE retrievals ADD COLUMN IF NOT EXISTS version_id UUID;
ALTER TABLE retrievals ADD COLUMN IF NOT EXISTS payload_text TEXT;
ALTER TABLE retrievals ADD COLUMN IF NOT EXISTS search_vector TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', COALESCE(payload_text, ''))) STORED;
ALTER TABLE retrievals ALTER COLUMN payload_size_bytes TYPE BIGINT;

CREATE INDEX IF NOT EXISTS retrievals_source_id_idx ON retrievals (source_id);
CREATE INDEX IF NOT EXISTS retrievals_generated_url_idx ON retrievals (generated_url);
CREATE UNIQUE INDEX IF NOT EXISTS retrievals_dedupe_idx ON retrievals (source_id, method, generated_url, request_body_hash);
CREATE INDEX IF NOT EXISTS retrievals_search_vector_idx ON retrievals USING GIN (search_vector);
CREATE UNIQUE INDEX IF NOT EXISTS retrievals_resource_version_dedupe_idx
  ON retrievals (resource_id, version_id, method, generated_url, request_body_hash);

CREATE TABLE IF NOT EXISTS retrieval_output_schemas (
  resource_id UUID NOT NULL,
  version_id  UUID NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (resource_id, version_id)
);

CREATE TABLE IF NOT EXISTS retrieval_output_parameters (
  resource_id UUID NOT NULL,
  version_id  UUID NOT NULL,
  name        TEXT NOT NULL,
  kind        TEXT NOT NULL,
  PRIMARY KEY (resource_id, version_id, name),
  FOREIGN KEY (resource_id, version_id)
    REFERENCES retrieval_output_schemas (resource_id, version_id)
    ON DELETE CASCADE
);

-- +goose Down

-- Intentionally retain schema and data: this migration upgrades databases that
-- may already contain retrieval records from the legacy migration runner.
SELECT 1;
