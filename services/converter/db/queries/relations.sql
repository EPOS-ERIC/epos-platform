-- name: CreateRelation :one
INSERT INTO
    conversion_relations (
        distribution_id,
        plugin_id,
        input_format,
        output_format,
        created_by,
        updated_by
    )
VALUES
    ($1, $2, $3, $4, $5, $5)
RETURNING
    *;

-- name: GetRelation :one
SELECT
    *
FROM
    conversion_relations
WHERE
    id = $1;

-- name: ListRelations :many
SELECT
    *
FROM
    conversion_relations
ORDER BY
    id;

-- name: UpdateRelation :one
UPDATE
    conversion_relations
SET
    distribution_id = $2,
    plugin_id = $3,
    input_format = $4,
    output_format = $5,
    updated_by = $6
WHERE
    id = $1
RETURNING
    *;

-- name: DeleteRelation :execrows
DELETE FROM
    conversion_relations
WHERE
    id = $1;
