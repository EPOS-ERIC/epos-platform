-- name: ListGroups :many
SELECT
    sqlc.embed(groups)
FROM
    groups
ORDER BY
    position,
    name,
    id;

-- name: GetGroupByID :one
SELECT
    sqlc.embed(groups)
FROM
    groups
WHERE
    id = $1;
