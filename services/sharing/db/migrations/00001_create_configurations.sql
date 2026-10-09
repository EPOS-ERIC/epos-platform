-- +goose Up

CREATE DOMAIN nonblank_text AS text
    CHECK (length(btrim(VALUE)) > 0);

CREATE DOMAIN nonblank_text_opt AS text
    CHECK (VALUE IS NULL OR length(btrim(VALUE)) > 0);

CREATE TABLE IF NOT EXISTS configurations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  configuration JSONB NOT NULL,
  title nonblank_text NOT NULL CHECK (char_length(title) <= 500),
  description nonblank_text_opt,
  list_of_services nonblank_text[] NOT NULL DEFAULT '{}' CHECK (array_position(list_of_services, NULL::nonblank_text) IS NULL),
  is_official BOOLEAN NOT NULL DEFAULT FALSE,
  is_public BOOLEAN NOT NULL DEFAULT FALSE,
  owner_id nonblank_text NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT configurations_official_public CHECK (NOT is_official OR is_public)
);

CREATE INDEX IF NOT EXISTS configurations_updated_at_idx ON configurations (updated_at DESC);
CREATE INDEX IF NOT EXISTS configurations_owner_id_idx ON configurations (owner_id);
CREATE INDEX IF NOT EXISTS configurations_official_idx ON configurations (is_official) WHERE is_official;
-- +goose Down

DROP TABLE IF EXISTS configurations;
DROP DOMAIN IF EXISTS nonblank_text_opt, nonblank_text;
