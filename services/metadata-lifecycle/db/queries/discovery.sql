-- name: SearchDiscoveryDatasets :many
SELECT
    dataset.id AS resource_id,
    version.id AS version_id,
    dataset.group_id,
    version.status,
    version.title AS name,
    version.updated_at,
    resource_group.concept,
    version.created_by AS created_by_id,
    concat_ws(' ', creator.given_name, creator.family_name) AS created_by_name
FROM
    datasets AS dataset
    JOIN dataset_versions AS version ON version.dataset_id = dataset.id
    JOIN groups AS resource_group ON resource_group.id = dataset.group_id
    JOIN users AS creator ON creator.id = version.created_by
WHERE
    version.status = ANY(
        sqlc.arg(versioning_statuses)::text []::resource_status []
    )
    AND (
        cardinality(sqlc.arg(concepts)::text []) = 0
        OR resource_group.concept = ANY(
            sqlc.arg(concepts)::text []::top_level_concept []
        )
    )
    AND (
        sqlc.narg(group_id)::uuid IS NULL
        OR dataset.group_id = sqlc.narg(group_id)::uuid
    )
    AND (
        sqlc.narg(query)::text IS NULL
        OR lower(
            concat_ws(
                ' ',
                version.title,
                version.description,
                version.keywords
            )
        ) LIKE '%' || lower(sqlc.narg(query)::text) || '%'
    )
    AND (
        sqlc.narg(spatial_filter)::text IS NULL
        OR ST_Intersects(
            version.spatial_point,
            ST_GeomFromText(sqlc.narg(spatial_filter)::text, 4326)
        )
        OR ST_Intersects(
            version.spatial_polygon,
            ST_GeomFromText(sqlc.narg(spatial_filter)::text, 4326)
        )
    )
    AND (
        sqlc.narg(date_from)::timestamptz IS NULL
        OR version.temporal_end IS NULL
        OR version.temporal_end >= sqlc.narg(date_from)::timestamptz
    )
    AND (
        sqlc.narg(date_to)::timestamptz IS NULL
        OR version.temporal_start IS NULL
        OR version.temporal_start <= sqlc.narg(date_to)::timestamptz
    )
    AND (
        cardinality(sqlc.arg(provider_ids)::uuid []) = 0
        OR EXISTS (
            SELECT
                1
            FROM
                dataset_version_providers AS provider
            WHERE
                provider.dataset_version_id = version.id
                AND provider.organization_id = ANY(sqlc.arg(provider_ids)::uuid [])
        )
    )
    AND (
        cardinality(sqlc.arg(keywords)::text []) = 0
        OR NOT EXISTS (
            SELECT
                1
            FROM
                unnest(sqlc.arg(keywords)::text []) AS requested(keyword)
            WHERE
                NOT EXISTS (
                    SELECT
                        1
                    FROM
                        unnest(string_to_array(version.keywords, ',')) AS stored(keyword)
                    WHERE
                        lower(trim(stored.keyword)) = lower(trim(requested.keyword))
                )
        )
    )
    AND (
        sqlc.narg(editor_id)::text IS NULL
        OR version.created_by = sqlc.narg(editor_id)::text
    )
    AND (
        cardinality(sqlc.arg(resource_types)::text []) = 0
        OR 'dataset' = ANY(sqlc.arg(resource_types)::text [])
    )
ORDER BY
    lower(version.title),
    dataset.id,
    version.id;

-- name: SearchDiscoveryApplications :many
SELECT
    resource.id AS resource_id,
    version.id AS version_id,
    resource.group_id,
    version.status,
    version.name,
    version.updated_at,
    resource_group.concept,
    version.created_by AS created_by_id,
    concat_ws(' ', creator.given_name, creator.family_name) AS created_by_name
FROM
    software_applications AS resource
    JOIN software_application_versions AS version ON version.software_application_id = resource.id
    JOIN groups AS resource_group ON resource_group.id = resource.group_id
    JOIN users AS creator ON creator.id = version.created_by
WHERE
    version.status = ANY(
        sqlc.arg(versioning_statuses)::text []::resource_status []
    )
    AND (
        cardinality(sqlc.arg(concepts)::text []) = 0
        OR resource_group.concept = ANY(
            sqlc.arg(concepts)::text []::top_level_concept []
        )
    )
    AND (
        sqlc.narg(group_id)::uuid IS NULL
        OR resource.group_id = sqlc.narg(group_id)::uuid
    )
    AND (
        sqlc.narg(query)::text IS NULL
        OR lower(
            concat_ws(
                ' ',
                version.name,
                version.description,
                version.keywords
            )
        ) LIKE '%' || lower(sqlc.narg(query)::text) || '%'
    )
    AND (
        cardinality(sqlc.arg(provider_ids)::uuid []) = 0
        OR EXISTS (
            SELECT
                1
            FROM
                software_application_version_creators AS creator
            WHERE
                creator.software_application_version_id = version.id
                AND creator.organization_id = ANY(sqlc.arg(provider_ids)::uuid [])
        )
    )
    AND (
        cardinality(sqlc.arg(keywords)::text []) = 0
        OR NOT EXISTS (
            SELECT
                1
            FROM
                unnest(sqlc.arg(keywords)::text []) AS requested(keyword)
            WHERE
                NOT EXISTS (
                    SELECT
                        1
                    FROM
                        unnest(string_to_array(version.keywords, ',')) AS stored(keyword)
                    WHERE
                        lower(trim(stored.keyword)) = lower(trim(requested.keyword))
                )
        )
    )
    AND (
        sqlc.narg(editor_id)::text IS NULL
        OR version.created_by = sqlc.narg(editor_id)::text
    )
    AND (
        cardinality(sqlc.arg(resource_types)::text []) = 0
        OR 'software-application' = ANY(sqlc.arg(resource_types)::text [])
    );

-- name: SearchDiscoverySourceCodes :many
SELECT
    resource.id AS resource_id,
    version.id AS version_id,
    resource.group_id,
    version.status,
    version.name,
    version.updated_at,
    resource_group.concept,
    version.created_by AS created_by_id,
    concat_ws(' ', creator.given_name, creator.family_name) AS created_by_name
FROM
    software_source_codes AS resource
    JOIN software_source_code_versions AS version ON version.software_source_code_id = resource.id
    JOIN groups AS resource_group ON resource_group.id = resource.group_id
    JOIN users AS creator ON creator.id = version.created_by
WHERE
    version.status = ANY(
        sqlc.arg(versioning_statuses)::text []::resource_status []
    )
    AND (
        cardinality(sqlc.arg(concepts)::text []) = 0
        OR resource_group.concept = ANY(
            sqlc.arg(concepts)::text []::top_level_concept []
        )
    )
    AND (
        sqlc.narg(group_id)::uuid IS NULL
        OR resource.group_id = sqlc.narg(group_id)::uuid
    )
    AND (
        sqlc.narg(query)::text IS NULL
        OR lower(
            concat_ws(
                ' ',
                version.name,
                version.description,
                version.keywords
            )
        ) LIKE '%' || lower(sqlc.narg(query)::text) || '%'
    )
    AND (
        cardinality(sqlc.arg(provider_ids)::uuid []) = 0
        OR EXISTS (
            SELECT
                1
            FROM
                software_source_code_version_creators AS creator
            WHERE
                creator.software_source_code_version_id = version.id
                AND creator.organization_id = ANY(sqlc.arg(provider_ids)::uuid [])
        )
    )
    AND (
        cardinality(sqlc.arg(keywords)::text []) = 0
        OR NOT EXISTS (
            SELECT
                1
            FROM
                unnest(sqlc.arg(keywords)::text []) AS requested(keyword)
            WHERE
                NOT EXISTS (
                    SELECT
                        1
                    FROM
                        unnest(string_to_array(version.keywords, ',')) AS stored(keyword)
                    WHERE
                        lower(trim(stored.keyword)) = lower(trim(requested.keyword))
                )
        )
    )
    AND (
        sqlc.narg(editor_id)::text IS NULL
        OR version.created_by = sqlc.narg(editor_id)::text
    )
    AND (
        cardinality(sqlc.arg(resource_types)::text []) = 0
        OR 'software-source-code' = ANY(sqlc.arg(resource_types)::text [])
    );
