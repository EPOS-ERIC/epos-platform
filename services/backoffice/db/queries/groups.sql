-- TODO: we sholud probably split this huge file in separate files
-- name: ListGroups :many
SELECT
    id,
    name,
    description,
    source_iri,
    concept,
    label,
    logo,
    PAGE,
    color,
    position,
    created_at,
    updated_at
FROM
    groups
ORDER BY
    position,
    name,
    id;

-- name: GetGroupByID :one
SELECT
    id,
    name,
    description,
    source_iri,
    concept,
    label,
    logo,
    PAGE,
    color,
    position,
    created_at,
    updated_at
FROM
    groups
WHERE
    id = sqlc.arg(id);

-- name: CreateGroup :one
INSERT INTO
    groups (
        id,
        name,
        description,
        source_iri,
        concept,
        label,
        logo,
        PAGE,
        color,
        position
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(name),
        sqlc.arg(description),
        sqlc.arg(source_iri),
        sqlc.arg(concept),
        sqlc.arg(label),
        sqlc.arg(logo),
        sqlc.arg(PAGE),
        sqlc.arg(color),
        sqlc.arg(position)
    )
RETURNING
    id,
    name,
    description,
    source_iri,
    concept,
    label,
    logo,
    PAGE,
    color,
    position,
    created_at,
    updated_at;

-- name: ReplaceGroup :one
UPDATE
    groups
SET
    name = sqlc.arg(name),
    description = sqlc.arg(description),
    label = sqlc.arg(label),
    logo = sqlc.arg(logo),
    PAGE = sqlc.arg(PAGE),
    color = sqlc.arg(color),
    position = sqlc.arg(position),
    concept = sqlc.arg(concept)
WHERE
    id = sqlc.arg(id)
RETURNING
    id,
    name,
    description,
    source_iri,
    concept,
    label,
    logo,
    PAGE,
    color,
    position,
    created_at,
    updated_at;

-- name: DeleteGroup :one
WITH target AS (
    SELECT
        group_row.id
    FROM
        groups AS group_row
    WHERE
        group_row.id = sqlc.arg(id)
),
deleted AS (
    DELETE FROM
        groups AS group_row
    WHERE
        group_row.id = (
            SELECT
                target.id
            FROM
                target
        )
        AND NOT EXISTS (
            SELECT
                1
            FROM
                datasets
            WHERE
                group_id = sqlc.arg(id)
            UNION
            ALL
            SELECT
                1
            FROM
                software_applications
            WHERE
                group_id = sqlc.arg(id)
            UNION
            ALL
            SELECT
                1
            FROM
                software_source_codes
            WHERE
                group_id = sqlc.arg(id)
        )
    RETURNING
        id
)
SELECT
    CASE
        WHEN EXISTS (
            SELECT
                1
            FROM
                deleted
        ) THEN 'DELETED'
        WHEN NOT EXISTS (
            SELECT
                1
            FROM
                target
        ) THEN 'NOT_FOUND'
        ELSE 'CONFLICT'
    END AS outcome;

-- name: ListGroupCategories :many
SELECT
    id,
    group_id,
    parent_category_id,
    source_iri,
    name,
    created_at,
    updated_at
FROM
    categories
WHERE
    group_id = sqlc.arg(group_id)
ORDER BY
    parent_category_id NULLS FIRST,
    lower(trim(name)),
    id;

-- name: GetGroupCategory :one
SELECT
    id,
    group_id,
    parent_category_id,
    source_iri,
    name,
    created_at,
    updated_at
FROM
    categories
WHERE
    group_id = sqlc.arg(group_id)
    AND id = sqlc.arg(id);

-- name: CreateGroupCategory :one
INSERT INTO
    categories (
        id,
        group_id,
        parent_category_id,
        source_iri,
        name
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(group_id),
        sqlc.narg(parent_category_id),
        sqlc.arg(source_iri),
        sqlc.arg(name)
    )
RETURNING
    id,
    group_id,
    parent_category_id,
    source_iri,
    name,
    created_at,
    updated_at;

-- name: ListGroupMemberships :many
SELECT
    sqlc.embed(membership),
    sqlc.embed(users)
FROM
    group_memberships AS membership
    JOIN users ON users.id = membership.user_id
WHERE
    membership.group_id = sqlc.arg(group_id)
ORDER BY
    users.family_name,
    users.given_name,
    membership.user_id;

-- name: GetGroupMembership :one
SELECT
    sqlc.embed(membership),
    sqlc.embed(users)
FROM
    group_memberships AS membership
    JOIN users ON users.id = membership.user_id
WHERE
    membership.group_id = sqlc.arg(group_id)
    AND membership.user_id = sqlc.arg(user_id);

-- name: ListGroupMembershipRoles :many
SELECT
    user_id,
    group_id,
    role_key,
    assigned_by,
    assigned_at
FROM
    group_membership_roles
WHERE
    group_id = sqlc.arg(group_id)
ORDER BY
    user_id,
    role_key;

-- name: ListGroupMembershipPermissions :many
SELECT
    user_id,
    group_id,
    permission_key,
    granted_by,
    granted_at
FROM
    group_permission_grants
WHERE
    group_id = sqlc.arg(group_id)
ORDER BY
    user_id,
    permission_key;

-- name: ListPendingGroupRoleRequests :many
SELECT
    id,
    user_id,
    group_id,
    role_key,
    STATUS,
    requested_at,
    decided_by,
    decided_at
FROM
    group_role_requests
WHERE
    group_id = sqlc.arg(group_id)
    AND STATUS = 'PENDING'
ORDER BY
    user_id,
    role_key,
    id;

-- name: RequestGroupMembership :one
INSERT INTO
    group_memberships (user_id, group_id, STATUS)
VALUES
    (sqlc.arg(user_id), sqlc.arg(group_id), 'PENDING') ON CONFLICT (user_id, group_id) DO
UPDATE
SET
    STATUS = 'PENDING'
WHERE
    group_memberships.STATUS = 'REJECTED'
RETURNING
    user_id,
    group_id,
    STATUS,
    created_at,
    updated_at;

-- name: CreateGroupRoleRequest :one
INSERT INTO
    group_role_requests (user_id, group_id, role_key)
SELECT
    sqlc.arg(user_id),
    sqlc.arg(group_id),
    sqlc.arg(role_key)
FROM
    group_memberships
WHERE
    user_id = sqlc.arg(user_id)
    AND group_id = sqlc.arg(group_id)
    AND STATUS = 'ACTIVE'
    AND NOT EXISTS (
        SELECT
            1
        FROM
            group_membership_roles
        WHERE
            user_id = sqlc.arg(user_id)
            AND group_id = sqlc.arg(group_id)
            AND role_key = sqlc.arg(role_key)
    )
RETURNING
    id,
    user_id,
    group_id,
    role_key,
    STATUS,
    requested_at;

-- name: CreatePendingGroupRoleRequest :one
INSERT INTO
    group_role_requests (user_id, group_id, role_key)
SELECT
    sqlc.arg(user_id),
    sqlc.arg(group_id),
    sqlc.arg(role_key)
FROM
    group_memberships
WHERE
    user_id = sqlc.arg(user_id)
    AND group_id = sqlc.arg(group_id)
    AND STATUS = 'PENDING'
RETURNING
    id,
    user_id,
    group_id,
    role_key,
    STATUS,
    requested_at;

-- name: WithdrawGroupMembershipRequest :execrows
DELETE FROM
    group_memberships
WHERE
    group_id = sqlc.arg(group_id)
    AND user_id = sqlc.arg(user_id)
    AND STATUS = 'PENDING';

-- name: RevokeGroupMembership :execrows
DELETE FROM
    group_memberships
WHERE
    group_id = sqlc.arg(group_id)
    AND user_id = sqlc.arg(user_id);

-- name: DecideGroupMembership :one
UPDATE
    group_memberships
SET
    STATUS = sqlc.arg(STATUS)::membership_status
WHERE
    group_id = sqlc.arg(group_id)
    AND user_id = sqlc.arg(user_id)
    AND STATUS = 'PENDING'
RETURNING
    user_id,
    group_id,
    STATUS,
    created_at,
    updated_at;

-- name: RejectPendingGroupRoleRequests :exec
UPDATE
    group_role_requests
SET
    STATUS = 'REJECTED',
    decided_by = sqlc.arg(decided_by),
    decided_at = NOW()
WHERE
    group_id = sqlc.arg(group_id)
    AND user_id = sqlc.arg(user_id)
    AND STATUS = 'PENDING';

-- name: ApplyRoleRequestDecision :one
UPDATE
    group_role_requests
SET
    STATUS = sqlc.arg(STATUS)::role_request_status,
    decided_by = sqlc.arg(decided_by),
    decided_at = NOW()
WHERE
    group_role_requests.id = sqlc.arg(request_id)
    AND group_role_requests.group_id = sqlc.arg(group_id)
    AND group_role_requests.STATUS = 'PENDING'
    AND EXISTS (
        SELECT
            1
        FROM
            group_memberships
        WHERE
            user_id = group_role_requests.user_id
            AND group_id = group_role_requests.group_id
            AND STATUS = 'ACTIVE'
    )
RETURNING
    group_role_requests.id,
    group_role_requests.user_id,
    group_role_requests.group_id,
    group_role_requests.role_key,
    group_role_requests.STATUS,
    group_role_requests.requested_at,
    group_role_requests.decided_by,
    group_role_requests.decided_at;

-- name: AssignGroupRoleFromRequest :exec
INSERT INTO
    group_membership_roles (user_id, group_id, role_key, assigned_by)
SELECT
    group_role_requests.user_id,
    group_role_requests.group_id,
    group_role_requests.role_key,
    sqlc.arg(assigned_by)
FROM
    group_role_requests
    JOIN group_memberships ON group_memberships.user_id = group_role_requests.user_id
    AND group_memberships.group_id = group_role_requests.group_id
WHERE
    group_role_requests.id = sqlc.arg(request_id)
    AND group_role_requests.group_id = sqlc.arg(group_id)
    AND group_role_requests.STATUS = 'ACCEPTED'
    AND group_memberships.STATUS = 'ACTIVE' ON CONFLICT (user_id, group_id, role_key) DO NOTHING;

-- name: WithdrawGroupRoleRequest :execrows
DELETE FROM
    group_role_requests
WHERE
    id = sqlc.arg(request_id)
    AND group_id = sqlc.arg(group_id)
    AND user_id = sqlc.arg(user_id)
    AND STATUS = 'PENDING';

-- name: GroupMembershipRoleExists :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            group_membership_roles
        WHERE
            user_id = sqlc.arg(user_id)
            AND group_id = sqlc.arg(group_id)
            AND role_key = sqlc.arg(role_key)
    );

-- name: GetPendingGroupRoleRequest :one
SELECT
    id,
    user_id,
    group_id,
    role_key,
    STATUS,
    requested_at,
    decided_by,
    decided_at
FROM
    group_role_requests
WHERE
    user_id = sqlc.arg(user_id)
    AND group_id = sqlc.arg(group_id)
    AND role_key = sqlc.arg(role_key)
    AND STATUS = 'PENDING';

-- name: AssignGroupRole :execrows
INSERT INTO
    group_membership_roles (user_id, group_id, role_key, assigned_by)
SELECT
    sqlc.arg(user_id),
    sqlc.arg(group_id),
    sqlc.arg(role_key),
    sqlc.arg(assigned_by)
FROM
    group_memberships
WHERE
    user_id = sqlc.arg(user_id)
    AND group_id = sqlc.arg(group_id)
    AND STATUS = 'ACTIVE' ON CONFLICT (user_id, group_id, role_key) DO NOTHING;

-- name: RemoveGroupRole :execrows
DELETE FROM
    group_membership_roles
WHERE
    user_id = sqlc.arg(user_id)
    AND group_id = sqlc.arg(group_id)
    AND role_key = sqlc.arg(role_key);

-- name: GrantGroupPermission :exec
INSERT INTO
    group_permission_grants (user_id, group_id, permission_key, granted_by)
SELECT
    sqlc.arg(user_id),
    sqlc.arg(group_id),
    sqlc.arg(permission_key),
    sqlc.arg(granted_by)
FROM
    group_memberships
WHERE
    user_id = sqlc.arg(user_id)
    AND group_id = sqlc.arg(group_id)
    AND STATUS = 'ACTIVE' ON CONFLICT (user_id, group_id, permission_key) DO NOTHING;

-- name: RevokeGroupPermission :exec
DELETE FROM
    group_permission_grants
WHERE
    user_id = sqlc.arg(user_id)
    AND group_id = sqlc.arg(group_id)
    AND permission_key = sqlc.arg(permission_key);
