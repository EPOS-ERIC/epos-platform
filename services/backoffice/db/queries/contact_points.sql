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
