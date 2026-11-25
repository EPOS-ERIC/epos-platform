-- name: CreateLifecycleDataset :one
INSERT INTO
    datasets (group_id, created_by)
VALUES
    (
        sqlc.arg(group_id),
        sqlc.arg(created_by)
    )
RETURNING
    id,
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
        conforms_to,
        provenance,
        TYPE,
        version,
        spatial_resolution_in_meters,
        temporal_resolution,
        access_right,
        source_created_at,
        source_updated_at,
        spatial_coverage,
        temporal_start,
        temporal_end,
        geological_temporal_start_mya,
        geological_temporal_end_mya,
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
        sqlc.narg(conforms_to),
        sqlc.narg(provenance),
        sqlc.narg(TYPE),
        sqlc.narg(version),
        sqlc.narg(spatial_resolution_in_meters),
        sqlc.narg(temporal_resolution),
        sqlc.narg(access_right),
        sqlc.narg(source_created_at),
        sqlc.narg(source_updated_at),
        sqlc.narg(spatial_coverage),
        sqlc.narg(temporal_start),
        sqlc.narg(temporal_end),
        sqlc.narg(geological_temporal_start_mya),
        sqlc.narg(geological_temporal_end_mya),
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
    conforms_to = sqlc.narg(conforms_to),
    provenance = sqlc.narg(provenance),
    TYPE = sqlc.narg(TYPE),
    version = sqlc.narg(version),
    spatial_resolution_in_meters = sqlc.narg(spatial_resolution_in_meters),
    temporal_resolution = sqlc.narg(temporal_resolution),
    access_right = sqlc.narg(access_right),
    source_created_at = sqlc.narg(source_created_at),
    source_updated_at = sqlc.narg(source_updated_at),
    spatial_coverage = sqlc.narg(spatial_coverage),
    temporal_start = sqlc.narg(temporal_start),
    temporal_end = sqlc.narg(temporal_end),
    geological_temporal_start_mya = sqlc.narg(geological_temporal_start_mya),
    geological_temporal_end_mya = sqlc.narg(geological_temporal_end_mya),
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

-- name: ArchivePublishedLifecycleDatasetVersion :exec
UPDATE
    dataset_versions
SET
    STATUS = 'ARCHIVED',
    archived_by = sqlc.arg(actor_id),
    archived_at = NOW()
WHERE
    dataset_id = sqlc.arg(dataset_id)
    AND STATUS = 'PUBLISHED';

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

-- name: GetDistributionLifecycleDatasetVersion :one
SELECT
    dataset_versions.dataset_id,
    dataset_versions.id AS version_id
FROM
    distributions
    JOIN dataset_versions ON dataset_versions.id = distributions.dataset_version_id
WHERE
    distributions.id = sqlc.arg(distribution_id);

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
        dataset_versions.*,
        dataset_versions.dataset_id = sqlc.arg(target_dataset_id) AS same_dataset
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
            conforms_to,
            provenance,
            TYPE,
            version,
            spatial_resolution_in_meters,
            temporal_resolution,
            access_right,
            source_created_at,
            source_updated_at,
            spatial_coverage,
            temporal_start,
            temporal_end,
            geological_temporal_start_mya,
            geological_temporal_end_mya,
            created_by
        )
    SELECT
        sqlc.arg(target_dataset_id),
        title,
        description,
        keywords,
        accrual_periodicity,
        quality_annotation,
        conforms_to,
        provenance,
        TYPE,
        version,
        spatial_resolution_in_meters,
        temporal_resolution,
        access_right,
        source_created_at,
        source_updated_at,
        spatial_coverage,
        temporal_start,
        temporal_end,
        geological_temporal_start_mya,
        geological_temporal_end_mya,
        sqlc.arg(actor_id)
    FROM
        source
    RETURNING
        *
),
identifiers AS (
    INSERT INTO
        identifiers (resource_id, TYPE, value, dataset_version_id)
    SELECT
        CASE
            WHEN (
                SELECT
                    same_dataset
                FROM
                    source
            ) THEN resource_id
            ELSE uuid_generate_v4()
        END,
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
        group_id,
        category_id
    FROM
        dataset_version_categories
    WHERE
        dataset_version_id = sqlc.arg(source_version_id)
        AND (
            SELECT
                same_dataset
            FROM
                source
        )
),
inserted_web_services AS (
    INSERT INTO
        web_services (
            id,
            resource_id,
            dataset_version_id,
            TYPE,
            visible_on,
            provider_organization_id,
            documentation_uri,
            keywords,
            access_rights,
            license
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
        CASE
            WHEN (
                SELECT
                    same_dataset
                FROM
                    source
            ) THEN source.resource_id
            ELSE uuid_generate_v4()
        END,
        (
            SELECT
                id
            FROM
                inserted
        ),
        TYPE,
        visible_on,
        provider_organization_id,
        documentation_uri,
        keywords,
        access_rights,
        license
    FROM
        web_services AS source
    WHERE
        dataset_version_id = sqlc.arg(source_version_id)
    RETURNING
        id
),
web_service_categories_copy AS (
    INSERT INTO
        web_service_categories (web_service_id, group_id, category_id)
    SELECT
        target.id,
        link.group_id,
        link.category_id
    FROM
        web_service_categories AS link
        JOIN inserted_web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            link.web_service_id::text
        )
    WHERE
        (
            SELECT
                same_dataset
            FROM
                source
        )
),
inserted_generic_web_services AS (
    INSERT INTO
        generic_web_services (web_service_id, template_url, method, format)
    SELECT
        target.id,
        source.template_url,
        source.method,
        source.format
    FROM
        generic_web_services AS source
        JOIN inserted_web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.web_service_id::text
        )
    RETURNING
        web_service_id
),
inserted_wfs_services AS (
    INSERT INTO
        wfs_services (
            web_service_id,
            get_feature_template_url,
            version,
            feature_types,
            default_feature_type,
            output_formats,
            default_output_format,
            result_limit,
            result_limit_editable
        )
    SELECT
        target.id,
        source.get_feature_template_url,
        source.version,
        source.feature_types,
        source.default_feature_type,
        source.output_formats,
        source.default_output_format,
        source.result_limit,
        source.result_limit_editable
    FROM
        wfs_services AS source
        JOIN inserted_web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.web_service_id::text
        )
    RETURNING
        web_service_id
),
inserted_wms_services AS (
    INSERT INTO
        wms_services (
            web_service_id,
            capabilities_url,
            default_layer,
            transparency_editable,
            default_transparency
        )
    SELECT
        target.id,
        source.capabilities_url,
        source.default_layer,
        source.transparency_editable,
        source.default_transparency
    FROM
        wms_services AS source
        JOIN inserted_web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.web_service_id::text
        )
    RETURNING
        web_service_id
),
inserted_wms_layers AS (
    INSERT INTO
        wms_layers (
            resource_id,
            web_service_id,
            name,
            position,
            default_style,
            styles
        )
    SELECT
        CASE
            WHEN (
                SELECT
                    same_dataset
                FROM
                    source
            ) THEN source.resource_id
            ELSE uuid_generate_v4()
        END,
        target.web_service_id,
        source.name,
        source.position,
        source.default_style,
        source.styles
    FROM
        wms_layers AS source
        JOIN inserted_wms_services AS target ON target.web_service_id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.web_service_id::text
        )
    RETURNING
        web_service_id,
        name
),
inserted_wmts_services AS (
    INSERT INTO
        wmts_services (web_service_id, capabilities_url, default_layer)
    SELECT
        target.id,
        source.capabilities_url,
        source.default_layer
    FROM
        wmts_services AS source
        JOIN inserted_web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.web_service_id::text
        )
    RETURNING
        web_service_id
),
inserted_wmts_layers AS (
    INSERT INTO
        wmts_layers (
            resource_id,
            web_service_id,
            name,
            position,
            default_style,
            styles
        )
    SELECT
        CASE
            WHEN (
                SELECT
                    same_dataset
                FROM
                    source
            ) THEN source.resource_id
            ELSE uuid_generate_v4()
        END,
        target.web_service_id,
        source.name,
        source.position,
        source.default_style,
        source.styles
    FROM
        wmts_layers AS source
        JOIN inserted_wmts_services AS target ON target.web_service_id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.web_service_id::text
        )
    RETURNING
        web_service_id,
        name
),
inserted_web_service_bodies AS (
    INSERT INTO
        web_service_bodies (
            id,
            web_service_id,
            content_type,
            template,
            body_required
        )
    SELECT
        uuid_generate_v5(target.id, source.id::text),
        target.id,
        source.content_type,
        source.template,
        source.body_required
    FROM
        web_service_bodies AS source
        JOIN web_services AS source_service ON source_service.id = source.web_service_id
        JOIN inserted_web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source_service.id::text
        )
        JOIN inserted_generic_web_services AS generic_target ON generic_target.web_service_id = target.id
    WHERE
        source_service.dataset_version_id = sqlc.arg(source_version_id)
    RETURNING
        id,
        web_service_id
),
web_service_contacts AS (
    INSERT INTO
        web_service_contact_points (web_service_id, contact_point_id)
    SELECT
        target.id,
        link.contact_point_id
    FROM
        web_service_contact_points AS link
        JOIN inserted_web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            link.web_service_id::text
        )
),
inserted_parameters AS (
    INSERT INTO
        parameters (
            id,
            resource_id,
            web_service_id,
            wfs_service_id,
            wms_get_map_service_id,
            wms_get_feature_info_service_id,
            wmts_get_tile_service_id,
            wmts_get_feature_info_service_id,
            web_service_body_id,
            position,
            name,
            label,
            description,
            required,
            READ_ONLY,
            property,
            role,
            location,
            explode,
            schema_kind,
            template_parameter_id
        )
    SELECT
        uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            parameter.id::text
        ),
        CASE
            WHEN (
                SELECT
                    same_dataset
                FROM
                    source
            ) THEN parameter.resource_id
            ELSE uuid_generate_v4()
        END,
        CASE
            WHEN parameter.web_service_id IS NOT NULL THEN generic_target.web_service_id
            ELSE NULL
        END,
        CASE
            WHEN parameter.wfs_service_id IS NOT NULL THEN wfs_target.web_service_id
            ELSE NULL
        END,
        CASE
            WHEN parameter.wms_get_map_service_id IS NOT NULL THEN wms_target.web_service_id
            ELSE NULL
        END,
        CASE
            WHEN parameter.wms_get_feature_info_service_id IS NOT NULL THEN wms_target.web_service_id
            ELSE NULL
        END,
        CASE
            WHEN parameter.wmts_get_tile_service_id IS NOT NULL THEN wmts_target.web_service_id
            ELSE NULL
        END,
        CASE
            WHEN parameter.wmts_get_feature_info_service_id IS NOT NULL THEN wmts_target.web_service_id
            ELSE NULL
        END,
        target_body.id,
        parameter.position,
        parameter.name,
        parameter.label,
        parameter.description,
        parameter.required,
        parameter.read_only,
        parameter.property,
        parameter.role,
        parameter.location,
        parameter.explode,
        parameter.schema_kind,
        NULL
    FROM
        parameters AS parameter
        LEFT JOIN web_service_bodies AS source_body ON source_body.id = parameter.web_service_body_id
        LEFT JOIN inserted_web_services AS target_service ON target_service.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            COALESCE(
                parameter.web_service_id,
                parameter.wfs_service_id,
                parameter.wms_get_map_service_id,
                parameter.wms_get_feature_info_service_id,
                parameter.wmts_get_tile_service_id,
                parameter.wmts_get_feature_info_service_id,
                source_body.web_service_id
            )::text
        )
        LEFT JOIN inserted_generic_web_services AS generic_target ON generic_target.web_service_id = target_service.id
        LEFT JOIN inserted_wfs_services AS wfs_target ON wfs_target.web_service_id = target_service.id
        LEFT JOIN inserted_wms_services AS wms_target ON wms_target.web_service_id = target_service.id
        LEFT JOIN inserted_wmts_services AS wmts_target ON wmts_target.web_service_id = target_service.id
        LEFT JOIN inserted_web_service_bodies AS target_body ON target_body.id = uuid_generate_v5(
            target_service.id,
            source_body.id::text
        )
    WHERE
        parameter.template_parameter_id IS NULL
        AND (
            generic_target.web_service_id IS NOT NULL
            OR wfs_target.web_service_id IS NOT NULL
            OR wms_target.web_service_id IS NOT NULL
            OR wmts_target.web_service_id IS NOT NULL
            OR target_body.id IS NOT NULL
        )
    RETURNING
        id
),
inserted_parameter_templates AS (
    INSERT INTO
        parameter_templates (parameter_id, template)
    SELECT
        uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source_template.parameter_id::text
        ),
        source_template.template
    FROM
        parameter_templates AS source_template
        JOIN inserted_parameters AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source_template.parameter_id::text
        )
    RETURNING
        parameter_id
),
template_parameters AS (
    INSERT INTO
        parameters (
            id,
            resource_id,
            template_parameter_id,
            position,
            name,
            label,
            description,
            required,
            READ_ONLY,
            property,
            role,
            schema_kind
        )
    SELECT
        uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source_parameter.id::text
        ),
        CASE
            WHEN (
                SELECT
                    same_dataset
                FROM
                    source
            ) THEN source_parameter.resource_id
            ELSE uuid_generate_v4()
        END,
        uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source_parameter.template_parameter_id::text
        ),
        source_parameter.position,
        source_parameter.name,
        source_parameter.label,
        source_parameter.description,
        source_parameter.required,
        source_parameter.read_only,
        source_parameter.property,
        source_parameter.role,
        source_parameter.schema_kind
    FROM
        parameters AS source_parameter
        JOIN inserted_parameter_templates AS target_template ON target_template.parameter_id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source_parameter.template_parameter_id::text
        )
    RETURNING
        id
),
parameter_map AS (
    SELECT
        source.id AS source_id,
        target.id
    FROM
        parameters AS source
        JOIN inserted_parameters AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.id::text
        )
    UNION
    ALL
    SELECT
        source.id AS source_id,
        target.id
    FROM
        parameters AS source
        JOIN template_parameters AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            source.id::text
        )
),
inserted_string_parameter_schemas AS (
    INSERT INTO
        string_parameter_schemas (
            parameter_id,
            pattern,
            default_value,
            enum,
            multiple_values
        )
    SELECT
        target.id,
        source.pattern,
        source.default_value,
        source.enum,
        source.multiple_values
    FROM
        string_parameter_schemas AS source
        JOIN parameter_map AS target ON target.source_id = source.parameter_id
),
inserted_boolean_parameter_schemas AS (
    INSERT INTO
        boolean_parameter_schemas (parameter_id, default_value)
    SELECT
        target.id,
        source.default_value
    FROM
        boolean_parameter_schemas AS source
        JOIN parameter_map AS target ON target.source_id = source.parameter_id
),
inserted_integer_parameter_schemas AS (
    INSERT INTO
        integer_parameter_schemas (
            parameter_id,
            minimum,
            maximum,
            default_value,
            enum,
            multiple_values
        )
    SELECT
        target.id,
        source.minimum,
        source.maximum,
        source.default_value,
        source.enum,
        source.multiple_values
    FROM
        integer_parameter_schemas AS source
        JOIN parameter_map AS target ON target.source_id = source.parameter_id
),
inserted_float_parameter_schemas AS (
    INSERT INTO
        float_parameter_schemas (
            parameter_id,
            minimum,
            maximum,
            default_value,
            enum,
            multiple_values
        )
    SELECT
        target.id,
        source.minimum,
        source.maximum,
        source.default_value,
        source.enum,
        source.multiple_values
    FROM
        float_parameter_schemas AS source
        JOIN parameter_map AS target ON target.source_id = source.parameter_id
),
inserted_datetime_parameter_schemas AS (
    INSERT INTO
        datetime_parameter_schemas (
            parameter_id,
            minimum,
            maximum,
            default_value,
            layout
        )
    SELECT
        target.id,
        source.minimum,
        source.maximum,
        source.default_value,
        source.layout
    FROM
        datetime_parameter_schemas AS source
        JOIN parameter_map AS target ON target.source_id = source.parameter_id
),
distributions AS (
    INSERT INTO
        distributions (
            id,
            resource_id,
            dataset_version_id,
            title,
            description,
            license,
            byte_size,
            TYPE,
            downloadable_file_url,
            downloadable_file_format,
            web_service_id
        )
    SELECT
        uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            distribution.id::text
        ),
        CASE
            WHEN (
                SELECT
                    same_dataset
                FROM
                    source
            ) THEN distribution.resource_id
            ELSE uuid_generate_v4()
        END,
        (
            SELECT
                id
            FROM
                inserted
        ),
        distribution.title,
        distribution.description,
        distribution.license,
        distribution.byte_size,
        distribution.type,
        distribution.downloadable_file_url,
        distribution.downloadable_file_format,
        target.id
    FROM
        distributions AS distribution
        LEFT JOIN inserted_web_services AS target ON target.id = uuid_generate_v5(
            (
                SELECT
                    id
                FROM
                    inserted
            ),
            distribution.web_service_id::text
        )
    WHERE
        distribution.dataset_version_id = sqlc.arg(source_version_id)
)
SELECT
    id,
    dataset_id
FROM
    inserted;

-- name: ListClonedLifecycleDistributions :many
SELECT
    source.id AS source_id,
    uuid_generate_v5(
        sqlc.arg(target_version_id)::uuid,
        source.id::text
    )::uuid AS target_id
FROM
    distributions AS source
WHERE
    source.dataset_version_id = sqlc.arg(source_version_id);

-- name: AddLifecycleDatasetIdentifier :exec
INSERT INTO
    identifiers (id, resource_id, TYPE, value, dataset_version_id)
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(resource_id),
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

-- name: AddLifecycleWebService :exec
INSERT INTO
    web_services (
        id,
        resource_id,
        dataset_version_id,
        TYPE,
        visible_on,
        provider_organization_id,
        documentation_uri,
        keywords,
        access_rights,
        license
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(resource_id),
        sqlc.arg(dataset_version_id),
        sqlc.arg(TYPE),
        sqlc.arg(visible_on),
        sqlc.narg(provider_organization_id),
        sqlc.narg(documentation_uri),
        sqlc.arg(keywords),
        sqlc.narg(access_rights),
        sqlc.narg(license)
    );

-- name: AddLifecycleWebServiceCategories :exec
INSERT INTO
    web_service_categories (web_service_id, group_id, category_id)
SELECT
    service.id,
    dataset.group_id,
    category.id
FROM
    web_services AS service
    JOIN dataset_versions AS version ON version.id = service.dataset_version_id
    JOIN datasets AS dataset ON dataset.id = version.dataset_id
    CROSS JOIN unnest(sqlc.arg(category_ids)::uuid []) AS category(id)
WHERE
    service.id = sqlc.arg(web_service_id);

-- name: AddLifecycleGenericWebService :exec
INSERT INTO
    generic_web_services (web_service_id, template_url, method, format)
VALUES
    (
        sqlc.arg(web_service_id),
        sqlc.arg(template_url),
        sqlc.arg(method),
        sqlc.arg(format)
    );

-- name: AddLifecycleWFSService :exec
INSERT INTO
    wfs_services (
        web_service_id,
        get_feature_template_url,
        version,
        feature_types,
        default_feature_type,
        output_formats,
        default_output_format,
        result_limit,
        result_limit_editable
    )
VALUES
    (
        sqlc.arg(web_service_id),
        sqlc.arg(get_feature_template_url),
        sqlc.arg(version),
        sqlc.arg(feature_types),
        sqlc.arg(default_feature_type),
        sqlc.arg(output_formats),
        sqlc.arg(default_output_format),
        sqlc.narg(result_limit),
        sqlc.arg(result_limit_editable)
    );

-- name: AddLifecycleWMSService :exec
INSERT INTO
    wms_services (
        web_service_id,
        capabilities_url,
        default_layer,
        transparency_editable,
        default_transparency
    )
VALUES
    (
        sqlc.arg(web_service_id),
        sqlc.arg(capabilities_url),
        sqlc.arg(default_layer),
        sqlc.arg(transparency_editable),
        sqlc.arg(default_transparency)
    );

-- name: AddLifecycleWMSLayer :exec
INSERT INTO
    wms_layers (
        id,
        resource_id,
        web_service_id,
        name,
        position,
        default_style,
        styles
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(resource_id),
        sqlc.arg(web_service_id),
        sqlc.arg(name),
        sqlc.arg(position),
        sqlc.arg(default_style),
        sqlc.arg(styles)
    );

-- name: AddLifecycleWMTSService :exec
INSERT INTO
    wmts_services (web_service_id, capabilities_url, default_layer)
VALUES
    (
        sqlc.arg(web_service_id),
        sqlc.arg(capabilities_url),
        sqlc.arg(default_layer)
    );

-- name: AddLifecycleWMTSLayer :exec
INSERT INTO
    wmts_layers (
        id,
        resource_id,
        web_service_id,
        name,
        position,
        default_style,
        styles
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(resource_id),
        sqlc.arg(web_service_id),
        sqlc.arg(name),
        sqlc.arg(position),
        sqlc.arg(default_style),
        sqlc.arg(styles)
    );

-- name: AddLifecycleWebServiceBody :one
INSERT INTO
    web_service_bodies (
        web_service_id,
        content_type,
        template,
        body_required
    )
VALUES
    (
        sqlc.arg(web_service_id),
        sqlc.arg(content_type),
        sqlc.arg(template),
        sqlc.arg(body_required)
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

-- name: AddLifecycleParameter :one
INSERT INTO
    parameters (
        id,
        resource_id,
        web_service_id,
        wfs_service_id,
        wms_get_map_service_id,
        wms_get_feature_info_service_id,
        wmts_get_tile_service_id,
        wmts_get_feature_info_service_id,
        web_service_body_id,
        template_parameter_id,
        position,
        name,
        label,
        description,
        required,
        READ_ONLY,
        property,
        role,
        location,
        explode,
        schema_kind
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(resource_id),
        sqlc.narg(web_service_id),
        sqlc.narg(wfs_service_id),
        sqlc.narg(wms_get_map_service_id),
        sqlc.narg(wms_get_feature_info_service_id),
        sqlc.narg(wmts_get_tile_service_id),
        sqlc.narg(wmts_get_feature_info_service_id),
        sqlc.narg(web_service_body_id),
        sqlc.narg(template_parameter_id),
        sqlc.arg(position),
        sqlc.arg(name),
        sqlc.arg(label),
        sqlc.narg(description),
        sqlc.arg(required),
        sqlc.arg(READ_ONLY),
        sqlc.narg(property),
        sqlc.narg(role),
        sqlc.narg(location),
        sqlc.arg(explode),
        sqlc.arg(schema_kind)
    )
RETURNING
    *;

-- name: AddLifecycleStringParameterSchema :exec
INSERT INTO
    string_parameter_schemas (
        parameter_id,
        pattern,
        default_value,
        enum,
        multiple_values
    )
VALUES
    (
        sqlc.arg(parameter_id),
        sqlc.narg(pattern),
        sqlc.narg(default_value),
        sqlc.narg(enum),
        sqlc.arg(multiple_values)
    );

-- name: AddLifecycleBooleanParameterSchema :exec
INSERT INTO
    boolean_parameter_schemas (parameter_id, default_value)
VALUES
    (sqlc.arg(parameter_id), sqlc.narg(default_value));

-- name: AddLifecycleIntegerParameterSchema :exec
INSERT INTO
    integer_parameter_schemas (
        parameter_id,
        minimum,
        maximum,
        default_value,
        enum,
        multiple_values
    )
VALUES
    (
        sqlc.arg(parameter_id),
        sqlc.narg(minimum),
        sqlc.narg(maximum),
        sqlc.narg(default_value),
        sqlc.narg(enum),
        sqlc.arg(multiple_values)
    );

-- name: AddLifecycleFloatParameterSchema :exec
INSERT INTO
    float_parameter_schemas (
        parameter_id,
        minimum,
        maximum,
        default_value,
        enum,
        multiple_values
    )
VALUES
    (
        sqlc.arg(parameter_id),
        sqlc.narg(minimum),
        sqlc.narg(maximum),
        sqlc.narg(default_value),
        sqlc.narg(enum),
        sqlc.arg(multiple_values)
    );

-- name: AddLifecycleDatetimeParameterSchema :exec
INSERT INTO
    datetime_parameter_schemas (
        parameter_id,
        minimum,
        maximum,
        default_value,
        layout
    )
VALUES
    (
        sqlc.arg(parameter_id),
        sqlc.narg(minimum),
        sqlc.narg(maximum),
        sqlc.narg(default_value),
        sqlc.narg(layout)
    );

-- name: AddLifecycleParameterTemplate :exec
INSERT INTO
    parameter_templates (parameter_id, template)
VALUES
    (
        sqlc.arg(parameter_id),
        sqlc.arg(template)
    );

-- name: AddLifecycleDistribution :exec
INSERT INTO
    distributions (
        id,
        resource_id,
        dataset_version_id,
        title,
        description,
        license,
        byte_size,
        TYPE,
        downloadable_file_url,
        downloadable_file_format,
        web_service_id
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(resource_id),
        sqlc.arg(dataset_version_id),
        sqlc.arg(title),
        sqlc.arg(description),
        sqlc.narg(license),
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
    version.conforms_to,
    version.provenance,
    version.type,
    version.version,
    version.spatial_resolution_in_meters,
    version.temporal_resolution,
    version.access_right,
    version.source_created_at,
    version.source_updated_at,
    version.spatial_coverage,
    version.temporal_start,
    version.temporal_end,
    version.geological_temporal_start_mya,
    version.geological_temporal_end_mya,
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
    sqlc.embed(owning_group)
FROM
    dataset_versions AS version
    JOIN datasets AS dataset ON dataset.id = version.dataset_id
    JOIN groups AS owning_group ON owning_group.id = dataset.group_id
WHERE
    version.id = sqlc.arg(version_id)
    AND version.dataset_id = sqlc.arg(dataset_id);

-- name: ListLifecycleDatasetVersionSummaries :many
SELECT
    version.id,
    version.dataset_id,
    version.status,
    version.revision,
    version.title,
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
    version.rejection_note
FROM
    dataset_versions AS version
WHERE
    version.dataset_id = sqlc.arg(dataset_id)
ORDER BY
    version.created_at,
    version.id;

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
    organization.remote_nid,
    organization.name,
    organization.street,
    organization.locality,
    organization.postal_code,
    organization.country,
    organization.logo_url,
    organization.page_url,
    organization.email
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
    id = ANY(sqlc.arg(organization_ids)::uuid [])
ORDER BY
    id;

-- name: ListLifecycleDatasetContactPoints :many
SELECT
    contact.id,
    contact.remote_nid,
    contact.email
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
    category.name
FROM
    dataset_version_categories AS link
    JOIN categories AS category ON category.id = link.category_id
WHERE
    link.dataset_version_id = sqlc.arg(dataset_version_id)
ORDER BY
    category.id;

-- name: ListLifecycleWebServices :many
SELECT
    sqlc.embed(service)
FROM
    web_services AS service
WHERE
    service.dataset_version_id = sqlc.arg(dataset_version_id)
ORDER BY
    service.id;

-- name: ListLifecycleWebServiceCategories :many
SELECT
    link.web_service_id,
    category.id,
    category.name
FROM
    web_service_categories AS link
    JOIN categories AS category ON category.id = link.category_id
WHERE
    link.web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    link.web_service_id,
    category.id;

-- name: ListLifecycleGenericWebServices :many
SELECT
    configuration.web_service_id,
    configuration.template_url,
    configuration.method,
    configuration.format
FROM
    generic_web_services AS configuration
WHERE
    configuration.web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    configuration.web_service_id;

-- name: ListLifecycleWFSServices :many
SELECT
    web_service_id,
    get_feature_template_url,
    version,
    feature_types,
    default_feature_type,
    output_formats,
    default_output_format,
    result_limit,
    result_limit_editable
FROM
    wfs_services
WHERE
    web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    web_service_id;

-- name: ListLifecycleWMSServices :many
SELECT
    configuration.web_service_id,
    configuration.capabilities_url,
    configuration.default_layer,
    configuration.transparency_editable,
    configuration.default_transparency
FROM
    wms_services AS configuration
WHERE
    configuration.web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    configuration.web_service_id;

-- name: ListLifecycleWMSLayers :many
SELECT
    *
FROM
    wms_layers
WHERE
    web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    web_service_id,
    position;

-- name: ListLifecycleWMTSServices :many
SELECT
    *
FROM
    wmts_services
WHERE
    web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    web_service_id;

-- name: ListLifecycleWMTSLayers :many
SELECT
    *
FROM
    wmts_layers
WHERE
    web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    web_service_id,
    position;

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
    parameter.*
FROM
    parameters AS parameter
    LEFT JOIN web_service_bodies AS body ON body.id = parameter.web_service_body_id
WHERE
    parameter.web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
    OR parameter.wfs_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
    OR parameter.wms_get_map_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
    OR parameter.wms_get_feature_info_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
    OR parameter.wmts_get_tile_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
    OR parameter.wmts_get_feature_info_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
    OR body.web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    COALESCE(
        parameter.web_service_id,
        parameter.wfs_service_id,
        parameter.wms_get_map_service_id,
        parameter.wms_get_feature_info_service_id,
        parameter.wmts_get_tile_service_id,
        parameter.wmts_get_feature_info_service_id,
        body.web_service_id
    ),
    parameter.position,
    parameter.name;

-- name: ListLifecycleStringParameterSchemas :many
SELECT
    sqlc.embed(schema)
FROM
    string_parameter_schemas AS schema
WHERE
    schema.parameter_id = ANY(sqlc.arg(parameter_ids)::uuid []);

-- name: ListLifecycleBooleanParameterSchemas :many
SELECT
    sqlc.embed(schema)
FROM
    boolean_parameter_schemas AS schema
WHERE
    schema.parameter_id = ANY(sqlc.arg(parameter_ids)::uuid []);

-- name: ListLifecycleIntegerParameterSchemas :many
SELECT
    sqlc.embed(schema)
FROM
    integer_parameter_schemas AS schema
WHERE
    schema.parameter_id = ANY(sqlc.arg(parameter_ids)::uuid []);

-- name: ListLifecycleFloatParameterSchemas :many
SELECT
    sqlc.embed(schema)
FROM
    float_parameter_schemas AS schema
WHERE
    schema.parameter_id = ANY(sqlc.arg(parameter_ids)::uuid []);

-- name: ListLifecycleDatetimeParameterSchemas :many
SELECT
    sqlc.embed(schema)
FROM
    datetime_parameter_schemas AS schema
WHERE
    schema.parameter_id = ANY(sqlc.arg(parameter_ids)::uuid []);

-- name: ListLifecycleWebServiceBodies :many
SELECT
    *
FROM
    web_service_bodies
WHERE
    web_service_id = ANY(sqlc.arg(web_service_ids)::uuid [])
ORDER BY
    web_service_id;

-- name: ListLifecycleParameterTemplates :many
SELECT
    *
FROM
    parameter_templates
WHERE
    parameter_id = ANY(sqlc.arg(parameter_ids)::uuid [])
ORDER BY
    parameter_id;

-- name: ListLifecycleTemplateParameters :many
SELECT
    *
FROM
    parameters
WHERE
    template_parameter_id = ANY(sqlc.arg(parameter_ids)::uuid [])
ORDER BY
    template_parameter_id,
    position,
    name;

-- name: ListLifecycleDistributions :many
SELECT
    *
FROM
    distributions
WHERE
    dataset_version_id = sqlc.arg(dataset_version_id)
ORDER BY
    id;
