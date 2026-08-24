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
INSERT INTO contact_points (
    id,
    source_iri,
    email,
    role
)
VALUES (
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
