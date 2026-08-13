-- name: CreateLifecycleDataset :one
INSERT INTO
    datasets (source_iri, group_id, created_by)
VALUES
    (
        sqlc.arg(source_iri),
        sqlc.arg(group_id),
        sqlc.arg(created_by)
    )
RETURNING
    id,
    source_iri,
    group_id,
    created_by,
    created_at;

-- name: CreateLifecycleDatasetVersion :one
INSERT INTO
    dataset_versions (
        dataset_id,
        title,
        description,
        keywords,
        accrual_periodicity,
        quality_annotation,
        access_right,
        source_created_at,
        source_updated_at,
        spatial_point,
        spatial_polygon,
        temporal_start,
        temporal_end,
        created_by
    )
VALUES
    (
        sqlc.arg(dataset_id),
        sqlc.arg(title),
        sqlc.arg(description),
        sqlc.arg(keywords),
        sqlc.narg(accrual_periodicity),
        sqlc.narg(quality_annotation),
        sqlc.narg(access_right),
        sqlc.narg(source_created_at),
        sqlc.narg(source_updated_at),
        CASE
            WHEN sqlc.narg(spatial_point)::text IS NULL THEN NULL
            ELSE ST_SetSRID(
                ST_GeomFromGeoJSON(sqlc.narg(spatial_point)::text),
                4326
            )
        END,
        CASE
            WHEN sqlc.narg(spatial_polygon)::text IS NULL THEN NULL
            ELSE ST_SetSRID(
                ST_GeomFromGeoJSON(sqlc.narg(spatial_polygon)::text),
                4326
            )
        END,
        sqlc.narg(temporal_start),
        sqlc.narg(temporal_end),
        sqlc.arg(created_by)
    )
RETURNING
    *;

-- name: ReplaceLifecycleDatasetDraft :one
UPDATE
    dataset_versions
SET
    title = sqlc.arg(title),
    description = sqlc.arg(description),
    keywords = sqlc.arg(keywords),
    accrual_periodicity = sqlc.narg(accrual_periodicity),
    quality_annotation = sqlc.narg(quality_annotation),
    access_right = sqlc.narg(access_right),
    source_created_at = sqlc.narg(source_created_at),
    source_updated_at = sqlc.narg(source_updated_at),
    spatial_point = CASE
        WHEN sqlc.narg(spatial_point)::text IS NULL THEN NULL
        ELSE ST_SetSRID(
            ST_GeomFromGeoJSON(sqlc.narg(spatial_point)::text),
            4326
        )
    END,
    spatial_polygon = CASE
        WHEN sqlc.narg(spatial_polygon)::text IS NULL THEN NULL
        ELSE ST_SetSRID(
            ST_GeomFromGeoJSON(sqlc.narg(spatial_polygon)::text),
            4326
        )
    END,
    temporal_start = sqlc.narg(temporal_start),
    temporal_end = sqlc.narg(temporal_end),
    revision = revision + 1
WHERE
    id = sqlc.arg(id)
    AND dataset_id = sqlc.arg(dataset_id)
    AND STATUS = 'DRAFT'
    AND revision = sqlc.arg(revision)
RETURNING
    *;

-- name: SubmitLifecycleDatasetVersion :one
UPDATE
    dataset_versions
SET
    STATUS = 'SUBMITTED',
    submitted_by = sqlc.arg(actor_id),
    submitted_at = NOW()
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'DRAFT'
RETURNING
    *;

-- name: WithdrawLifecycleDatasetVersion :one
UPDATE
    dataset_versions
SET
    STATUS = 'DRAFT'
WHERE
    dataset_versions.id = sqlc.arg(id)
    AND dataset_versions.status = 'SUBMITTED'
RETURNING
    *;

-- name: RejectLifecycleDatasetVersion :one
UPDATE
    dataset_versions
SET
    STATUS = 'REJECTED',
    rejected_by = sqlc.arg(actor_id),
    rejected_at = NOW(),
    rejection_note = sqlc.arg(rejection_note)
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'SUBMITTED'
RETURNING
    *;

-- name: ReopenLifecycleDatasetVersion :one
UPDATE
    dataset_versions
SET
    STATUS = 'DRAFT'
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'REJECTED'
RETURNING
    *;

-- name: PublishLifecycleDatasetVersion :one
WITH target AS (
    SELECT
        dataset_id
    FROM
        dataset_versions
    WHERE
        id = sqlc.arg(version_id)
        AND STATUS = 'SUBMITTED' FOR
    UPDATE
),
archived AS (
    UPDATE
        dataset_versions
    SET
        STATUS = 'ARCHIVED',
        archived_by = sqlc.arg(actor_id),
        archived_at = NOW()
    WHERE
        dataset_id = (
            SELECT
                dataset_id
            FROM
                target
        )
        AND STATUS = 'PUBLISHED'
)
UPDATE
    dataset_versions
SET
    STATUS = 'PUBLISHED',
    published_by = sqlc.arg(actor_id),
    published_at = NOW(),
    rejected_by = NULL,
    rejected_at = NULL,
    rejection_note = NULL
WHERE
    dataset_versions.id = sqlc.arg(version_id)
    AND dataset_versions.status = 'SUBMITTED'
RETURNING
    *;

-- name: UnpublishLifecycleDatasetVersion :one
UPDATE
    dataset_versions
SET
    STATUS = 'ARCHIVED',
    archived_by = sqlc.arg(actor_id),
    archived_at = NOW()
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'PUBLISHED'
RETURNING
    *;

-- name: DiscardLifecycleDatasetVersion :one
DELETE FROM
    dataset_versions
WHERE
    id = sqlc.arg(id)
    AND STATUS IN ('DRAFT', 'REJECTED')
RETURNING
    dataset_id;

-- name: DeleteEmptyLifecycleDataset :exec
DELETE FROM
    datasets
WHERE
    datasets.id = sqlc.arg(id)
    AND NOT EXISTS (
        SELECT
            1
        FROM
            dataset_versions
        WHERE
            dataset_id = datasets.id
    );

-- name: GetLifecycleDataset :one
SELECT
    id,
    source_iri,
    group_id,
    created_by,
    created_at
FROM
    datasets
WHERE
    id = sqlc.arg(id);

-- name: EnsureLifecycleDatasetVersion :one
SELECT
    id
FROM
    dataset_versions
WHERE
    id = sqlc.arg(version_id)
    AND dataset_id = sqlc.arg(dataset_id) FOR
UPDATE
;

-- name: GetPublishedLifecycleDatasetVersion :one
SELECT
    *
FROM
    dataset_versions
WHERE
    dataset_id = sqlc.arg(dataset_id)
    AND STATUS = 'PUBLISHED';

-- name: GetActiveLifecycleDatasetVersion :one
SELECT
    *
FROM
    dataset_versions
WHERE
    dataset_id = sqlc.arg(dataset_id)
    AND STATUS IN ('DRAFT', 'SUBMITTED', 'REJECTED')
ORDER BY
    created_at DESC,
    id DESC
LIMIT
    1;

-- name: GetLatestArchivedLifecycleDatasetVersion :one
SELECT
    *
FROM
    dataset_versions
WHERE
    dataset_versions.dataset_id = sqlc.arg(dataset_id)
    AND dataset_versions.STATUS = 'ARCHIVED'
    AND NOT EXISTS (
        SELECT
            1
        FROM
            dataset_versions AS published
        WHERE
            published.dataset_id = sqlc.arg(dataset_id)
            AND published.status = 'PUBLISHED'
    )
ORDER BY
    archived_at DESC NULLS LAST,
    id DESC
LIMIT
    1;

-- name: CloneLifecycleDatasetVersion :one
WITH source AS (
    SELECT
        *
    FROM
        dataset_versions
    WHERE
        dataset_versions.id = sqlc.arg(source_version_id)
        AND STATUS = sqlc.arg(source_status)::resource_status
),
inserted AS (
    INSERT INTO
        dataset_versions (
            dataset_id,
            title,
            description,
            keywords,
            accrual_periodicity,
            quality_annotation,
            access_right,
            source_created_at,
            source_updated_at,
            spatial_point,
            spatial_polygon,
            temporal_start,
            temporal_end,
            created_by
        )
    SELECT
        sqlc.arg(target_dataset_id),
        title,
        description,
        keywords,
        accrual_periodicity,
        quality_annotation,
        access_right,
        source_created_at,
        source_updated_at,
        spatial_point,
        spatial_polygon,
        temporal_start,
        temporal_end,
        sqlc.arg(actor_id)
    FROM
        source
    RETURNING
        *
),
identifiers AS (
    INSERT INTO
        identifiers (source_iri, TYPE, value, dataset_version_id)
    SELECT
        source_iri,
        TYPE,
        value,
        (
            SELECT
                id
            FROM
                inserted
        )
    FROM
        identifiers
    WHERE
        dataset_version_id = sqlc.arg(source_version_id)
),
providers AS (
    INSERT INTO
        dataset_version_providers (dataset_version_id, organization_id)
    SELECT
        (
            SELECT
                id
            FROM
                inserted
        ),
        organization_id
    FROM
        dataset_version_providers
    WHERE
        dataset_version_id = sqlc.arg(source_version_id)
),
contacts AS (
    INSERT INTO
        dataset_version_contact_points (dataset_version_id, contact_point_id)
    SELECT
        (
            SELECT
                id
            FROM
                inserted
        ),
        contact_point_id
    FROM
        dataset_version_contact_points
    WHERE
        dataset_version_id = sqlc.arg(source_version_id)
),
categories AS (
    INSERT INTO
        dataset_version_categories (dataset_version_id, group_id, category_id)
    SELECT
        (
            SELECT
                id
            FROM
                inserted
        ),
        sqlc.arg(target_group_id),
        category_id
    FROM
        dataset_version_categories
    WHERE
        dataset_version_id = sqlc.arg(source_version_id)
        AND sqlc.arg(copy_categories)::boolean
),
web_services AS (
    INSERT INTO
        web_services (
            id,
            dataset_version_id,
            name,
            description,
            template_url,
            format,
            provider_organization_id,
            documentation_title,
            documentation_description,
            documentation_uri,
            spatial_point,
            spatial_polygon,
            temporal_start,
            temporal_end
        )
    SELECT
        uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.id::text
        ),
        (
            SELECT
                id
            FROM
                inserted
        ),
        name,
        description,
        template_url,
        format,
        provider_organization_id,
        documentation_title,
        documentation_description,
        documentation_uri,
        spatial_point,
        spatial_polygon,
        temporal_start,
        temporal_end
    FROM
        web_services AS source
    WHERE
        dataset_version_id = sqlc.arg(source_version_id)
    RETURNING
        id,
        dataset_version_id
),
web_service_contacts AS (
    INSERT INTO
        web_service_contact_points (web_service_id, contact_point_id)
    SELECT
        target.id,
        link.contact_point_id
    FROM
        web_service_contact_points AS link
        JOIN web_services AS source ON source.id = link.web_service_id
        JOIN web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.id::text
        )
        AND target.dataset_version_id = (
            SELECT
                id
            FROM
                inserted
        )
    WHERE
        source.dataset_version_id = sqlc.arg(source_version_id)
),
web_service_parameters AS (
    INSERT INTO
        web_service_parameters (
            web_service_id,
            kind,
            variable,
            property,
            label,
            required,
            multiple_values,
            READ_ONLY,
            config
        )
    SELECT
        target.id,
        parameter.kind,
        parameter.variable,
        parameter.property,
        parameter.label,
        parameter.required,
        parameter.multiple_values,
        parameter.read_only,
        parameter.config
    FROM
        web_service_parameters AS parameter
        JOIN web_services AS source ON source.id = parameter.web_service_id
        JOIN web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.id::text
        )
        AND target.dataset_version_id = (
            SELECT
                id
            FROM
                inserted
        )
    WHERE
        source.dataset_version_id = sqlc.arg(source_version_id)
),
distributions AS (
    INSERT INTO
        distributions (
            dataset_version_id,
            title,
            description,
            license,
            source_issued_at,
            source_modified_at,
            byte_size,
            TYPE,
            downloadable_file_url,
            downloadable_file_format,
            web_service_id
        )
    SELECT
        (
            SELECT
                id
            FROM
                inserted
        ),
        distribution.title,
        distribution.description,
        distribution.license,
        distribution.source_issued_at,
        distribution.source_modified_at,
        distribution.byte_size,
        distribution.type,
        distribution.downloadable_file_url,
        distribution.downloadable_file_format,
        target.id
    FROM
        distributions AS distribution
        LEFT JOIN web_services AS source ON source.id = distribution.web_service_id
        LEFT JOIN web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.id::text
        )
        AND target.dataset_version_id = (
            SELECT
                id
            FROM
                inserted
        )
    WHERE
        distribution.dataset_version_id = sqlc.arg(source_version_id)
)
SELECT
    id,
    dataset_id,
    STATUS,
    revision,
    title,
    description,
    keywords,
    accrual_periodicity,
    quality_annotation,
    access_right,
    source_created_at,
    source_updated_at,
    spatial_point,
    spatial_polygon,
    temporal_start,
    temporal_end,
    created_by,
    created_at,
    updated_at,
    submitted_by,
    submitted_at,
    published_by,
    published_at,
    archived_by,
    archived_at,
    rejected_by,
    rejected_at,
    rejection_note
FROM
    inserted;

-- name: AddLifecycleDatasetIdentifier :exec
INSERT INTO
    identifiers (source_iri, TYPE, value, dataset_version_id)
VALUES
    (
        sqlc.arg(source_iri),
        sqlc.arg(TYPE),
        sqlc.arg(value),
        sqlc.arg(dataset_version_id)
    );

-- name: AddLifecycleDatasetProvider :exec
INSERT INTO
    dataset_version_providers (dataset_version_id, organization_id)
VALUES
    (
        sqlc.arg(dataset_version_id),
        sqlc.arg(organization_id)
    );

-- name: AddLifecycleDatasetContactPoint :exec
INSERT INTO
    dataset_version_contact_points (dataset_version_id, contact_point_id)
VALUES
    (
        sqlc.arg(dataset_version_id),
        sqlc.arg(contact_point_id)
    );

-- name: AddLifecycleDatasetCategory :exec
INSERT INTO
    dataset_version_categories (dataset_version_id, group_id, category_id)
SELECT
    sqlc.arg(dataset_version_id),
    dataset.group_id,
    sqlc.arg(category_id)
FROM
    datasets AS dataset
    JOIN dataset_versions AS version ON version.dataset_id = dataset.id
WHERE
    version.id = sqlc.arg(dataset_version_id);

-- name: AddLifecycleWebService :one
INSERT INTO
    web_services (
        dataset_version_id,
        name,
        description,
        template_url,
        format,
        provider_organization_id,
        documentation_title,
        documentation_description,
        documentation_uri,
        spatial_point,
        spatial_polygon,
        temporal_start,
        temporal_end
    )
VALUES
    (
        sqlc.arg(dataset_version_id),
        sqlc.arg(name),
        sqlc.arg(description),
        sqlc.arg(template_url),
        sqlc.arg(format),
        sqlc.narg(provider_organization_id),
        sqlc.narg(documentation_title),
        sqlc.narg(documentation_description),
        sqlc.narg(documentation_uri),
        CASE
            WHEN sqlc.narg(spatial_point)::text IS NULL THEN NULL
            ELSE ST_SetSRID(
                ST_GeomFromGeoJSON(sqlc.narg(spatial_point)::text),
                4326
            )
        END,
        CASE
            WHEN sqlc.narg(spatial_polygon)::text IS NULL THEN NULL
            ELSE ST_SetSRID(
                ST_GeomFromGeoJSON(sqlc.narg(spatial_polygon)::text),
                4326
            )
        END,
        sqlc.narg(temporal_start),
        sqlc.narg(temporal_end)
    )
RETURNING
    *;

-- name: AddLifecycleWebServiceContactPoint :exec
INSERT INTO
    web_service_contact_points (web_service_id, contact_point_id)
VALUES
    (
        sqlc.arg(web_service_id),
        sqlc.arg(contact_point_id)
    );

-- name: AddLifecycleWebServiceParameter :exec
INSERT INTO
    web_service_parameters (
        web_service_id,
        kind,
        variable,
        property,
        label,
        required,
        multiple_values,
        READ_ONLY,
        config
    )
VALUES
    (
        sqlc.arg(web_service_id),
        sqlc.arg(kind),
        sqlc.arg(variable),
        sqlc.arg(property),
        sqlc.arg(label),
        sqlc.arg(required),
        sqlc.arg(multiple_values),
        sqlc.arg(READ_ONLY),
        sqlc.arg(config)
    );

-- name: AddLifecycleDistribution :exec
INSERT INTO
    distributions (
        dataset_version_id,
        title,
        description,
        license,
        source_issued_at,
        source_modified_at,
        byte_size,
        TYPE,
        downloadable_file_url,
        downloadable_file_format,
        web_service_id
    )
VALUES
    (
        sqlc.arg(dataset_version_id),
        sqlc.arg(title),
        sqlc.arg(description),
        sqlc.narg(license),
        sqlc.narg(source_issued_at),
        sqlc.narg(source_modified_at),
        sqlc.narg(byte_size),
        sqlc.arg(TYPE),
        sqlc.narg(downloadable_file_url),
        sqlc.narg(downloadable_file_format),
        sqlc.narg(web_service_id)
    );

-- name: DeleteLifecycleDatasetVersionChildren :exec
WITH deleted_distributions AS (
    DELETE FROM
        distributions
    WHERE
        dataset_version_id = sqlc.arg(dataset_version_id)
),
deleted_web_services AS (
    DELETE FROM
        web_services
    WHERE
        dataset_version_id = sqlc.arg(dataset_version_id)
),
deleted_providers AS (
    DELETE FROM
        dataset_version_providers
    WHERE
        dataset_version_id = sqlc.arg(dataset_version_id)
),
deleted_contacts AS (
    DELETE FROM
        dataset_version_contact_points
    WHERE
        dataset_version_id = sqlc.arg(dataset_version_id)
),
deleted_categories AS (
    DELETE FROM
        dataset_version_categories
    WHERE
        dataset_version_id = sqlc.arg(dataset_version_id)
)
DELETE FROM
    identifiers AS identifier
WHERE
    identifier.dataset_version_id = sqlc.arg(dataset_version_id);

-- name: GetLifecycleDatasetVersion :one
SELECT
    version.id,
    version.dataset_id,
    version.status,
    version.revision,
    version.title,
    version.description,
    version.keywords,
    version.accrual_periodicity,
    version.quality_annotation,
    version.access_right,
    version.source_created_at,
    version.source_updated_at,
    COALESCE(ST_AsGeoJSON(version.spatial_point), '')::text AS spatial_point_geojson,
    COALESCE(ST_AsGeoJSON(version.spatial_polygon), '')::text AS spatial_polygon_geojson,
    version.temporal_start,
    version.temporal_end,
    version.created_by,
    version.created_at,
    version.updated_at,
    version.submitted_by,
    version.submitted_at,
    version.published_by,
    version.published_at,
    version.archived_by,
    version.archived_at,
    version.rejected_by,
    version.rejected_at,
    version.rejection_note,
    dataset.source_iri,
    sqlc.embed(owning_group)
FROM
    dataset_versions AS version
    JOIN datasets AS dataset ON dataset.id = version.dataset_id
    JOIN groups AS owning_group ON owning_group.id = dataset.group_id
WHERE
    version.id = sqlc.arg(version_id)
    AND version.dataset_id = sqlc.arg(dataset_id);

-- name: GetLifecycleDatasetGroup :one
SELECT
    sqlc.embed(owning_group)
FROM
    datasets AS dataset
    JOIN groups AS owning_group ON owning_group.id = dataset.group_id
WHERE
    dataset.id = sqlc.arg(dataset_id);

-- name: ListLifecycleDatasetIdentifiers :many
SELECT
    sqlc.embed(identifier)
FROM
    identifiers AS identifier
WHERE
    dataset_version_id = sqlc.arg(dataset_version_id)
ORDER BY
    id;

-- name: ListLifecycleDatasetProviders :many
SELECT
    organization.id,
    organization.source_iri,
    organization.name,
    organization.street,
    organization.locality,
    organization.postal_code,
    organization.country,
    organization.logo_url,
    organization.page_url,
    organization.email,
    organization.lei_code
FROM
    dataset_version_providers AS link
    JOIN organizations AS organization ON organization.id = link.organization_id
WHERE
    link.dataset_version_id = sqlc.arg(dataset_version_id)
ORDER BY
    organization.id;

-- name: ListLifecycleOrganizations :many
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
    id = ANY(sqlc.arg(organization_ids)::uuid [])
ORDER BY
    id;

-- name: ListLifecycleOrganizationContactPoints :many
SELECT
    link.organization_id,
    sqlc.embed(contact)
FROM
    organization_contact_points AS link
    JOIN contact_points AS contact ON contact.id = link.contact_point_id
WHERE
    link.organization_id = ANY(sqlc.arg(organization_ids)::uuid [])
ORDER BY
    link.organization_id,
    contact.id;

-- name: ListLifecycleDatasetContactPoints :many
SELECT
    contact.id,
    contact.source_iri,
    contact.email,
    contact.role
FROM
    dataset_version_contact_points AS link
    JOIN contact_points AS contact ON contact.id = link.contact_point_id
WHERE
    link.dataset_version_id = sqlc.arg(dataset_version_id)
ORDER BY
    contact.id;

-- name: ListLifecycleDatasetCategories :many
SELECT
    category.id,
    category.source_iri,
    category.name,
    category.group_id
FROM
    dataset_version_categories AS link
    JOIN categories AS category ON category.id = link.category_id
WHERE
    link.dataset_version_id = sqlc.arg(dataset_version_id)
ORDER BY
    category.id;

-- name: ListLifecycleWebServices :many
SELECT
    sqlc.embed(service),
    COALESCE(ST_AsGeoJSON(service.spatial_point), '')::text AS spatial_point_geojson,
    COALESCE(ST_AsGeoJSON(service.spatial_polygon), '')::text AS spatial_polygon_geojson
FROM
    web_services AS service
WHERE
    service.dataset_version_id = sqlc.arg(dataset_version_id)
ORDER BY
    service.id;

-- name: ListLifecycleWebServiceContactPoints :many
SELECT
    link.web_service_id,
    sqlc.embed(contact)
FROM
    web_service_contact_points AS link
    JOIN contact_points AS contact ON contact.id = link.contact_point_id
WHERE
    link.web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    link.web_service_id,
    contact.id;

-- name: ListLifecycleWebServiceParameters :many
SELECT
    id,
    web_service_id,
    kind,
    variable,
    property,
    label,
    required,
    multiple_values,
    READ_ONLY,
    config
FROM
    web_service_parameters
WHERE
    web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    web_service_id,
    id;

-- name: ListLifecycleDistributions :many
SELECT
    *
FROM
    distributions
WHERE
    dataset_version_id = sqlc.arg(dataset_version_id)
ORDER BY
    id;
