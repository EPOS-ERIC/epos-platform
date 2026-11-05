-- name: CreateLifecycleSoftwareSourceCode :one
INSERT INTO
    software_source_codes (group_id, created_by)
VALUES
    (
        sqlc.arg(group_id),
        sqlc.arg(created_by)
    )
RETURNING
    *;

-- name: CreateLifecycleSoftwareSourceCodeVersion :one
INSERT INTO
    software_source_code_versions (
        software_source_code_id,
        name,
        description,
        programming_language,
        keywords,
        repository,
        license,
        software_requirements,
        runtime_platform,
        main_entity_of_page,
        version,
        created_by
    )
VALUES
    (
        sqlc.arg(software_source_code_id),
        sqlc.arg(name),
        sqlc.arg(description),
        sqlc.narg(programming_language),
        sqlc.arg(keywords),
        sqlc.narg(repository),
        sqlc.narg(license),
        sqlc.narg(software_requirements),
        sqlc.narg(runtime_platform),
        sqlc.narg(main_entity_of_page),
        sqlc.narg(version),
        sqlc.arg(created_by)
    )
RETURNING
    *;

-- name: ReplaceLifecycleSoftwareSourceCodeDraft :one
UPDATE
    software_source_code_versions
SET
    name = sqlc.arg(name),
    description = sqlc.arg(description),
    programming_language = sqlc.narg(programming_language),
    keywords = sqlc.arg(keywords),
    repository = sqlc.narg(repository),
    license = sqlc.narg(license),
    software_requirements = sqlc.narg(software_requirements),
    runtime_platform = sqlc.narg(runtime_platform),
    main_entity_of_page = sqlc.narg(main_entity_of_page),
    version = sqlc.narg(version),
    revision = revision + 1
WHERE
    id = sqlc.arg(id)
    AND software_source_code_id = sqlc.arg(software_source_code_id)
    AND STATUS = 'DRAFT'
    AND revision = sqlc.arg(revision)
RETURNING
    *;

-- name: SubmitLifecycleSoftwareSourceCodeVersion :one
UPDATE
    software_source_code_versions
SET
    STATUS = 'SUBMITTED',
    submitted_by = sqlc.arg(actor_id),
    submitted_at = NOW()
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'DRAFT'
RETURNING
    *;

-- name: WithdrawLifecycleSoftwareSourceCodeVersion :one
UPDATE
    software_source_code_versions
SET
    STATUS = 'DRAFT'
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'SUBMITTED'
RETURNING
    *;

-- name: RejectLifecycleSoftwareSourceCodeVersion :one
UPDATE
    software_source_code_versions
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

-- name: ReopenLifecycleSoftwareSourceCodeVersion :one
UPDATE
    software_source_code_versions
SET
    STATUS = 'DRAFT'
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'REJECTED'
RETURNING
    *;

-- name: PublishLifecycleSoftwareSourceCodeVersion :one
UPDATE
    software_source_code_versions
SET
    STATUS = 'PUBLISHED',
    published_by = sqlc.arg(actor_id),
    published_at = NOW(),
    rejected_by = NULL,
    rejected_at = NULL,
    rejection_note = NULL
WHERE
    software_source_code_versions.id = sqlc.arg(version_id)
    AND software_source_code_versions.status = 'SUBMITTED'
RETURNING
    *;

-- name: ArchivePublishedLifecycleSoftwareSourceCodeVersion :exec
UPDATE
    software_source_code_versions
SET
    STATUS = 'ARCHIVED',
    archived_by = sqlc.arg(actor_id),
    archived_at = NOW()
WHERE
    software_source_code_id = sqlc.arg(software_source_code_id)
    AND STATUS = 'PUBLISHED';

-- name: UnpublishLifecycleSoftwareSourceCodeVersion :one
UPDATE
    software_source_code_versions
SET
    STATUS = 'ARCHIVED',
    archived_by = sqlc.arg(actor_id),
    archived_at = NOW()
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'PUBLISHED'
RETURNING
    *;

-- name: DiscardLifecycleSoftwareSourceCodeVersion :one
DELETE FROM
    software_source_code_versions
WHERE
    id = sqlc.arg(id)
    AND STATUS IN ('DRAFT', 'REJECTED')
RETURNING
    software_source_code_id;

-- name: DeleteEmptyLifecycleSoftwareSourceCode :exec
DELETE FROM
    software_source_codes AS source_code
WHERE
    source_code.id = sqlc.arg(id)
    AND NOT EXISTS (
        SELECT
            1
        FROM
            software_source_code_versions
        WHERE
            software_source_code_id = source_code.id
    );

-- name: EnsureLifecycleSoftwareSourceCodeVersion :one
SELECT
    id
FROM
    software_source_code_versions
WHERE
    id = sqlc.arg(version_id)
    AND software_source_code_id = sqlc.arg(software_source_code_id) FOR
UPDATE
;

-- name: GetLifecycleSoftwareSourceCode :one
SELECT
    *
FROM
    software_source_codes
WHERE
    id = sqlc.arg(id);

-- name: GetPublishedLifecycleSoftwareSourceCodeVersion :one
SELECT
    *
FROM
    software_source_code_versions
WHERE
    software_source_code_id = sqlc.arg(software_source_code_id)
    AND STATUS = 'PUBLISHED';

-- name: GetActiveLifecycleSoftwareSourceCodeVersion :one
SELECT
    *
FROM
    software_source_code_versions
WHERE
    software_source_code_id = sqlc.arg(software_source_code_id)
    AND STATUS IN ('DRAFT', 'SUBMITTED', 'REJECTED')
ORDER BY
    created_at DESC,
    id DESC
LIMIT
    1;

-- name: GetLatestArchivedLifecycleSoftwareSourceCodeVersion :one
SELECT
    *
FROM
    software_source_code_versions AS version
WHERE
    version.software_source_code_id = sqlc.arg(software_source_code_id)
    AND version.status = 'ARCHIVED'
    AND NOT EXISTS (
        SELECT
            1
        FROM
            software_source_code_versions AS published
        WHERE
            published.software_source_code_id = version.software_source_code_id
            AND published.status = 'PUBLISHED'
    )
ORDER BY
    archived_at DESC NULLS LAST,
    id DESC
LIMIT
    1;

-- name: CloneLifecycleSoftwareSourceCodeVersion :one
WITH source AS (
    SELECT
        *
    FROM
        software_source_code_versions
    WHERE
        software_source_code_versions.id = sqlc.arg(source_version_id)
        AND software_source_code_versions.status = sqlc.arg(source_status)::resource_status
),
inserted AS (
    INSERT INTO
        software_source_code_versions (
            software_source_code_id,
            name,
            description,
            programming_language,
            keywords,
            repository,
            license,
            software_requirements,
            runtime_platform,
            main_entity_of_page,
            version,
            created_by
        )
    SELECT
        sqlc.arg(target_software_source_code_id),
        name,
        description,
        programming_language,
        keywords,
        repository,
        license,
        software_requirements,
        runtime_platform,
        main_entity_of_page,
        version,
        sqlc.arg(actor_id)
    FROM
        source
    RETURNING
        *
),
identifiers AS (
    INSERT INTO
        identifiers (
            resource_id,
            TYPE,
            value,
            software_source_code_version_id
        )
    SELECT
        CASE
            WHEN (
                SELECT
                    software_source_code_id
                FROM
                    source
            ) = sqlc.arg(target_software_source_code_id) THEN resource_id
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
        software_source_code_version_id = sqlc.arg(source_version_id)
),
creators AS (
    INSERT INTO
        software_source_code_version_creators (software_source_code_version_id, organization_id)
    SELECT
        (
            SELECT
                id
            FROM
                inserted
        ),
        organization_id
    FROM
        software_source_code_version_creators
    WHERE
        software_source_code_version_id = sqlc.arg(source_version_id)
),
contact_points AS (
    INSERT INTO
        software_source_code_version_contact_points (
            software_source_code_version_id,
            contact_point_id
        )
    SELECT
        (
            SELECT
                id
            FROM
                inserted
        ),
        contact_point_id
    FROM
        software_source_code_version_contact_points
    WHERE
        software_source_code_version_id = sqlc.arg(source_version_id)
),
categories AS (
    INSERT INTO
        software_source_code_version_categories (
            software_source_code_version_id,
            group_id,
            category_id
        )
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
        software_source_code_version_categories
    WHERE
        software_source_code_version_id = sqlc.arg(source_version_id)
        AND sqlc.arg(copy_categories)::boolean
)
SELECT
    id,
    software_source_code_id,
    STATUS,
    revision,
    name,
    description,
    programming_language,
    keywords,
    repository,
    license,
    software_requirements,
    runtime_platform,
    main_entity_of_page,
    version,
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

-- name: AddLifecycleSoftwareSourceCodeIdentifier :exec
INSERT INTO
    identifiers (
        id,
        resource_id,
        TYPE,
        value,
        software_source_code_version_id
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(resource_id),
        sqlc.arg(TYPE),
        sqlc.arg(value),
        sqlc.arg(software_source_code_version_id)
    );

-- name: AddLifecycleSoftwareSourceCodeCreator :exec
INSERT INTO
    software_source_code_version_creators (software_source_code_version_id, organization_id)
VALUES
    (
        sqlc.arg(software_source_code_version_id),
        sqlc.arg(organization_id)
    );

-- name: AddLifecycleSoftwareSourceCodeContactPoint :exec
INSERT INTO
    software_source_code_version_contact_points (
        software_source_code_version_id,
        contact_point_id
    )
VALUES
    (
        sqlc.arg(software_source_code_version_id),
        sqlc.arg(contact_point_id)
    );

-- name: AddLifecycleSoftwareSourceCodeCategory :exec
INSERT INTO
    software_source_code_version_categories (
        software_source_code_version_id,
        group_id,
        category_id
    )
SELECT
    sqlc.arg(software_source_code_version_id),
    source_code.group_id,
    sqlc.arg(category_id)
FROM
    software_source_codes AS source_code
    JOIN software_source_code_versions AS version ON version.software_source_code_id = source_code.id
WHERE
    version.id = sqlc.arg(software_source_code_version_id);

-- name: DeleteLifecycleSoftwareSourceCodeVersionChildren :exec
WITH deleted_creators AS (
    DELETE FROM
        software_source_code_version_creators
    WHERE
        software_source_code_version_id = sqlc.arg(version_id)
),
deleted_contact_points AS (
    DELETE FROM
        software_source_code_version_contact_points
    WHERE
        software_source_code_version_id = sqlc.arg(version_id)
),
deleted_categories AS (
    DELETE FROM
        software_source_code_version_categories
    WHERE
        software_source_code_version_id = sqlc.arg(version_id)
)
DELETE FROM
    identifiers AS identifier
WHERE
    identifier.software_source_code_version_id = sqlc.arg(version_id);

-- name: GetLifecycleSoftwareSourceCodeVersion :one
SELECT
    sqlc.embed(version),
    sqlc.embed(source_code),
    sqlc.embed(owning_group)
FROM
    software_source_code_versions AS version
    JOIN software_source_codes AS source_code ON source_code.id = version.software_source_code_id
    JOIN groups AS owning_group ON owning_group.id = source_code.group_id
WHERE
    version.id = sqlc.arg(version_id)
    AND version.software_source_code_id = sqlc.arg(software_source_code_id);

-- name: ListLifecycleSoftwareSourceCodeVersionSummaries :many
SELECT
    version.id,
    version.software_source_code_id,
    version.status,
    version.revision,
    version.name,
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
    software_source_code_versions AS version
WHERE
    version.software_source_code_id = sqlc.arg(software_source_code_id)
ORDER BY
    version.created_at,
    version.id;

-- name: ListLifecycleSoftwareSourceCodeIdentifiers :many
SELECT
    sqlc.embed(identifier)
FROM
    identifiers AS identifier
WHERE
    software_source_code_version_id = sqlc.arg(version_id)
ORDER BY
    id;

-- name: ListLifecycleSoftwareSourceCodeCreators :many
SELECT
    organization.*
FROM
    software_source_code_version_creators AS link
    JOIN organizations AS organization ON organization.id = link.organization_id
WHERE
    link.software_source_code_version_id = sqlc.arg(version_id)
ORDER BY
    organization.id;

-- name: ListLifecycleSoftwareSourceCodeContactPoints :many
SELECT
    sqlc.embed(contact)
FROM
    software_source_code_version_contact_points AS link
    JOIN contact_points AS contact ON contact.id = link.contact_point_id
WHERE
    link.software_source_code_version_id = sqlc.arg(version_id)
ORDER BY
    contact.id;

-- name: ListLifecycleSoftwareSourceCodeCategories :many
SELECT
    sqlc.embed(category),
    sqlc.embed(owning_group)
FROM
    software_source_code_version_categories AS link
    JOIN categories AS category ON category.id = link.category_id
    JOIN groups AS owning_group ON owning_group.id = category.group_id
WHERE
    link.software_source_code_version_id = sqlc.arg(version_id)
ORDER BY
    category.id;
