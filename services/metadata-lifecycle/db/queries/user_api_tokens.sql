-- name: IssueUserAPIToken :execrows
UPDATE
    users
SET
    api_token_digest = sqlc.arg(digest)
WHERE
    id = sqlc.arg(user_id)
    AND STATUS = 'ACTIVE'
    AND api_token_digest IS NULL;

-- name: RotateUserAPIToken :execrows
UPDATE
    users
SET
    api_token_digest = sqlc.arg(digest)
WHERE
    id = sqlc.arg(user_id)
    AND STATUS = 'ACTIVE'
    AND api_token_digest IS NOT NULL;

-- name: RevokeUserAPIToken :execrows
UPDATE
    users
SET
    api_token_digest = NULL
WHERE
    id = sqlc.arg(user_id)
    AND api_token_digest IS NOT NULL;

-- name: FindUserByAPITokenDigest :one
SELECT
    id
FROM
    users
WHERE
    api_token_digest = sqlc.arg(digest)
    AND STATUS = 'ACTIVE';
