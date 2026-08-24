-- name: ListOrganizations :many
SELECT
    id,
    source_iri,
    name,
    street,
    locality,
    postal_code,
    country,
    logo_url,
    page_url,
    email,
    lei_code
FROM
    organizations
ORDER BY
    id;

-- name: GetOrganization :one
SELECT
    id,
    source_iri,
    name,
    street,
    locality,
    postal_code,
    country,
    logo_url,
    page_url,
    email,
    lei_code
FROM
    organizations
WHERE
    id = sqlc.arg(id);

-- name: ListOrganizationsByIDs :many
SELECT
    id,
    source_iri,
    name,
    street,
    locality,
    postal_code,
    country,
    logo_url,
    page_url,
    email,
    lei_code
FROM
    organizations
WHERE
    id = ANY(sqlc.arg(organization_ids)::uuid []);

-- name: ListContactPointIDs :many
SELECT
    id
FROM
    contact_points
WHERE
    id = ANY(sqlc.arg(contact_point_ids)::uuid []);

-- name: CreateOrganization :one
INSERT INTO
    organizations (
        id,
        source_iri,
        name,
        street,
        locality,
        postal_code,
        country,
        logo_url,
        page_url,
        email,
        lei_code
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(source_iri),
        sqlc.arg(name),
        sqlc.narg(street),
        sqlc.narg(locality),
        sqlc.narg(postal_code),
        sqlc.narg(country),
        sqlc.narg(logo_url),
        sqlc.narg(page_url),
        sqlc.narg(email),
        sqlc.narg(lei_code)
    )
RETURNING
    *;

-- name: CreateOrganizationIdentifier :exec
INSERT INTO
    identifiers (id, source_iri, TYPE, value, organization_id)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(source_iri),
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

-- name: CreateOrganizationContactPoint :exec
INSERT INTO
    organization_contact_points (organization_id, contact_point_id)
VALUES
    (
        sqlc.arg(organization_id),
        sqlc.arg(contact_point_id)
    );

-- name: ListOrganizationIdentifiers :many
SELECT
    organization_id,
    sqlc.embed(identifier)
FROM
    identifiers AS identifier
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

-- name: ListOrganizationContactPointIDs :many
SELECT
    organization_id,
    contact_point_id
FROM
    organization_contact_points
WHERE
    organization_id = ANY(sqlc.arg(organization_ids)::uuid [])
ORDER BY
    organization_id,
    contact_point_id;
