-- name: GetAuthorizationUser :one
SELECT
    id,
    STATUS,
    EXISTS (
        SELECT
            1
        FROM
            user_global_roles
        WHERE
            user_id = users.id
            AND role_key = 'ADMIN'
    ) AS global_admin
FROM
    users
WHERE
    id = sqlc.arg(id);

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
    id = sqlc.arg(id);

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
    user_id = sqlc.arg(user_id)
ORDER BY
    role_key;

-- name: GetUserGlobalGrantedPermissions :many
SELECT
    global_grant.permission_key
FROM
    global_permission_grants AS global_grant
WHERE
    global_grant.user_id = sqlc.arg(user_id)
ORDER BY
    permission_key;

-- name: GetUserGlobalRolePermissions :many
SELECT
    role_permission.permission_key
FROM
    user_global_roles AS global_role
    JOIN role_permissions AS role_permission ON role_permission.role_key = global_role.role_key
WHERE
    global_role.user_id = sqlc.arg(user_id)
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
            global_role.user_id = sqlc.arg(user_id)
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
    membership.user_id = sqlc.arg(user_id)
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
    user_id = sqlc.arg(user_id)
ORDER BY
    group_id,
    role_key;

-- name: GetUserMembershipGrantedPermissions :many
SELECT
    direct_grant.group_id AS group_id,
    direct_grant.permission_key AS permission_key
FROM
    group_permission_grants AS direct_grant
    JOIN group_memberships AS membership ON membership.user_id = direct_grant.user_id
    AND membership.group_id = direct_grant.group_id
WHERE
    direct_grant.user_id = sqlc.arg(user_id)
    AND membership.status = 'ACTIVE'
ORDER BY
    1,
    2;

-- name: GetUserMembershipRolePermissions :many
SELECT
    DISTINCT membership.group_id AS group_id,
    role_permission.permission_key AS permission_key
FROM
    group_memberships AS membership
    JOIN group_membership_roles AS role_assignment ON role_assignment.user_id = membership.user_id
    AND role_assignment.group_id = membership.group_id
    JOIN role_permissions AS role_permission ON role_permission.role_key = role_assignment.role_key
WHERE
    membership.user_id = sqlc.arg(user_id)
    AND membership.status = 'ACTIVE'
ORDER BY
    1,
    2;

-- name: ListPendingUserGroupRoleRequests :many
SELECT
    id,
    group_id,
    role_key,
    STATUS,
    requested_at
FROM
    group_role_requests
WHERE
    user_id = sqlc.arg(user_id)
    AND STATUS = 'PENDING'
ORDER BY
    group_id,
    role_key,
    id;

-- name: CreateUser :one
INSERT INTO
    users (
        id,
        email,
        given_name,
        family_name
    )
VALUES
    (
        sqlc.arg(id),
        sqlc.arg(email),
        sqlc.arg(given_name),
        sqlc.arg(family_name)
    )
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
    (
        sqlc.arg(user_id),
        sqlc.arg(permission_key),
        sqlc.arg(granted_by)
    ) ON CONFLICT (user_id, permission_key) DO NOTHING;

-- name: AssignGlobalAdmin :exec
INSERT INTO
    user_global_roles (user_id, role_key, assigned_by)
VALUES
    (
        sqlc.arg(user_id),
        'ADMIN',
        sqlc.arg(assigned_by)
    ) ON CONFLICT (user_id, role_key) DO NOTHING;

-- name: RemoveGlobalAdmin :execrows
DELETE FROM
    user_global_roles
WHERE
    user_id = sqlc.arg(user_id)
    AND role_key = 'ADMIN';

-- name: CountActiveGlobalAdmins :one
SELECT
    COUNT(*)::int
FROM
    user_global_roles AS global_role
    JOIN users ON users.id = global_role.user_id
WHERE
    global_role.role_key = 'ADMIN'
    AND users.status = 'ACTIVE';

-- name: BlockUser :execrows
UPDATE
    users
SET
    STATUS = 'BLOCKED',
    block_reason = sqlc.arg(reason),
    blocked_by = sqlc.arg(blocked_by),
    blocked_at = NOW()
WHERE
    id = sqlc.arg(user_id)
    AND STATUS <> 'BLOCKED';

-- name: UnblockUser :execrows
UPDATE
    users
SET
    STATUS = 'ACTIVE',
    block_reason = NULL,
    blocked_by = NULL,
    blocked_at = NULL
WHERE
    id = sqlc.arg(user_id)
    AND STATUS = 'BLOCKED';

-- name: LockGlobalAdminChanges :exec
SELECT
    pg_advisory_xact_lock(hashtextextended('global-admin-role', 0));

-- name: RevokeGlobalPermission :exec
DELETE FROM
    global_permission_grants
WHERE
    user_id = sqlc.arg(user_id)
    AND permission_key = sqlc.arg(permission_key);

-- name: GetPermissionScope :one
SELECT
    scope
FROM
    permissions
WHERE
    KEY = sqlc.arg(permission_key);

-- name: IsGlobalAdmin :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            user_global_roles
        WHERE
            user_id = sqlc.arg(user_id)
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
            global_grant.user_id = sqlc.arg(user_id)
            AND global_grant.permission_key = sqlc.arg(permission_key)
        UNION
        ALL
        SELECT
            1
        FROM
            user_global_roles AS global_role
        WHERE
            global_role.user_id = sqlc.arg(user_id)
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
            user_id = sqlc.arg(user_id)
            AND group_id = sqlc.arg(group_id)
            AND STATUS = 'ACTIVE'
    ) AS active;

-- name: ListActiveMemberGroupIDs :many
SELECT
    group_id
FROM
    group_memberships
WHERE
    user_id = $1
    AND STATUS = 'ACTIVE';

-- name: HasAnyActiveMembership :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            group_memberships
        WHERE
            user_id = sqlc.arg(user_id)
            AND STATUS = 'ACTIVE'
    ) AS active;

-- name: HasGroupRole :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            group_membership_roles
        WHERE
            user_id = sqlc.arg(user_id)
            AND group_id = sqlc.arg(group_id)
            AND role_key = sqlc.arg(role_key)
    ) AS has_role;

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
            group_permission_grants.user_id = sqlc.arg(user_id)
            AND group_permission_grants.group_id = sqlc.arg(group_id)
            AND group_permission_grants.permission_key = sqlc.arg(permission_key)
        UNION
        ALL
        SELECT
            1
        FROM
            group_membership_roles AS assignment
            JOIN role_permissions ON role_permissions.role_key = assignment.role_key
        WHERE
            assignment.user_id = sqlc.arg(user_id)
            AND assignment.group_id = sqlc.arg(group_id)
            AND role_permissions.permission_key = sqlc.arg(permission_key)
    ) AS allowed;

-- name: GetDatasetGroup :one
SELECT
    group_id
FROM
    datasets
WHERE
    id = sqlc.arg(dataset_id);

-- name: GetSoftwareApplicationGroup :one
SELECT
    group_id
FROM
    software_applications
WHERE
    id = sqlc.arg(software_application_id);

-- name: GetSoftwareSourceCodeGroup :one
SELECT
    group_id
FROM
    software_source_codes
WHERE
    id = sqlc.arg(software_source_code_id);

-- name: IsDatasetPublished :one
SELECT
    EXISTS (
        SELECT
            1
        FROM
            dataset_versions
        WHERE
            dataset_id = sqlc.arg(dataset_id)
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
            software_application_id = sqlc.arg(software_application_id)
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
            software_source_code_id = sqlc.arg(software_source_code_id)
            AND STATUS = 'PUBLISHED'
    ) AS published;
