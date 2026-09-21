-- +goose Up

CREATE TYPE permission_scope AS ENUM ('GLOBAL', 'GROUP');
CREATE TYPE role_request_status AS ENUM ('PENDING', 'ACCEPTED', 'REJECTED');
CREATE TYPE notification_scope AS ENUM ('GLOBAL', 'GROUP');
CREATE TYPE notification_channel AS ENUM ('EMAIL');

-- Authorization catalog is fixed migration data, not user-managed configuration.
CREATE TABLE permissions (
    key         text PRIMARY KEY,
    label       text NOT NULL,
    description text NOT NULL,
    scope       permission_scope NOT NULL
);
CREATE TABLE roles (
    key   text PRIMARY KEY,
    scope permission_scope NOT NULL
);
CREATE TABLE role_permissions (
    role_key       text NOT NULL REFERENCES roles(key) ON DELETE CASCADE,
    permission_key text NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
    PRIMARY KEY (role_key, permission_key)
);

CREATE TABLE group_memberships (
    user_id    text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id   uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    status     membership_status NOT NULL DEFAULT 'PENDING',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, group_id)
);
CREATE TABLE group_membership_roles (
    user_id     text NOT NULL,
    group_id    uuid NOT NULL,
    role_key    text NOT NULL REFERENCES roles(key) ON DELETE RESTRICT,
    assigned_by text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    assigned_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, group_id, role_key),
    FOREIGN KEY (user_id, group_id)
        REFERENCES group_memberships(user_id, group_id)
        ON DELETE CASCADE
);
CREATE TABLE user_global_roles (
    user_id     text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_key    text NOT NULL REFERENCES roles(key) ON DELETE RESTRICT,
    assigned_by text REFERENCES users(id) ON DELETE RESTRICT,
    assigned_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_key)
);
CREATE TABLE global_permission_grants (
    user_id        text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_key text NOT NULL REFERENCES permissions(key) ON DELETE RESTRICT,
    granted_by     text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    granted_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, permission_key)
);
CREATE TABLE group_permission_grants (
    user_id        text NOT NULL,
    group_id       uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    permission_key text NOT NULL REFERENCES permissions(key) ON DELETE RESTRICT,
    granted_by     text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    granted_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, group_id, permission_key),
    FOREIGN KEY (user_id, group_id)
        REFERENCES group_memberships(user_id, group_id)
        ON DELETE CASCADE
);
CREATE TABLE group_role_requests (
    id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      text NOT NULL,
    group_id     uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    role_key     text NOT NULL REFERENCES roles(key) ON DELETE RESTRICT CHECK (role_key IN ('EDITOR', 'REVIEWER')),
    status       role_request_status NOT NULL DEFAULT 'PENDING',
    requested_at timestamptz NOT NULL DEFAULT now(),
    decided_by   text REFERENCES users(id) ON DELETE RESTRICT,
    decided_at   timestamptz,
    FOREIGN KEY (user_id, group_id)
        REFERENCES group_memberships(user_id, group_id)
        ON DELETE CASCADE
);
CREATE UNIQUE INDEX group_role_requests_pending_uq ON group_role_requests (user_id, group_id, role_key) WHERE status = 'PENDING';

CREATE TABLE notification_event_types (
    key                     text PRIMARY KEY,
    scope                   notification_scope NOT NULL,
    required_permission_key text REFERENCES permissions(key) ON DELETE RESTRICT
);
CREATE TABLE global_notification_subscriptions (
    user_id    text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type text NOT NULL REFERENCES notification_event_types(key) ON DELETE CASCADE,
    channel    notification_channel NOT NULL DEFAULT 'EMAIL',
    PRIMARY KEY (user_id, event_type, channel)
);
CREATE TABLE group_notification_subscriptions (
    user_id    text NOT NULL,
    group_id   uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    event_type text NOT NULL REFERENCES notification_event_types(key) ON DELETE CASCADE,
    channel    notification_channel NOT NULL DEFAULT 'EMAIL',
    PRIMARY KEY (user_id, group_id, event_type, channel),
    FOREIGN KEY (user_id, group_id)
        REFERENCES group_memberships(user_id, group_id)
        ON DELETE CASCADE
);

-- Keep this list synchronized with permission enums in docs/openapi.yaml for
-- Metadata Lifecycle and Gateway, then regenerate the Gateway Metadata Lifecycle client.
INSERT INTO permissions (key, label, description, scope) VALUES
    ('group.resource.create',     'Create resources',         'Create an empty draft, draft from an existing resource, or copy a published resource into this group.',     'GROUP'),
    ('group.resource.edit',       'Edit drafts',              'Change metadata and links on draft resources owned by this group.',                                         'GROUP'),
    ('group.resource.submit',     'Submit drafts',            'Submit this group''s draft resources for review. Submitted versions become immutable.',                     'GROUP'),
    ('group.resource.withdraw',   'Withdraw submissions',     'Return this group''s submitted resource versions to editable draft status.',                                'GROUP'),
    ('group.resource.reopen',     'Reopen rejections',        'Return this group''s rejected resource versions to editable draft status.',                                 'GROUP'),
    ('group.resource.discard',    'Discard changes',          'Permanently delete this group''s draft or rejected resource versions.',                                     'GROUP'),
    ('group.resource.reject',     'Reject submissions',       'Reject submitted resource versions in this group and provide a rejection note.',                            'GROUP'),
    ('group.resource.publish',    'Publish resources',        'Publish submitted resource versions in this group. Existing publications are archived.',                    'GROUP'),
    ('group.resource.unpublish',  'Unpublish resources',      'Archive this group''s currently published resource version without publishing a replacement.',              'GROUP'),
    ('group.membership.manage',   'Manage memberships',       'Accept, reject, or revoke membership requests for this group.',                                             'GROUP'),
    ('group.role.manage',         'Manage group roles',       'Assign or remove EDITOR and REVIEWER roles for active members of this group. Does not manage GROUP_ADMIN.', 'GROUP'),
    ('group.permission.manage',   'Manage group permissions', 'Grant or revoke group-scoped direct permissions for active members of this group.',                         'GROUP'),
    ('group.category.manage',     'Manage categories',        'Create, edit, reparent, and delete canonical categories in this group.',                                    'GROUP'),
    ('global.membership.manage',  'Manage memberships',       'List users and accept, reject, or revoke memberships in every group.',                                      'GLOBAL'),
    ('global.role.manage',        'Manage group roles',       'List users and assign or remove group roles in every group, including GROUP_ADMIN.',                        'GLOBAL'),
    ('global.permission.manage',  'Manage permissions',       'List users and grant or revoke global permissions.',                                                        'GLOBAL'),
    ('global.organization.manage',    'Manage organizations',     'Create, edit, delete, and manage identifiers and hierarchy for canonical organizations across all groups.', 'GLOBAL'),
    ('global.contact_point.manage',   'Manage contact points',    'Create, edit, and delete canonical contact points across all groups.',                                      'GLOBAL'),
    ('global.category_scheme.manage', 'Manage category schemes',  'Edit category-scheme metadata stored on groups across the system. Does not manage categories.',             'GLOBAL'),
    ('global.resource.create',    'Create resources',         'Create an empty draft, draft from an existing resource, or copy a published resource into any group.',      'GLOBAL'),
    ('global.resource.edit',      'Edit drafts',              'Change metadata and links on draft resources owned by any group.',                                          'GLOBAL'),
    ('global.resource.submit',    'Submit drafts',            'Submit draft resources from any group for review. Submitted versions become immutable.',                    'GLOBAL'),
    ('global.resource.withdraw',  'Withdraw submissions',     'Return submitted resource versions from any group to editable draft status.',                               'GLOBAL'),
    ('global.resource.reopen',    'Reopen rejections',        'Return rejected resource versions from any group to editable draft status.',                                'GLOBAL'),
    ('global.resource.discard',   'Discard changes',          'Permanently delete draft or rejected resource versions from any group.',                                    'GLOBAL'),
    ('global.resource.reject',    'Reject submissions',       'Reject submitted resource versions from any group and provide a rejection note.',                           'GLOBAL'),
    ('global.resource.publish',   'Publish resources',        'Publish submitted resource versions from any group. Existing publications are archived.',                   'GLOBAL'),
    ('global.resource.unpublish', 'Unpublish resources',      'Archive currently published resource versions from any group without publishing a replacement.',            'GLOBAL'),
    ('global.category.manage',    'Manage categories',        'Create, edit, reparent, and delete canonical categories in any group.',                                     'GLOBAL');
INSERT INTO roles (key, scope) VALUES
    ('EDITOR',      'GROUP'),
    ('REVIEWER',    'GROUP'),
    ('GROUP_ADMIN', 'GROUP'),
    ('ADMIN',       'GLOBAL');
INSERT INTO role_permissions (role_key, permission_key) VALUES
    ('EDITOR',      'group.resource.create'),
    ('EDITOR',      'group.resource.edit'),
    ('EDITOR',      'group.resource.submit'),
    ('EDITOR',      'group.resource.withdraw'),
    ('EDITOR',      'group.resource.reopen'),
    ('EDITOR',      'group.resource.discard'),
    ('REVIEWER',    'group.resource.reject'),
    ('REVIEWER',    'group.resource.publish'),
    ('GROUP_ADMIN', 'group.resource.create'),
    ('GROUP_ADMIN', 'group.resource.edit'),
    ('GROUP_ADMIN', 'group.resource.submit'),
    ('GROUP_ADMIN', 'group.resource.withdraw'),
    ('GROUP_ADMIN', 'group.resource.reopen'),
    ('GROUP_ADMIN', 'group.resource.discard'),
    ('GROUP_ADMIN', 'group.resource.reject'),
    ('GROUP_ADMIN', 'group.resource.publish'),
    ('GROUP_ADMIN', 'group.resource.unpublish'),
    ('GROUP_ADMIN', 'group.membership.manage'),
    ('GROUP_ADMIN', 'group.role.manage'),
    ('GROUP_ADMIN', 'group.permission.manage'),
    ('ADMIN',       'global.membership.manage'),
    ('ADMIN',       'global.role.manage'),
    ('ADMIN',       'global.permission.manage'),
    ('ADMIN',       'global.organization.manage'),
    ('ADMIN',       'global.contact_point.manage'),
    ('ADMIN',       'global.category_scheme.manage'),
    ('ADMIN',       'global.resource.create'),
    ('ADMIN',       'global.resource.edit'),
    ('ADMIN',       'global.resource.submit'),
    ('ADMIN',       'global.resource.withdraw'),
    ('ADMIN',       'global.resource.reopen'),
    ('ADMIN',       'global.resource.discard'),
    ('ADMIN',       'global.resource.reject'),
    ('ADMIN',       'global.resource.publish'),
    ('ADMIN',       'global.resource.unpublish'),
    ('ADMIN',       'global.category.manage');

-- +goose Down

DROP TABLE IF EXISTS group_notification_subscriptions, global_notification_subscriptions, notification_event_types, group_role_requests, group_permission_grants, global_permission_grants, user_global_roles, group_membership_roles, group_memberships, role_permissions, roles, permissions;
DROP TYPE IF EXISTS notification_channel, notification_scope, role_request_status, permission_scope;
