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
    (
        sqlc.arg(name),
        sqlc.arg(description),
        sqlc.arg(repository),
        sqlc.arg(branch),
        sqlc.arg(runtime),
        sqlc.arg(arguments),
        sqlc.arg(created_by),
        sqlc.arg(created_by)
    )
RETURNING
    *;

-- name: GetPlugin :one
SELECT
    *
FROM
    plugins
WHERE
    id = sqlc.arg(id);

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
    name = sqlc.arg(name),
    description = sqlc.arg(description),
    repository = sqlc.arg(repository),
    branch = sqlc.arg(branch),
    runtime = sqlc.arg(runtime),
    arguments = sqlc.arg(arguments),
    enabled = sqlc.arg(enabled),
    updated_by = sqlc.arg(updated_by)
WHERE
    id = sqlc.arg(id)
RETURNING
    *;

-- name: DeletePlugin :execrows
DELETE FROM
    plugins
WHERE
    id = sqlc.arg(id);

-- name: GetPluginInstallation :one
SELECT
    installed_commit,
    runtime
FROM
    plugins
WHERE
    id = sqlc.arg(id);

-- name: SetPluginInstallation :execrows
UPDATE
    plugins
SET
    installed_commit = sqlc.narg(installed_commit)
WHERE
    id = sqlc.arg(id);
