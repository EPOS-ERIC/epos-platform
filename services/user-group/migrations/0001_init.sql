BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    auth_identifier text NOT NULL UNIQUE,
    family_name text,
    given_name text,
    email text UNIQUE,
    status text NOT NULL DEFAULT 'ACTIVE',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_status_check CHECK (status IN ('ACTIVE', 'INACTIVE', 'BLOCKED'))
);

CREATE TABLE IF NOT EXISTS groups (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug text NOT NULL UNIQUE,
    name text NOT NULL,
    description text,
    is_system boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS group_memberships (
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    group_id uuid NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    role text NOT NULL,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, group_id),
    CONSTRAINT group_memberships_role_check CHECK (role IN ('VIEWER', 'EDITOR', 'REVIEWER', 'ADMIN')),
    CONSTRAINT group_memberships_status_check CHECK (status IN ('PENDING', 'ACTIVE', 'REVOKED', 'REJECTED'))
);

CREATE TABLE IF NOT EXISTS group_permissions (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id uuid NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    service text NOT NULL,
    action text NOT NULL,
    resource_type text,
    resource_id text,
    effect text NOT NULL DEFAULT 'ALLOW',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT group_permissions_effect_check CHECK (effect IN ('ALLOW', 'DENY'))
);

CREATE UNIQUE INDEX IF NOT EXISTS group_permissions_scope_uq
    ON group_permissions (group_id, service, action, coalesce(resource_type, ''), coalesce(resource_id, ''), effect);

CREATE INDEX IF NOT EXISTS users_auth_identifier_idx ON users (auth_identifier);
CREATE INDEX IF NOT EXISTS group_memberships_group_status_idx ON group_memberships (group_id, status);
CREATE INDEX IF NOT EXISTS group_permissions_service_action_idx ON group_permissions (service, action);

INSERT INTO groups (slug, name, description, is_system)
VALUES ('all', 'ALL', 'Basic group', true)
ON CONFLICT (slug) DO NOTHING;

COMMIT;
