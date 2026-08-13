-- name: UpsertImportGroup :one
INSERT INTO
    groups (
        id,
        name,
        description,
        source_iri,
        concept,
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
        NULL,
        sqlc.arg(source_iri),
        sqlc.arg(concept),
        sqlc.narg(label),
        sqlc.narg(logo),
        sqlc.narg(PAGE),
        sqlc.narg(color),
        sqlc.arg(position)
    ) ON CONFLICT (source_iri) DO
UPDATE
SET
    name = EXCLUDED.name,
    concept = EXCLUDED.concept,
    label = EXCLUDED.label,
    logo = EXCLUDED.logo,
    PAGE = EXCLUDED.page,
    color = EXCLUDED.color,
    position = EXCLUDED.position,
    updated_at = NOW()
RETURNING
    id;

-- name: UpsertImportOrganization :one
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
    ) ON CONFLICT (source_iri) DO
UPDATE
SET
    name = EXCLUDED.name,
    street = EXCLUDED.street,
    locality = EXCLUDED.locality,
    postal_code = EXCLUDED.postal_code,
    country = EXCLUDED.country,
    logo_url = EXCLUDED.logo_url,
    page_url = EXCLUDED.page_url,
    email = EXCLUDED.email,
    lei_code = EXCLUDED.lei_code
RETURNING
    id;

-- name: DeleteImportOrganizationIdentifiers :exec
DELETE FROM
    identifiers
WHERE
    organization_id = sqlc.arg(organization_id);

-- name: AddImportOrganizationIdentifier :exec
INSERT INTO
    identifiers (source_iri, TYPE, value, organization_id)
VALUES
    (
        sqlc.arg(source_iri),
        sqlc.arg(TYPE),
        sqlc.arg(value),
        sqlc.arg(organization_id)
    );

-- name: UpsertImportContactPoint :one
INSERT INTO
    contact_points (id, source_iri, email, role)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(source_iri),
        sqlc.arg(email),
        sqlc.arg(role)
    ) ON CONFLICT (source_iri) DO
UPDATE
SET
    email = EXCLUDED.email,
    role = EXCLUDED.role
RETURNING
    id;

-- name: UpsertImportCategory :one
INSERT INTO
    categories (
        id,
        group_id,
        parent_category_id,
        source_iri,
        name
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(group_id),
        NULL,
        sqlc.arg(source_iri),
        sqlc.arg(name)
    ) ON CONFLICT (source_iri) DO
UPDATE
SET
    name = EXCLUDED.name,
    parent_category_id = NULL,
    updated_at = NOW()
RETURNING
    id,
    group_id;

-- name: SetImportCategoryParent :exec
UPDATE
    categories
SET
    parent_category_id = sqlc.arg(parent_id),
    updated_at = NOW()
WHERE
    id = sqlc.arg(child_id)
    AND group_id = sqlc.arg(group_id);

-- name: GetImportDataset :one
SELECT
    d.*,
    v.id AS active_version_id,
    v.status AS active_status,
    v.revision AS active_revision
FROM
    datasets d
    LEFT JOIN dataset_versions v ON v.dataset_id = d.id
    AND v.status IN ('DRAFT', 'SUBMITTED', 'REJECTED')
WHERE
    d.source_iri = sqlc.arg(source_iri) FOR
UPDATE
    OF d;
