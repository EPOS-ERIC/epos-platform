-- name: CreateConfiguration :one
INSERT INTO
    configurations (
        configuration,
        title,
        description,
        list_of_services,
        is_public,
        owner_id
    )
VALUES
    (
        sqlc.arg(configuration)::jsonb,
        sqlc.arg(title)::text::nonblank_text,
        sqlc.narg(description)::text::nonblank_text_opt,
        sqlc.arg(list_of_services)::text []::nonblank_text [],
        sqlc.arg(is_public),
        sqlc.arg(owner_id)::text::nonblank_text
    )
RETURNING
    id,
    configuration,
    title::text AS title,
    COALESCE(description::text, '')::text AS description,
    list_of_services::text [] AS list_of_services,
    is_official,
    is_public,
    owner_id::text AS owner_id,
    created_at,
    updated_at;

-- name: GetConfigurationByID :one
SELECT
    id,
    configuration,
    title::text AS title,
    COALESCE(description::text, '')::text AS description,
    list_of_services::text [] AS list_of_services,
    is_official,
    is_public,
    owner_id::text AS owner_id,
    created_at,
    updated_at
FROM
    configurations
WHERE
    id = sqlc.arg(id);

-- name: ListConfigurations :many
SELECT
    id,
    title::text AS title,
    COALESCE(description::text, '')::text AS description,
    list_of_services::text [] AS list_of_services,
    is_official,
    is_public,
    owner_id::text AS owner_id,
    created_at,
    updated_at
FROM
    configurations
WHERE
    (
        (
            sqlc.arg(scope)::text = 'OWN'
            AND owner_id::text = sqlc.arg(caller_id)::text
        )
        OR (
            sqlc.arg(scope)::text = 'OWNER'
            AND owner_id::text = sqlc.arg(owner_id)::text
        )
        OR sqlc.arg(scope)::text = 'ALL'
        OR (
            sqlc.arg(scope)::text = 'OFFICIAL'
            AND is_official
        )
    )
ORDER BY
    updated_at DESC,
    id DESC;

-- name: CountNonOfficialConfigurationsForOwner :one
SELECT
    count(*)::bigint AS total
FROM
    configurations
WHERE
    owner_id = sqlc.arg(owner_id)::text::nonblank_text
    AND NOT is_official;

-- name: LockOwnerConfigurations :exec
SELECT
    pg_advisory_xact_lock(hashtextextended(sqlc.arg(owner_id)::text, 0));

-- name: GetConfigurationOwnerByID :one
SELECT
    owner_id::text AS owner_id
FROM
    configurations
WHERE
    id = sqlc.arg(id);

-- name: UpdateConfiguration :one
UPDATE
    configurations
SET
    configuration = sqlc.arg(configuration)::jsonb,
    title = sqlc.arg(title)::text::nonblank_text,
    description = sqlc.narg(description)::text::nonblank_text_opt,
    list_of_services = sqlc.arg(list_of_services)::text []::nonblank_text [],
    is_public = is_official
    OR COALESCE(sqlc.narg(is_public)::boolean, FALSE),
    updated_at = NOW()
WHERE
    id = sqlc.arg(id)
    AND (
        (
            NOT is_official
            AND owner_id = sqlc.arg(caller_id)
        )
        OR sqlc.arg(can_manage)::boolean
    )
    AND (
        NOT is_official
        OR sqlc.narg(is_public)::boolean IS DISTINCT
        FROM
            FALSE
    )
RETURNING
    id,
    configuration,
    title::text AS title,
    COALESCE(description::text, '')::text AS description,
    list_of_services::text [] AS list_of_services,
    is_official,
    is_public,
    owner_id::text AS owner_id,
    created_at,
    updated_at;

-- name: SetConfigurationOfficialStatus :one
UPDATE
    configurations
SET
    is_official = sqlc.arg(is_official),
    is_public = sqlc.arg(is_official)::boolean
    OR is_public,
    updated_at = NOW()
WHERE
    id = sqlc.arg(id)
    AND sqlc.arg(can_manage)::boolean
    AND is_official IS DISTINCT
FROM
    sqlc.arg(is_official)
RETURNING
    id,
    configuration,
    title::text AS title,
    COALESCE(description::text, '')::text AS description,
    list_of_services::text [] AS list_of_services,
    is_official,
    is_public,
    owner_id::text AS owner_id,
    created_at,
    updated_at;

-- name: DeleteConfiguration :execrows
DELETE FROM
    configurations
WHERE
    id = sqlc.arg(id)
    AND (
        (
            NOT is_official
            AND owner_id = sqlc.arg(caller_id)
        )
        OR sqlc.arg(can_manage)::boolean
    );
