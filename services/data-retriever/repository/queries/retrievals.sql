-- name: UpsertOutputSchema :exec
INSERT INTO retrieval_output_schemas (resource_id, version_id)
VALUES (sqlc.arg(resource_id), sqlc.arg(version_id))
ON CONFLICT (resource_id, version_id) DO UPDATE
SET updated_at = NOW();

-- name: DeleteOutputParameters :exec
DELETE FROM retrieval_output_parameters
WHERE resource_id = sqlc.arg(resource_id)
  AND version_id = sqlc.arg(version_id);

-- name: CreateOutputParameter :exec
INSERT INTO retrieval_output_parameters (resource_id, version_id, name, kind, pattern)
VALUES (
  sqlc.arg(resource_id),
  sqlc.arg(version_id),
  sqlc.arg(name),
  sqlc.arg(kind),
  sqlc.narg(pattern)
);

-- name: UpsertRetrieval :one
INSERT INTO retrievals (
  resource_id, version_id, method, url_template, generated_url,
  path_params_json, query_params_json, headers_json,
  request_body_text, request_body_hash, request_body_is_json,
  http_status, content_type, is_binary, payload_hash, payload_size_bytes,
  used_auth, fetched_at
) VALUES (
  sqlc.arg(resource_id), sqlc.arg(version_id),
  sqlc.arg(method), sqlc.arg(url_template), sqlc.arg(generated_url),
  sqlc.arg(path_params_json)::jsonb, sqlc.arg(query_params_json)::jsonb,
  sqlc.arg(headers_json)::jsonb, sqlc.narg(request_body_text),
  sqlc.arg(request_body_hash), sqlc.arg(request_body_is_json),
  sqlc.arg(http_status), sqlc.arg(content_type), sqlc.arg(is_binary),
  sqlc.arg(payload_hash), sqlc.arg(payload_size_bytes), sqlc.arg(used_auth),
  sqlc.arg(fetched_at)
)
ON CONFLICT (resource_id, version_id, method, generated_url, request_body_hash)
DO UPDATE SET
  url_template = EXCLUDED.url_template,
  path_params_json = EXCLUDED.path_params_json,
  query_params_json = EXCLUDED.query_params_json,
  headers_json = EXCLUDED.headers_json,
  request_body_text = EXCLUDED.request_body_text,
  request_body_is_json = EXCLUDED.request_body_is_json,
  http_status = EXCLUDED.http_status,
  content_type = EXCLUDED.content_type,
  is_binary = EXCLUDED.is_binary,
  payload_hash = EXCLUDED.payload_hash,
  payload_size_bytes = EXCLUDED.payload_size_bytes,
  used_auth = EXCLUDED.used_auth,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = NOW()
RETURNING id, resource_id, version_id, method, url_template, generated_url,
  http_status, content_type, is_binary, payload_hash, payload_size_bytes, used_auth,
  fetched_at, created_at, request_body_text;

-- name: FinalizeRetrieval :exec
UPDATE retrievals
SET payload_hash = sqlc.arg(payload_hash),
    payload_size_bytes = sqlc.arg(payload_size_bytes),
    updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: GetRetrievalByID :one
SELECT id, resource_id, version_id, method, url_template, generated_url,
  http_status, content_type, is_binary, payload_hash, payload_size_bytes, used_auth,
  fetched_at, created_at, request_body_text
FROM retrievals
WHERE id = sqlc.arg(id);

-- name: GetOutputParameters :many
SELECT name, kind, pattern
FROM retrieval_output_parameters
WHERE resource_id = sqlc.arg(resource_id)
  AND version_id = sqlc.arg(version_id)
ORDER BY name;
