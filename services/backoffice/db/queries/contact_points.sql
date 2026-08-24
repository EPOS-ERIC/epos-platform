-- name: ListContactPoints :many
SELECT
    id,
    source_iri,
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
    source_iri,
    email,
    role
FROM
    contact_points
WHERE
    id = sqlc.arg(id);

-- name: CreateContactPoint :one
INSERT INTO
    contact_points (id, source_iri, email, role)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(source_iri),
        sqlc.arg(email),
        sqlc.arg(role)
    )
RETURNING
    id,
    source_iri,
    email,
    role;

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
    source_iri,
    email,
    role;
