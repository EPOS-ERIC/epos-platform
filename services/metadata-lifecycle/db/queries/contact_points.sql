-- name: ListContactPoints :many
SELECT
    id,
    email
FROM
    contact_points
ORDER BY
    email,
    id;

-- name: GetContactPoint :one
SELECT
    id,
    email
FROM
    contact_points
WHERE
    id = sqlc.arg(id);

-- name: CreateContactPoint :one
INSERT INTO
    contact_points (id, email)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(email)
    )
RETURNING
    id,
    email;

-- name: DeleteContactPoint :execrows
DELETE FROM
    contact_points
WHERE
    id = sqlc.arg(id);

-- name: ReplaceContactPoint :one
UPDATE
    contact_points
SET
    email = sqlc.arg(email)
WHERE
    id = sqlc.arg(id)
RETURNING
    id,
    email;
