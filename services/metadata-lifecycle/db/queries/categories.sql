-- name: ListGroupCategories :many
SELECT
    id,
    group_id,
    parent_category_id,
    name,
    created_at,
    updated_at
FROM
    categories
WHERE
    group_id = sqlc.arg(group_id)
ORDER BY
    parent_category_id NULLS FIRST,
    lower(trim(name)),
    id;

-- name: GetGroupCategory :one
SELECT
    id,
    group_id,
    parent_category_id,
    name,
    created_at,
    updated_at
FROM
    categories
WHERE
    group_id = sqlc.arg(group_id)
    AND id = sqlc.arg(id);

-- name: ListLifecycleCategoryIDsByGroup :many
SELECT
    id
FROM
    categories
WHERE
    group_id = sqlc.arg(group_id)
    AND id = ANY(sqlc.arg(category_ids)::uuid [])
ORDER BY
    id;

-- name: CreateGroupCategory :one
INSERT INTO
    categories (
        id,
        group_id,
        parent_category_id,
        name
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(group_id),
        sqlc.narg(parent_category_id),
        sqlc.arg(name)
    )
RETURNING
    id,
    group_id,
    parent_category_id,
    name,
    created_at,
    updated_at;

-- name: ReplaceGroupCategory :one
UPDATE
    categories
SET
    name = sqlc.arg(name),
    parent_category_id = sqlc.arg(parent_category_id),
    updated_at = NOW()
WHERE
    group_id = sqlc.arg(group_id)
    AND id = sqlc.arg(id)
RETURNING
    id,
    group_id,
    parent_category_id,
    name,
    created_at,
    updated_at;

-- name: DeleteGroupCategory :execrows
DELETE FROM
    categories
WHERE
    group_id = sqlc.arg(group_id)
    AND id = sqlc.arg(id);
