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
