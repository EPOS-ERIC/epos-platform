-- +goose Up

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE DOMAIN nonblank_text AS text
    CHECK (length(btrim(VALUE)) > 0);

CREATE DOMAIN nonblank_text_opt AS text
    CHECK (VALUE IS NULL OR length(btrim(VALUE)) > 0);

CREATE DOMAIN positive_int AS integer
    CHECK (VALUE > 0);

CREATE TYPE user_status AS ENUM ('ACTIVE', 'BLOCKED');
CREATE TYPE membership_status AS ENUM ('PENDING', 'ACTIVE', 'REJECTED');
CREATE TYPE membership_role AS ENUM ('EDITOR', 'REVIEWER', 'GROUP_ADMIN');

CREATE TABLE concepts (
    id   uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name nonblank_text NOT NULL UNIQUE
);

INSERT INTO concepts (name) VALUES ('Data'), ('Facilities'), ('Software');

CREATE TABLE users (
    id               text PRIMARY KEY,
    family_name      nonblank_text NOT NULL,
    given_name       nonblank_text NOT NULL,
    email            nonblank_text NOT NULL UNIQUE,
    status           user_status NOT NULL DEFAULT 'ACTIVE',
    block_reason     nonblank_text_opt,
    blocked_by       text REFERENCES users(id) ON DELETE RESTRICT,
    blocked_at       timestamptz,
    api_token_digest bytea UNIQUE,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'BLOCKED') = (block_reason IS NOT NULL AND blocked_by IS NOT NULL AND blocked_at IS NOT NULL))
);

CREATE TABLE groups (
    -- Group identity and authorization boundary
    id          uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        nonblank_text NOT NULL UNIQUE,
    description nonblank_text_opt,

    -- Group classification and display metadata
    concept_id uuid NOT NULL REFERENCES concepts(id) ON DELETE RESTRICT,
    label       nonblank_text_opt,
    logo        nonblank_text_opt,
    page        nonblank_text_opt,
    color       nonblank_text_opt,
    position    integer NOT NULL DEFAULT 0,

    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE IF EXISTS groups, users, concepts;
DROP DOMAIN IF EXISTS positive_int, nonblank_text_opt, nonblank_text;
DROP TYPE IF EXISTS membership_role, membership_status, user_status;
