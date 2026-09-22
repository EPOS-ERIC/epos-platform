-- +goose Up

CREATE TABLE IF NOT EXISTS configurations (
  id UUID PRIMARY KEY,
  configuration JSONB NOT NULL,
  title TEXT,
  description TEXT,
  list_of_services JSONB NOT NULL DEFAULT '[]',
  is_official BOOLEAN NOT NULL DEFAULT FALSE,
  is_public BOOLEAN NOT NULL DEFAULT TRUE,
  owner_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT configurations_official_public CHECK (NOT is_official OR is_public)
);

CREATE INDEX IF NOT EXISTS configurations_updated_at_idx ON configurations (updated_at DESC);
CREATE INDEX IF NOT EXISTS configurations_owner_id_idx ON configurations (owner_id);
CREATE INDEX IF NOT EXISTS configurations_official_idx ON configurations (is_official) WHERE is_official;
CREATE INDEX IF NOT EXISTS configurations_configuration_idx ON configurations USING GIN (configuration jsonb_path_ops);

-- +goose Down

DROP TABLE IF EXISTS configurations;
