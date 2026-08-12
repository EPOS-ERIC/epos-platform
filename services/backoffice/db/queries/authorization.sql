-- name: GetAuthorizationUser :one
SELECT id, status FROM users WHERE id = $1;

-- name: GetPermissionScope :one
SELECT scope FROM permissions WHERE key = $1;

-- name: IsGlobalAdmin :one
SELECT EXISTS (
    SELECT 1 FROM user_global_roles WHERE user_id = $1 AND role_key = 'ADMIN'
) AS allowed;

-- name: HasGlobalPermission :one
SELECT EXISTS (
    SELECT 1 FROM global_permission_grants WHERE user_id = $1 AND permission_key = $2
) AS allowed;

-- name: IsActiveMember :one
SELECT EXISTS (
    SELECT 1 FROM group_memberships WHERE user_id = $1 AND group_id = $2 AND status = 'ACTIVE'
) AS active;

-- name: HasAnyActiveMembership :one
SELECT EXISTS (
    SELECT 1 FROM group_memberships WHERE user_id = $1 AND status = 'ACTIVE'
) AS active;

-- name: HasGroupPermission :one
SELECT EXISTS (
    SELECT 1
    FROM group_permission_grants
    WHERE group_permission_grants.user_id = $1
      AND group_permission_grants.group_id = $2
      AND group_permission_grants.permission_key = $3
    UNION ALL
    SELECT 1
    FROM group_membership_roles AS assignment
    JOIN role_permissions ON role_permissions.role_key = assignment.role_key
    WHERE assignment.user_id = $1
      AND assignment.group_id = $2
      AND role_permissions.permission_key = $3
) AS allowed;

-- name: GetDatasetGroup :one
SELECT group_id FROM datasets WHERE id = $1;

-- name: GetSoftwareApplicationGroup :one
SELECT group_id FROM software_applications WHERE id = $1;

-- name: GetSoftwareSourceCodeGroup :one
SELECT group_id FROM software_source_codes WHERE id = $1;

-- name: IsDatasetPublished :one
SELECT EXISTS (
    SELECT 1 FROM dataset_versions WHERE dataset_id = $1 AND status = 'PUBLISHED'
) AS published;

-- name: IsSoftwareApplicationPublished :one
SELECT EXISTS (
    SELECT 1 FROM software_application_versions WHERE software_application_id = $1 AND status = 'PUBLISHED'
) AS published;

-- name: IsSoftwareSourceCodePublished :one
SELECT EXISTS (
    SELECT 1 FROM software_source_code_versions WHERE software_source_code_id = $1 AND status = 'PUBLISHED'
) AS published;
