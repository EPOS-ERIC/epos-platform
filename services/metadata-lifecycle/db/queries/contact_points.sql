-- name: ListContactPoints :many
SELECT
    id,
    remote_nid,
    email
FROM
    contact_points
ORDER BY
    email,
    id;

-- name: GetContactPoint :one
SELECT
    id,
    remote_nid,
    email
FROM
    contact_points
WHERE
    id = sqlc.arg(id);

-- name: CreateContactPoint :one
INSERT INTO
    contact_points (id, email)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(email)
    )
RETURNING
    id,
    remote_nid,
    email;

-- name: UpsertSyncedContactPoint :exec
INSERT INTO
    contact_points (remote_nid, email)
VALUES
    (sqlc.arg(remote_nid), sqlc.arg(email)) ON CONFLICT (remote_nid) DO
UPDATE
SET
    email = EXCLUDED.email;

-- name: DeleteAbsentSyncedContactPoints :exec
DELETE FROM
    contact_points
WHERE
    remote_nid IS NOT NULL
    AND remote_nid <> ALL (sqlc.arg(remote_nids)::bigint []);

-- name: DeleteContactPoint :execrows
DELETE FROM
    contact_points
WHERE
    id = sqlc.arg(id);

-- name: ReplaceContactPoint :one
UPDATE
    contact_points
SET
    email = sqlc.arg(email)
WHERE
    id = sqlc.arg(id)
RETURNING
    id,
    remote_nid,
    email;
