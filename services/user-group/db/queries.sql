-- name: ListUsers :many
SELECT
    id,
    auth_identifier,
    COALESCE(email, '') AS email,
    COALESCE(given_name, '') AS given_name,
    COALESCE(family_name, '') AS family_name,
    status,
    created_at,
    updated_at
FROM users
ORDER BY created_at DESC;

-- name: GetUserByID :one
SELECT
    id,
    auth_identifier,
    COALESCE(email, '') AS email,
    COALESCE(given_name, '') AS given_name,
    COALESCE(family_name, '') AS family_name,
    status,
    created_at,
    updated_at
FROM users
WHERE id = $1;

-- name: GetUserByAuthIdentifier :one
SELECT
    id,
    auth_identifier,
    COALESCE(email, '') AS email,
    COALESCE(given_name, '') AS given_name,
    COALESCE(family_name, '') AS family_name,
    status,
    created_at,
    updated_at
FROM users
WHERE auth_identifier = $1;

-- name: CreateUser :one
INSERT INTO users (auth_identifier, email, given_name, family_name, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING
    id,
    auth_identifier,
    COALESCE(email, '') AS email,
    COALESCE(given_name, '') AS given_name,
    COALESCE(family_name, '') AS family_name,
    status,
    created_at,
    updated_at;

-- name: UpdateUser :one
UPDATE users
SET
    auth_identifier = $1,
    email = $2,
    given_name = $3,
    family_name = $4,
    status = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $6
RETURNING
    id,
    auth_identifier,
    COALESCE(email, '') AS email,
    COALESCE(given_name, '') AS given_name,
    COALESCE(family_name, '') AS family_name,
    status,
    created_at,
    updated_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListGroups :many
SELECT
    id,
    slug,
    name,
    COALESCE(description, '') AS description,
    is_system,
    created_at,
    updated_at
FROM groups
ORDER BY created_at DESC;

-- name: GetGroupByID :one
SELECT
    id,
    slug,
    name,
    COALESCE(description, '') AS description,
    is_system,
    created_at,
    updated_at
FROM groups
WHERE id = $1;

-- name: CreateGroup :one
INSERT INTO groups (slug, name, description, is_system)
VALUES ($1, $2, $3, $4)
RETURNING
    id,
    slug,
    name,
    COALESCE(description, '') AS description,
    is_system,
    created_at,
    updated_at;

-- name: UpdateGroup :one
UPDATE groups
SET
    slug = $1,
    name = $2,
    description = $3,
    is_system = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $5
RETURNING
    id,
    slug,
    name,
    COALESCE(description, '') AS description,
    is_system,
    created_at,
    updated_at;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = $1;

-- name: ListMembershipsByGroupID :many
SELECT
    user_id,
    group_id,
    role,
    status,
    created_at,
    updated_at
FROM group_memberships
WHERE group_id = $1
ORDER BY created_at DESC;

-- name: GetMembershipByIDs :one
SELECT
    user_id,
    group_id,
    role,
    status,
    created_at,
    updated_at
FROM group_memberships
WHERE user_id = $1 AND group_id = $2;

-- name: UpsertMembership :one
INSERT INTO group_memberships (user_id, group_id, role, status)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, group_id) DO UPDATE SET
    role = excluded.role,
    status = excluded.status,
    updated_at = CURRENT_TIMESTAMP
RETURNING
    user_id,
    group_id,
    role,
    status,
    created_at,
    updated_at;

-- name: DeleteMembership :exec
DELETE FROM group_memberships WHERE user_id = $1 AND group_id = $2;

-- name: IsActiveMember :one
SELECT
    status = 'ACTIVE' AS active
FROM group_memberships
WHERE user_id = $1 AND group_id = $2;

-- name: CreateFriendship :exec
INSERT INTO friendships (user_id, friend_id)
VALUES ($1, $2), ($3, $4)
ON CONFLICT DO NOTHING;

-- name: GetFriendship :one
SELECT
    user_id,
    friend_id,
    created_at
FROM friendships
WHERE user_id = $1 AND friend_id = $2;

-- name: DeleteFriendship :exec
DELETE FROM friendships
WHERE (user_id = $1 AND friend_id = $2)
   OR (user_id = $3 AND friend_id = $4);

-- name: AreFriends :one
SELECT EXISTS(
    SELECT 1 FROM friendships WHERE user_id = $1 AND friend_id = $2
) AS are_friends;
