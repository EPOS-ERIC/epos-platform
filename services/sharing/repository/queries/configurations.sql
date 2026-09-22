-- name: CreateConfiguration :one
INSERT INTO configurations (
  id, configuration, title, description, list_of_services,
  is_official, is_public, owner_id
) VALUES (
  sqlc.arg(id),
  sqlc.arg(configuration)::jsonb,
  sqlc.narg(title),
  sqlc.narg(description),
  sqlc.arg(list_of_services)::jsonb,
  sqlc.arg(is_official),
  sqlc.arg(is_public),
  sqlc.arg(owner_id)
)
RETURNING id, configuration, title, description, list_of_services,
  is_official, is_public, owner_id, created_at, updated_at;

-- name: GetConfigurationByID :one
SELECT id, configuration, title, description, list_of_services,
  is_official, is_public, owner_id, created_at, updated_at
FROM configurations
WHERE id = sqlc.arg(id);

-- name: ListConfigurations :many
SELECT id, configuration, title, description, list_of_services,
  is_official, is_public, owner_id, created_at, updated_at
FROM configurations
WHERE (is_public OR owner_id = COALESCE(sqlc.narg(caller_id), '') OR sqlc.arg(include_all))
  AND (is_official = COALESCE(sqlc.narg(official), is_official))
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountConfigurations :one
SELECT COUNT(*) AS total
FROM configurations
WHERE (is_public OR owner_id = COALESCE(sqlc.narg(caller_id), '') OR sqlc.arg(include_all))
  AND (is_official = COALESCE(sqlc.narg(official), is_official));

-- name: UpdateConfiguration :one
UPDATE configurations
SET configuration = sqlc.arg(configuration)::jsonb,
    title = sqlc.narg(title),
    description = sqlc.narg(description),
    list_of_services = sqlc.arg(list_of_services)::jsonb,
    is_official = sqlc.arg(is_official),
    is_public = sqlc.arg(is_public),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, configuration, title, description, list_of_services,
  is_official, is_public, owner_id, created_at, updated_at;

-- name: DeleteConfiguration :execrows
DELETE FROM configurations
WHERE id = sqlc.arg(id);
