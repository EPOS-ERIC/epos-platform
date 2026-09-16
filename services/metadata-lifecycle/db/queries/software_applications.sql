-- name: CreateLifecycleSoftwareApplication :one
INSERT INTO
    software_applications (group_id, created_by)
VALUES
    (
        sqlc.arg(group_id),
        sqlc.arg(created_by)
    )
RETURNING
    *;

-- name: CreateLifecycleSoftwareApplicationVersion :one
INSERT INTO
    software_application_versions (
        software_application_id,
        name,
        description,
        main_entity_of_page,
        license,
        version,
        keywords,
        download_url,
        requirements,
        operating_system,
        created_by
    )
VALUES
    (
        sqlc.arg(software_application_id),
        sqlc.arg(name),
        sqlc.arg(description),
        sqlc.narg(main_entity_of_page),
        sqlc.narg(license),
        sqlc.narg(version),
        sqlc.arg(keywords),
        sqlc.narg(download_url),
        sqlc.narg(requirements),
        sqlc.narg(operating_system),
        sqlc.arg(created_by)
    )
RETURNING
    *;

-- name: ReplaceLifecycleSoftwareApplicationDraft :one
UPDATE
    software_application_versions
SET
    name = sqlc.arg(name),
    description = sqlc.arg(description),
    main_entity_of_page = sqlc.narg(main_entity_of_page),
    license = sqlc.narg(license),
    version = sqlc.narg(version),
    keywords = sqlc.arg(keywords),
    download_url = sqlc.narg(download_url),
    requirements = sqlc.narg(requirements),
    operating_system = sqlc.narg(operating_system),
    revision = revision + 1
WHERE
    id = sqlc.arg(id)
    AND software_application_id = sqlc.arg(software_application_id)
    AND STATUS = 'DRAFT'
    AND revision = sqlc.arg(revision)
RETURNING
    *;

-- name: SubmitLifecycleSoftwareApplicationVersion :one
UPDATE
    software_application_versions
SET
    STATUS = 'SUBMITTED',
    submitted_by = sqlc.arg(actor_id),
    submitted_at = NOW()
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'DRAFT'
RETURNING
    *;

-- name: WithdrawLifecycleSoftwareApplicationVersion :one
UPDATE
    software_application_versions
SET
    STATUS = 'DRAFT'
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'SUBMITTED'
RETURNING
    *;

-- name: RejectLifecycleSoftwareApplicationVersion :one
UPDATE
    software_application_versions
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

-- name: ReopenLifecycleSoftwareApplicationVersion :one
UPDATE
    software_application_versions
SET
    STATUS = 'DRAFT'
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'REJECTED'
RETURNING
    *;

-- name: PublishLifecycleSoftwareApplicationVersion :one
UPDATE
    software_application_versions
SET
    STATUS = 'PUBLISHED',
    published_by = sqlc.arg(actor_id),
    published_at = NOW(),
    rejected_by = NULL,
    rejected_at = NULL,
    rejection_note = NULL
WHERE
    software_application_versions.id = sqlc.arg(version_id)
    AND software_application_versions.status = 'SUBMITTED'
RETURNING
    *;

-- name: ArchivePublishedLifecycleSoftwareApplicationVersion :exec
UPDATE
    software_application_versions
SET
    STATUS = 'ARCHIVED',
    archived_by = sqlc.arg(actor_id),
    archived_at = NOW()
WHERE
    software_application_id = sqlc.arg(software_application_id)
    AND STATUS = 'PUBLISHED';

-- name: UnpublishLifecycleSoftwareApplicationVersion :one
UPDATE
    software_application_versions
SET
    STATUS = 'ARCHIVED',
    archived_by = sqlc.arg(actor_id),
    archived_at = NOW()
WHERE
    id = sqlc.arg(id)
    AND STATUS = 'PUBLISHED'
RETURNING
    *;

-- name: DiscardLifecycleSoftwareApplicationVersion :one
DELETE FROM
    software_application_versions
WHERE
    id = sqlc.arg(id)
    AND STATUS IN ('DRAFT', 'REJECTED')
RETURNING
    software_application_id;

-- name: DeleteEmptyLifecycleSoftwareApplication :exec
DELETE FROM
    software_applications AS application
WHERE
    application.id = sqlc.arg(id)
    AND NOT EXISTS (
        SELECT
            1
        FROM
            software_application_versions
        WHERE
            software_application_id = application.id
    );

-- name: EnsureLifecycleSoftwareApplicationVersion :one
SELECT
    id
FROM
    software_application_versions
WHERE
    id = sqlc.arg(version_id)
    AND software_application_id = sqlc.arg(software_application_id) FOR
UPDATE
;

-- name: GetLifecycleSoftwareApplication :one
SELECT
    *
FROM
    software_applications
WHERE
    id = sqlc.arg(id);

-- name: GetPublishedLifecycleSoftwareApplicationVersion :one
SELECT
    *
FROM
    software_application_versions
WHERE
    software_application_id = sqlc.arg(software_application_id)
    AND STATUS = 'PUBLISHED';

-- name: GetActiveLifecycleSoftwareApplicationVersion :one
SELECT
    *
FROM
    software_application_versions
WHERE
    software_application_id = sqlc.arg(software_application_id)
    AND STATUS IN ('DRAFT', 'SUBMITTED', 'REJECTED')
ORDER BY
    created_at DESC,
    id DESC
LIMIT
    1;

-- name: GetLatestArchivedLifecycleSoftwareApplicationVersion :one
SELECT
    *
FROM
    software_application_versions AS version
WHERE
    version.software_application_id = sqlc.arg(software_application_id)
    AND version.status = 'ARCHIVED'
    AND NOT EXISTS (
        SELECT
            1
        FROM
            software_application_versions AS published
        WHERE
            published.software_application_id = version.software_application_id
            AND published.status = 'PUBLISHED'
    )
ORDER BY
    archived_at DESC NULLS LAST,
    id DESC
LIMIT
    1;

-- name: CloneLifecycleSoftwareApplicationVersion :one
WITH source AS (
    SELECT
        *
    FROM
        software_application_versions
    WHERE
        software_application_versions.id = sqlc.arg(source_version_id)
        AND software_application_versions.status = sqlc.arg(source_status)::resource_status
),
inserted AS (
    INSERT INTO
        software_application_versions (
            software_application_id,
            name,
            description,
            main_entity_of_page,
            license,
            version,
            keywords,
            download_url,
            requirements,
            operating_system,
            created_by
        )
    SELECT
        sqlc.arg(target_software_application_id),
        name,
        description,
        main_entity_of_page,
        license,
        version,
        keywords,
        download_url,
        requirements,
        operating_system,
        sqlc.arg(actor_id)
    FROM
        source
    RETURNING
        *
),
identifiers AS (
    INSERT INTO
        identifiers (
            TYPE,
            value,
            software_application_version_id
        )
    SELECT
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
        software_application_version_id = sqlc.arg(source_version_id)
),
creators AS (
    INSERT INTO
        software_application_version_creators (software_application_version_id, organization_id)
    SELECT
        (
            SELECT
                id
            FROM
                inserted
        ),
        organization_id
    FROM
        software_application_version_creators
    WHERE
        software_application_version_id = sqlc.arg(source_version_id)
),
contact_points AS (
    INSERT INTO
        software_application_version_contact_points (
            software_application_version_id,
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
        software_application_version_contact_points
    WHERE
        software_application_version_id = sqlc.arg(source_version_id)
),
categories AS (
    INSERT INTO
        software_application_version_categories (
            software_application_version_id,
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
        software_application_version_categories
    WHERE
        software_application_version_id = sqlc.arg(source_version_id)
        AND sqlc.arg(copy_categories)::boolean
)
SELECT
    id,
    software_application_id,
    STATUS,
    revision,
    name,
    description,
    main_entity_of_page,
    license,
    version,
    keywords,
    download_url,
    requirements,
    operating_system,
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

-- name: AddLifecycleSoftwareApplicationIdentifier :exec
INSERT INTO
    identifiers (
        TYPE,
        value,
        software_application_version_id
    )
VALUES
    (
        sqlc.arg(TYPE),
        sqlc.arg(value),
        sqlc.arg(software_application_version_id)
    );

-- name: AddLifecycleSoftwareApplicationCreator :exec
INSERT INTO
    software_application_version_creators (software_application_version_id, organization_id)
VALUES
    (
        sqlc.arg(software_application_version_id),
        sqlc.arg(organization_id)
    );

-- name: AddLifecycleSoftwareApplicationContactPoint :exec
INSERT INTO
    software_application_version_contact_points (
        software_application_version_id,
        contact_point_id
    )
VALUES
    (
        sqlc.arg(software_application_version_id),
        sqlc.arg(contact_point_id)
    );

-- name: AddLifecycleSoftwareApplicationCategory :exec
INSERT INTO
    software_application_version_categories (
        software_application_version_id,
        group_id,
        category_id
    )
SELECT
    sqlc.arg(software_application_version_id),
    application.group_id,
    sqlc.arg(category_id)
FROM
    software_applications AS application
    JOIN software_application_versions AS version ON version.software_application_id = application.id
WHERE
    version.id = sqlc.arg(software_application_version_id);

-- name: DeleteLifecycleSoftwareApplicationVersionChildren :exec
WITH deleted_creators AS (
    DELETE FROM
        software_application_version_creators
    WHERE
        software_application_version_id = sqlc.arg(version_id)
),
deleted_contact_points AS (
    DELETE FROM
        software_application_version_contact_points
    WHERE
        software_application_version_id = sqlc.arg(version_id)
),
deleted_categories AS (
    DELETE FROM
        software_application_version_categories
    WHERE
        software_application_version_id = sqlc.arg(version_id)
)
DELETE FROM
    identifiers AS identifier
WHERE
    identifier.software_application_version_id = sqlc.arg(version_id);

-- name: GetLifecycleSoftwareApplicationVersion :one
SELECT
    sqlc.embed(version),
    sqlc.embed(application),
    sqlc.embed(owning_group)
FROM
    software_application_versions AS version
    JOIN software_applications AS application ON application.id = version.software_application_id
    JOIN groups AS owning_group ON owning_group.id = application.group_id
WHERE
    version.id = sqlc.arg(version_id)
    AND version.software_application_id = sqlc.arg(software_application_id);

-- name: ListLifecycleSoftwareApplicationVersionSummaries :many
SELECT
    version.id,
    version.software_application_id,
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
    software_application_versions AS version
WHERE
    version.software_application_id = sqlc.arg(software_application_id)
ORDER BY
    version.created_at,
    version.id;

-- name: ListLifecycleSoftwareApplicationIdentifiers :many
SELECT
    sqlc.embed(identifier)
FROM
    identifiers AS identifier
WHERE
    software_application_version_id = sqlc.arg(version_id)
ORDER BY
    id;

-- name: ListLifecycleSoftwareApplicationCreators :many
SELECT
    organization.*
FROM
    software_application_version_creators AS link
    JOIN organizations AS organization ON organization.id = link.organization_id
WHERE
    link.software_application_version_id = sqlc.arg(version_id)
ORDER BY
    organization.id;

-- name: ListLifecycleSoftwareApplicationContactPoints :many
SELECT
    sqlc.embed(contact)
FROM
    software_application_version_contact_points AS link
    JOIN contact_points AS contact ON contact.id = link.contact_point_id
WHERE
    link.software_application_version_id = sqlc.arg(version_id)
ORDER BY
    contact.id;

-- name: ListLifecycleSoftwareApplicationCategories :many
SELECT
    sqlc.embed(category),
    sqlc.embed(owning_group)
FROM
    software_application_version_categories AS link
    JOIN categories AS category ON category.id = link.category_id
    JOIN groups AS owning_group ON owning_group.id = category.group_id
WHERE
    link.software_application_version_id = sqlc.arg(version_id)
ORDER BY
    category.id;
