-- name: ListOrganizations :many
SELECT
    id,
    remote_nid,
    name,
    street,
    locality,
    postal_code,
    country,
    logo_url,
    page_url,
    email
FROM
    organizations
ORDER BY
    id;

-- name: GetOrganization :one
SELECT
    id,
    remote_nid,
    name,
    street,
    locality,
    postal_code,
    country,
    logo_url,
    page_url,
    email
FROM
    organizations
WHERE
    id = sqlc.arg(id);

-- name: GetOrganizationForUpdate :one
SELECT
    id,
    remote_nid,
    name,
    street,
    locality,
    postal_code,
    country,
    logo_url,
    page_url,
    email
FROM
    organizations
WHERE
    id = sqlc.arg(id) FOR
UPDATE
;

-- name: ListOrganizationsByIDs :many
SELECT
    id,
    remote_nid,
    name,
    street,
    locality,
    postal_code,
    country,
    logo_url,
    page_url,
    email
FROM
    organizations
WHERE
    id = ANY(sqlc.arg(organization_ids)::uuid []);

-- name: CreateOrganization :one
INSERT INTO
    organizations (
        id,
        name,
        street,
        locality,
        postal_code,
        country,
        logo_url,
        page_url,
        email
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(name),
        sqlc.narg(street),
        sqlc.narg(locality),
        sqlc.narg(postal_code),
        sqlc.narg(country),
        sqlc.narg(logo_url),
        sqlc.narg(page_url),
        sqlc.narg(email)
    )
RETURNING
    *;

-- name: CreateOrganizationIdentifier :exec
INSERT INTO
    organization_identifiers (id, TYPE, value, organization_id)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(TYPE),
        sqlc.arg(value),
        sqlc.arg(organization_id)
    );

-- name: CreateOrganizationParent :exec
INSERT INTO
    organization_memberships (child_organization_id, parent_organization_id)
VALUES
    (
        sqlc.arg(child_organization_id),
        sqlc.arg(parent_organization_id)
    );

-- name: ListOrganizationIdentifiers :many
SELECT
    organization_id,
    sqlc.embed(identifier)
FROM
    organization_identifiers AS identifier
WHERE
    organization_id = ANY(sqlc.arg(organization_ids)::uuid [])
ORDER BY
    organization_id,
    identifier.id;

-- name: ListOrganizationParents :many
SELECT
    child_organization_id,
    parent_organization_id
FROM
    organization_memberships
WHERE
    child_organization_id = ANY(sqlc.arg(organization_ids)::uuid [])
ORDER BY
    child_organization_id,
    parent_organization_id;

-- name: ReplaceOrganization :one
UPDATE
    organizations
SET
    name = sqlc.arg(name),
    street = sqlc.narg(street),
    locality = sqlc.narg(locality),
    postal_code = sqlc.narg(postal_code),
    country = sqlc.narg(country),
    logo_url = sqlc.narg(logo_url),
    page_url = sqlc.narg(page_url),
    email = sqlc.narg(email)
WHERE
    id = sqlc.arg(id)
RETURNING
    *;

-- name: DeleteOrganizationIdentifiers :exec
DELETE FROM
    organization_identifiers
WHERE
    organization_id = sqlc.arg(organization_id);

-- name: DeleteOrganizationIdentifier :exec
DELETE FROM
    organization_identifiers
WHERE
    id = sqlc.arg(id);

-- name: DeleteOrganization :execrows
DELETE FROM
    organizations
WHERE
    id = sqlc.arg(id);

-- name: DeleteOrganizationParents :exec
DELETE FROM
    organization_memberships
WHERE
    child_organization_id = sqlc.arg(organization_id);

-- name: UpsertSyncedOrganization :one
INSERT INTO
    organizations (
        remote_nid,
        name,
        street,
        locality,
        postal_code,
        country,
        logo_url,
        page_url,
        email
    )
VALUES
    (
        sqlc.arg(remote_nid),
        sqlc.arg(name),
        sqlc.narg(street),
        sqlc.narg(locality),
        sqlc.narg(postal_code),
        sqlc.narg(country),
        sqlc.narg(logo_url),
        sqlc.narg(page_url),
        sqlc.narg(email)
    ) ON CONFLICT (remote_nid) DO
UPDATE
SET
    name = EXCLUDED.name,
    street = EXCLUDED.street,
    locality = EXCLUDED.locality,
    postal_code = EXCLUDED.postal_code,
    country = EXCLUDED.country,
    logo_url = EXCLUDED.logo_url,
    page_url = EXCLUDED.page_url,
    email = EXCLUDED.email
RETURNING
    id;

-- name: DeleteAbsentSyncedOrganizations :exec
DELETE FROM
    organizations
WHERE
    remote_nid IS NOT NULL
    AND NOT (
        remote_nid = ANY(sqlc.arg(remote_nids)::bigint [])
    );
