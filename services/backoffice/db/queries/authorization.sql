-- name: GetAuthorizationUser :one
SELECT
    id,
    STATUS
FROM
    users
WHERE
    id = $1;

-- name: GetUserByID :one
SELECT
    id,
    email,
    given_name,
    family_name,
    STATUS,
    created_at,
    updated_at
FROM
    users
WHERE
    id = $1;

-- name: ListUsers :many
SELECT
    users.id,
    users.email,
    users.given_name,
    users.family_name,
    users.status
FROM
    users
ORDER BY
    users.family_name,
    users.given_name,
    users.id;

-- name: GetUserGlobalRoles :many
SELECT
    role_key
FROM
    user_global_roles
WHERE
    user_id = $1
ORDER BY
    role_key;

-- name: GetUserGlobalPermissions :many
SELECT
    role_permission.permission_key
FROM
    user_global_roles AS global_role
    JOIN role_permissions AS role_permission ON role_permission.role_key = global_role.role_key
WHERE
    global_role.user_id = $1
UNION
SELECT
    global_grant.permission_key
FROM
    global_permission_grants AS global_grant
WHERE
    global_grant.user_id = $1
UNION
SELECT
    permission.key
FROM
    permissions AS permission
WHERE
    permission.scope = 'GLOBAL'
    AND EXISTS (
        SELECT
            1
        FROM
            user_global_roles AS global_role
        WHERE
            global_role.user_id = $1
            AND global_role.role_key = 'ADMIN'
    )
ORDER BY
    permission_key;

-- name: GetUserMemberships :many
SELECT
    membership.group_id,
    groups.name AS group_name,
    membership.status,
    membership.created_at,
    membership.updated_at
FROM
    group_memberships AS membership
    JOIN groups ON groups.id = membership.group_id
WHERE
    membership.user_id = $1
ORDER BY
    groups.name,
    membership.group_id;

-- name: GetUserMembershipRoles :many
SELECT
    group_id,
    role_key
FROM
    group_membership_roles
WHERE
    user_id = $1
ORDER BY
    group_id,
    role_key;

-- name: GetUserMembershipPermissions :many
SELECT
    membership.group_id,
    role_permission.permission_key
FROM
    group_memberships AS membership
    JOIN group_membership_roles AS role_assignment ON role_assignment.user_id = membership.user_id
    AND role_assignment.group_id = membership.group_id
    JOIN role_permissions AS role_permission ON role_permission.role_key = role_assignment.role_key
WHERE
    membership.user_id = $1
    AND membership.status = 'ACTIVE'
UNION
SELECT
    direct_grant.group_id,
    direct_grant.permission_key
FROM
    group_permission_grants AS direct_grant
    JOIN group_memberships AS membership ON membership.user_id = direct_grant.user_id
    AND membership.group_id = direct_grant.group_id
WHERE
    direct_grant.user_id = $1
    AND membership.status = 'ACTIVE'
ORDER BY
    group_id,
    permission_key;

-- name: CreateUser :one
INSERT INTO
    users (
        id,
        email,
        given_name,
        family_name
    )
VALUES
    ($1, $2, $3, $4)
RETURNING
    id,
    email,
    given_name,
    family_name,
    STATUS,
    created_at,
    updated_at;

-- name: GrantGlobalPermission :exec
INSERT INTO
    global_permission_grants (user_id, permission_key, granted_by)
VALUES
    ($1, $2, $3) ON CONFLICT (user_id, permission_key) DO NOTHING;

-- name: RevokeGlobalPermission :exec
DELETE FROM
    global_permission_grants
WHERE
    user_id = $1
    AND permission_key = $2;

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
            global_permission_grants AS global_grant
        WHERE
            global_grant.user_id = $1
            AND global_grant.permission_key = $2
        UNION
        ALL
        SELECT
            1
        FROM
            user_global_roles AS global_role
        WHERE
            global_role.user_id = $1
            AND global_role.role_key = 'ADMIN'
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
