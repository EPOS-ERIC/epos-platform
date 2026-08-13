-- name: GetAuthorizationUser :one
SELECT
    id,
    STATUS
FROM
    users
WHERE
    id = $1;

-- name: GetPermissionScope :one
SELECT
    scope
FROM
    permissions
WHERE
    KEY = $1;

-- name: IsGlobalAdmin :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            user_global_roles
        WHERE
            user_id = $1
            AND role_key = 'ADMIN'
    ) AS allowed;

-- name: HasGlobalPermission :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            global_permission_grants
        WHERE
            user_id = $1
            AND permission_key = $2
    ) AS allowed;

-- name: IsActiveMember :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            group_memberships
        WHERE
            user_id = $1
            AND group_id = $2
            AND STATUS = 'ACTIVE'
    ) AS active;

-- name: HasAnyActiveMembership :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            group_memberships
        WHERE
            user_id = $1
            AND STATUS = 'ACTIVE'
    ) AS active;

-- name: GetDatasetVersionAuthorization :one
SELECT
    version.dataset_id,
    dataset.group_id,
    version.status
FROM
    dataset_versions AS version
    JOIN datasets AS dataset ON dataset.id = version.dataset_id
WHERE
    version.id = sqlc.arg(version_id)
    AND version.dataset_id = sqlc.arg(dataset_id);

-- name: GetSoftwareApplicationVersionAuthorization :one
SELECT
    version.software_application_id,
    application.group_id,
    version.status
FROM
    software_application_versions AS version
    JOIN software_applications AS application ON application.id = version.software_application_id
WHERE
    version.id = sqlc.arg(version_id)
    AND version.software_application_id = sqlc.arg(software_application_id);

-- name: GetSoftwareSourceCodeVersionAuthorization :one
SELECT
    version.software_source_code_id,
    source_code.group_id,
    version.status
FROM
    software_source_code_versions AS version
    JOIN software_source_codes AS source_code ON source_code.id = version.software_source_code_id
WHERE
    version.id = sqlc.arg(version_id)
    AND version.software_source_code_id = sqlc.arg(software_source_code_id);

-- name: HasGroupPermission :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            group_permission_grants
        WHERE
            group_permission_grants.user_id = $1
            AND group_permission_grants.group_id = $2
            AND group_permission_grants.permission_key = $3
        UNION
        ALL
        SELECT
            1
        FROM
            group_membership_roles AS assignment
            JOIN role_permissions ON role_permissions.role_key = assignment.role_key
        WHERE
            assignment.user_id = $1
            AND assignment.group_id = $2
            AND role_permissions.permission_key = $3
    ) AS allowed;

-- name: GetDatasetGroup :one
SELECT
    group_id
FROM
    datasets
WHERE
    id = $1;

-- name: GetSoftwareApplicationGroup :one
SELECT
    group_id
FROM
    software_applications
WHERE
    id = $1;

-- name: GetSoftwareSourceCodeGroup :one
SELECT
    group_id
FROM
    software_source_codes
WHERE
    id = $1;

-- name: IsDatasetPublished :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            dataset_versions
        WHERE
            dataset_id = $1
            AND STATUS = 'PUBLISHED'
    ) AS published;

-- name: IsSoftwareApplicationPublished :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            software_application_versions
        WHERE
            software_application_id = $1
            AND STATUS = 'PUBLISHED'
    ) AS published;

-- name: IsSoftwareSourceCodePublished :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            software_source_code_versions
        WHERE
            software_source_code_id = $1
            AND STATUS = 'PUBLISHED'
    ) AS published;
