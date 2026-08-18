-- +goose Up

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE user_status AS ENUM ('ACTIVE', 'INACTIVE', 'BLOCKED');
CREATE TYPE membership_status AS ENUM ('PENDING', 'ACTIVE', 'REJECTED');
CREATE TYPE membership_role AS ENUM ('EDITOR', 'REVIEWER', 'GROUP_ADMIN');
CREATE TYPE top_level_concept AS ENUM ('DATA', 'FACILITIES', 'SOFTWARE');

CREATE TABLE users (
    id              uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    auth_identifier text NOT NULL UNIQUE,
    family_name     text NOT NULL,
    given_name      text NOT NULL,
    email           text NOT NULL UNIQUE,
    status          user_status NOT NULL DEFAULT 'ACTIVE',
    block_reason    text,
    blocked_by      uuid REFERENCES users(id) ON DELETE RESTRICT,
    blocked_at      timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'BLOCKED') = (block_reason IS NOT NULL AND blocked_by IS NOT NULL AND blocked_at IS NOT NULL)),
    CHECK (block_reason IS NULL OR length(trim(block_reason)) > 0)
);

CREATE TABLE groups (
    -- Group identity and authorization boundary
    id          uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        text NOT NULL UNIQUE,
    description text,

    -- Category scheme metadata: groups own exactly one scheme
    source_iri  text NOT NULL UNIQUE,
    concept     top_level_concept NOT NULL,
    label       text,
    logo        text,
    page        text,
    color       text,
    position    integer NOT NULL DEFAULT 0,

    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE IF EXISTS groups, users;
DROP TYPE IF EXISTS top_level_concept, membership_role, membership_status, user_status;
