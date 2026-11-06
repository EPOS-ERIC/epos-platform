-- name: UpsertImportGroup :one
INSERT INTO
    groups (
        id,
        name,
        description,
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
        sqlc.narg(description),
        sqlc.arg(concept),
        sqlc.narg(label),
        sqlc.narg(logo),
        sqlc.narg(PAGE),
        sqlc.narg(color),
        sqlc.arg(position)
    ) ON CONFLICT (id) DO
UPDATE
SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
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
        sqlc.arg(name),
        sqlc.narg(street),
        sqlc.narg(locality),
        sqlc.narg(postal_code),
        sqlc.narg(country),
        sqlc.narg(logo_url),
        sqlc.narg(page_url),
        sqlc.narg(email),
        sqlc.narg(lei_code)
    ) ON CONFLICT (id) DO
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
    organization_identifiers
WHERE
    organization_id = sqlc.arg(organization_id);

-- name: AddImportOrganizationIdentifier :exec
INSERT INTO
    organization_identifiers (TYPE, value, organization_id)
VALUES
    (
        sqlc.arg(TYPE),
        sqlc.arg(value),
        sqlc.arg(organization_id)
    );

-- name: DeleteImportOrganizationMemberships :exec
DELETE FROM
    organization_memberships
WHERE
    child_organization_id = sqlc.arg(child_organization_id);

-- name: DeleteImportOrganizationContactPoints :exec
DELETE FROM
    organization_contact_points
WHERE
    organization_id = sqlc.arg(organization_id);

-- name: AddImportOrganizationContactPoint :exec
INSERT INTO
    organization_contact_points (organization_id, contact_point_id)
VALUES
    (
        sqlc.arg(organization_id),
        sqlc.arg(contact_point_id)
    );

-- name: AddImportOrganizationMembership :exec
INSERT INTO
    organization_memberships (child_organization_id, parent_organization_id)
VALUES
    (
        sqlc.arg(child_organization_id),
        sqlc.arg(parent_organization_id)
    );

-- name: UpsertImportContactPoint :one
WITH existing_natural_key AS (
    SELECT
        id
    FROM
        contact_points
    WHERE
        contact_points.email = sqlc.arg(email)
),
updated_id AS (
    UPDATE
        contact_points
    SET
        email = sqlc.arg(email)
    WHERE
        contact_points.id = sqlc.arg(id)
        AND NOT EXISTS (
            SELECT
                1
            FROM
                existing_natural_key
        )
    RETURNING
        id
),
inserted AS (
    INSERT INTO
        contact_points (id, email)
    SELECT
        sqlc.arg(id),
        sqlc.arg(email)
    WHERE
        NOT EXISTS (
            SELECT
                1
            FROM
                existing_natural_key
        )
        AND NOT EXISTS (
            SELECT
                1
            FROM
                updated_id
        ) ON CONFLICT (id) DO NOTHING
    RETURNING
        id
)
SELECT
    id
FROM
    existing_natural_key
UNION
ALL
SELECT
    id
FROM
    updated_id
UNION
ALL
SELECT
    id
FROM
    inserted
LIMIT
    1;

-- name: UpsertImportCategory :one
INSERT INTO
    categories (
        id,
        group_id,
        parent_category_id,
        name
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(group_id),
        NULL,
        sqlc.arg(name)
    ) ON CONFLICT (id) DO
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
    d.id = sqlc.arg(id) FOR
UPDATE
    OF d;

-- name: CreateImportDataset :one
INSERT INTO
    datasets (id, group_id, created_by)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(group_id),
        sqlc.arg(created_by)
    )
RETURNING
    id,
    group_id,
    created_by,
    created_at;

-- name: GetImportSoftwareApplication :one
SELECT
    resource.*,
    version.id AS active_version_id,
    version.status AS active_status,
    version.revision AS active_revision
FROM
    software_applications AS resource
    LEFT JOIN software_application_versions AS version ON version.software_application_id = resource.id
    AND version.status IN ('DRAFT', 'SUBMITTED', 'REJECTED')
WHERE
    resource.id = sqlc.arg(id) FOR
UPDATE
    OF resource;

-- name: CreateImportSoftwareApplication :one
INSERT INTO
    software_applications (id, group_id, created_by)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(group_id),
        sqlc.arg(created_by)
    )
RETURNING
    *;

-- name: GetImportSoftwareSourceCode :one
SELECT
    resource.*,
    version.id AS active_version_id,
    version.status AS active_status,
    version.revision AS active_revision
FROM
    software_source_codes AS resource
    LEFT JOIN software_source_code_versions AS version ON version.software_source_code_id = resource.id
    AND version.status IN ('DRAFT', 'SUBMITTED', 'REJECTED')
WHERE
    resource.id = sqlc.arg(id) FOR
UPDATE
    OF resource;

-- name: CreateImportSoftwareSourceCode :one
INSERT INTO
    software_source_codes (id, group_id, created_by)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(group_id),
        sqlc.arg(created_by)
    )
RETURNING
    *;

-- name: GetImportOrganizationByName :one
SELECT
    id
FROM
    organizations
WHERE
    name = sqlc.arg(name);
