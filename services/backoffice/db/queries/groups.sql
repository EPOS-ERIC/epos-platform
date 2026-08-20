-- name: ListGroups :many
SELECT
    sqlc.embed(groups)
FROM
    groups
ORDER BY
    position,
    name,
    id;

-- name: GetGroupByID :one
SELECT
    sqlc.embed(groups)
FROM
    groups
WHERE
    id = $1;

-- name: ListGroupMemberships :many
SELECT
    sqlc.embed(membership),
    sqlc.embed(users)
FROM
    group_memberships AS membership
    JOIN users ON users.id = membership.user_id
WHERE
    membership.group_id = $1
ORDER BY
    users.family_name,
    users.given_name,
    membership.user_id;

-- name: ListGroupMembershipRoles :many
SELECT
    sqlc.embed(group_membership_roles)
FROM
    group_membership_roles
WHERE
    group_id = $1
ORDER BY
    user_id,
    role_key;

-- name: ListGroupMembershipPermissions :many
SELECT
    sqlc.embed(group_permission_grants)
FROM
    group_permission_grants
WHERE
    group_id = $1
ORDER BY
    user_id,
    permission_key;

-- name: ListPendingGroupRoleRequests :many
SELECT
    sqlc.embed(group_role_requests)
FROM
    group_role_requests
WHERE
    group_id = $1
    AND STATUS = 'PENDING'
ORDER BY
    user_id,
    role_key,
    id;

-- name: RequestGroupMembership :one
INSERT INTO
    group_memberships (user_id, group_id, STATUS)
VALUES
    ($1, $2, 'PENDING') ON CONFLICT (user_id, group_id) DO
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
VALUES
    ($1, $2, $3)
RETURNING
    id,
    user_id,
    group_id,
    role_key,
    STATUS,
    requested_at;
