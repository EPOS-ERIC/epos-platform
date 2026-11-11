-- TODO: we sholud probably split this huge file in separate files
-- name: ListGroups :many
SELECT
    id,
    name,
    description,
    concept_id,
    label,
    logo,
    PAGE,
    color,
    position,
    created_at,
    updated_at
FROM
    groups
ORDER BY
    position,
    name,
    id;

-- name: GetGroupByID :one
SELECT
    id,
    name,
    description,
    concept_id,
    label,
    logo,
    PAGE,
    color,
    position,
    created_at,
    updated_at
FROM
    groups
WHERE
    id = sqlc.arg(id);

-- name: CreateGroup :one
INSERT INTO
    groups (
        id,
        name,
        description,
        concept_id,
        label,
        logo,
        PAGE,
        color,
        position
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(name),
        sqlc.arg(description),
        sqlc.arg(concept_id),
        sqlc.arg(label),
        sqlc.arg(logo),
        sqlc.arg(PAGE),
        sqlc.arg(color),
        sqlc.arg(position)
    )
RETURNING
    id,
    name,
    description,
    concept_id,
    label,
    logo,
    PAGE,
    color,
    position,
    created_at,
    updated_at;

-- name: ReplaceGroup :one
UPDATE
    groups
SET
    name = sqlc.arg(name),
    description = sqlc.arg(description),
    label = sqlc.arg(label),
    logo = sqlc.arg(logo),
    PAGE = sqlc.arg(PAGE),
    color = sqlc.arg(color),
    position = sqlc.arg(position),
    concept_id = sqlc.arg(concept_id)
WHERE
    id = sqlc.arg(id)
RETURNING
    id,
    name,
    description,
    concept_id,
    label,
    logo,
    PAGE,
    color,
    position,
    created_at,
    updated_at;

-- name: DeleteGroup :one
WITH target AS (
    SELECT
        group_row.id
    FROM
        groups AS group_row
    WHERE
        group_row.id = sqlc.arg(id)
),
deleted AS (
    DELETE FROM
        groups AS group_row
    WHERE
        group_row.id = (
            SELECT
                target.id
            FROM
                target
        )
        AND NOT EXISTS (
            SELECT
                1
            FROM
                datasets
            WHERE
                group_id = sqlc.arg(id)
            UNION
            ALL
            SELECT
                1
            FROM
                software_applications
            WHERE
                group_id = sqlc.arg(id)
            UNION
            ALL
            SELECT
                1
            FROM
                software_source_codes
            WHERE
                group_id = sqlc.arg(id)
        )
    RETURNING
        id
)
SELECT
    CASE
        WHEN EXISTS (
            SELECT
                1
            FROM
                deleted
        ) THEN 'DELETED'
        WHEN NOT EXISTS (
            SELECT
                1
            FROM
                target
        ) THEN 'NOT_FOUND'
        ELSE 'CONFLICT'
    END AS outcome;
