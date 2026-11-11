-- name: ListConceptRecords :many
SELECT
    id,
    name
FROM
    concepts
ORDER BY
    lower(name),
    id;

-- name: GetConceptRecord :one
SELECT
    id,
    name
FROM
    concepts
WHERE
    id = sqlc.arg(id);

-- name: CreateConceptRecord :one
INSERT INTO
    concepts (name)
VALUES
    (sqlc.arg(name))
RETURNING
    id,
    name;

-- name: ReplaceConceptRecord :one
UPDATE
    concepts
SET
    name = sqlc.arg(name)
WHERE
    id = sqlc.arg(id)
RETURNING
    id,
    name;

-- name: DeleteConceptRecord :execrows
DELETE FROM
    concepts
WHERE
    id = sqlc.arg(id);
