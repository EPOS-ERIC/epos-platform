-- +goose Up

CREATE TABLE IF NOT EXISTS retrievals (
  id BIGSERIAL PRIMARY KEY,
  source_id TEXT NOT NULL,
  method TEXT NOT NULL,
  url_template TEXT NOT NULL,
  generated_url TEXT NOT NULL,
  path_params_json JSONB NOT NULL,
  query_params_json JSONB NOT NULL,
  headers_json JSONB NOT NULL,
  request_body_text TEXT,
  request_body_hash TEXT NOT NULL,
  request_body_is_json BOOLEAN NOT NULL DEFAULT FALSE,
  http_status INTEGER NOT NULL,
  content_type TEXT NOT NULL,
  is_binary BOOLEAN NOT NULL,
  payload_text TEXT,
  payload_hash TEXT NOT NULL,
  payload_size_bytes INTEGER NOT NULL,
  used_auth BOOLEAN NOT NULL DEFAULT FALSE,
  fetched_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  search_vector tsvector GENERATED ALWAYS AS (to_tsvector('simple', COALESCE(payload_text, ''))) STORED
);

CREATE INDEX IF NOT EXISTS retrievals_source_id_idx ON retrievals (source_id);
CREATE INDEX IF NOT EXISTS retrievals_generated_url_idx ON retrievals (generated_url);
CREATE UNIQUE INDEX IF NOT EXISTS retrievals_dedupe_idx ON retrievals (source_id, method, generated_url, request_body_hash);

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM pg_attribute
    WHERE attrelid = 'retrievals'::regclass
      AND attname = 'search_vector'
      AND NOT attisdropped
  ) THEN
    CREATE INDEX IF NOT EXISTS retrievals_search_vector_idx ON retrievals USING GIN (search_vector);
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down

DROP TABLE IF EXISTS retrievals;
