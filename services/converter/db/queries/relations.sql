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
    (
        sqlc.arg(distribution_id),
        sqlc.arg(plugin_id),
        sqlc.arg(input_format),
        sqlc.arg(output_format),
        sqlc.arg(created_by),
        sqlc.arg(created_by)
    )
RETURNING
    *;

-- name: GetRelation :one
SELECT
    *
FROM
    conversion_relations
WHERE
    id = sqlc.arg(id);

-- name: ListRelations :many
SELECT
    *
FROM
    conversion_relations
ORDER BY
    id;

-- name: ListDistributionRelations :many
SELECT
    sqlc.embed(conversion_relations),
    sqlc.embed(plugins)
FROM
    conversion_relations
    JOIN plugins ON plugins.id = conversion_relations.plugin_id
WHERE
    conversion_relations.distribution_id = sqlc.arg(distribution_id)
ORDER BY
    plugins.name,
    plugins.id,
    conversion_relations.id;

-- name: UpdateRelation :one
UPDATE
    conversion_relations
SET
    distribution_id = sqlc.arg(distribution_id),
    plugin_id = sqlc.arg(plugin_id),
    input_format = sqlc.arg(input_format),
    output_format = sqlc.arg(output_format),
    updated_by = sqlc.arg(updated_by)
WHERE
    id = sqlc.arg(id)
RETURNING
    *;

-- name: DeleteRelation :execrows
DELETE FROM
    conversion_relations
WHERE
    id = sqlc.arg(id);
