-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    auth_identifier TEXT NOT NULL UNIQUE,
    family_name TEXT,
    given_name TEXT,
    email TEXT UNIQUE,
    is_super_admin BOOLEAN NOT NULL DEFAULT false,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_status_check CHECK (status IN ('ACTIVE', 'INACTIVE', 'BLOCKED'))
);

-- First user becomes super admin so service bootstraps with one trusted account.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION promote_first_user_as_super_admin()
RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM users) THEN
        NEW.is_super_admin := TRUE;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER users_promote_first_user_as_super_admin
BEFORE INSERT ON users
FOR EACH ROW
EXECUTE FUNCTION promote_first_user_as_super_admin();

CREATE TABLE groups (
    id uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE group_memberships (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, group_id),
    CONSTRAINT group_memberships_role_check CHECK (role IN ('VIEWER', 'EDITOR', 'REVIEWER', 'ADMIN')),
    CONSTRAINT group_memberships_status_check CHECK (status IN ('PENDING', 'ACTIVE', 'REVOKED', 'REJECTED'))
);

CREATE TABLE group_permissions (
    id uuid PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    group_id uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    service TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    effect TEXT NOT NULL DEFAULT 'ALLOW',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT group_permissions_effect_check CHECK (effect IN ('ALLOW', 'DENY'))
);

CREATE UNIQUE INDEX group_permissions_scope_uq
    ON group_permissions (group_id, service, action, coalesce(resource_type, ''), coalesce(resource_id, ''), effect);

CREATE INDEX users_auth_identifier_idx ON users (auth_identifier);
CREATE INDEX group_memberships_group_status_idx ON group_memberships (group_id, status);
CREATE INDEX group_permissions_service_action_idx ON group_permissions (service, action);

CREATE TABLE friendships (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, friend_id),
    CONSTRAINT friendships_self_check CHECK (user_id <> friend_id)
);

CREATE INDEX friendships_friend_idx ON friendships (friend_id);

INSERT INTO groups (slug, name, description, is_system)
VALUES ('all', 'ALL', 'Basic group', true)
ON CONFLICT (slug) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS group_permissions;
DROP TABLE IF EXISTS group_memberships;
DROP TABLE IF EXISTS friendships;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS users;
