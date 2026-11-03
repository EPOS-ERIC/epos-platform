-- name: CreatePlugin :one
INSERT INTO
    plugins (
        name,
        description,
        repository,
        branch,
        runtime,
        arguments,
        created_by,
        updated_by
    )
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $7)
RETURNING
    *;

-- name: GetPlugin :one
SELECT
    *
FROM
    plugins
WHERE
    id = $1;

-- name: ListPlugins :many
SELECT
    *
FROM
    plugins
ORDER BY
    name,
    id;

-- name: UpdatePlugin :one
UPDATE
    plugins
SET
    name = $2,
    description = $3,
    repository = $4,
    branch = $5,
    runtime = $6,
    arguments = $7,
    enabled = $8,
    updated_by = $9
WHERE
    id = $1
RETURNING
    *;

-- name: DeletePlugin :execrows
DELETE FROM
    plugins
WHERE
    id = $1;

-- name: GetPluginInstallation :one
SELECT
    installed_commit,
    runtime
FROM
    plugins
WHERE
    id = $1;

-- name: SetPluginInstallation :execrows
UPDATE
    plugins
SET
    installed_commit = $2
WHERE
    id = $1;
