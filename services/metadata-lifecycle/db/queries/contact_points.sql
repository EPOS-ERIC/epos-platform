-- name: ListContactPoints :many
SELECT
    id,
    email,
    role
FROM
    contact_points
ORDER BY
    email,
    role,
    id;

-- name: GetContactPoint :one
SELECT
    id,
    email,
    role
FROM
    contact_points
WHERE
    id = sqlc.arg(id);

-- name: CreateContactPoint :one
INSERT INTO
    contact_points (id, email, role)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(email),
        sqlc.arg(role)
    )
RETURNING
    id,
    email,
    role;

-- name: DeleteContactPoint :execrows
DELETE FROM
    contact_points
WHERE
    id = sqlc.arg(id);

-- name: ReplaceContactPoint :one
UPDATE
    contact_points
SET
    email = sqlc.arg(email),
    role = sqlc.arg(role)
WHERE
    id = sqlc.arg(id)
RETURNING
    id,
    email,
    role;
