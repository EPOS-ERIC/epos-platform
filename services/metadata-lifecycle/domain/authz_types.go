package domain

import "github.com/google/uuid"

// Result describes an authorization decision.
type Result struct {
	Allowed bool
	Reason  string
}

// UserStatus is the current state of an authorization user.
type UserStatus string

const (
	UserStatusActive  UserStatus = "ACTIVE"
	UserStatusBlocked UserStatus = "BLOCKED"
)

// PermissionScope identifies the resource level covered by a permission.
type PermissionScope string

const (
	PermissionScopeGlobal PermissionScope = "GLOBAL"
	PermissionScopeGroup  PermissionScope = "GROUP"
)

// Permission describes one fixed authorization catalog entry.
type Permission struct {
	Key         string
	Label       string
	Description string
	Scope       PermissionScope
}

// GlobalPermission is a permission that applies across all groups.
type GlobalPermission string

const (
	GlobalPermissionMembershipManage     GlobalPermission = "global.membership.manage"
	GlobalPermissionRoleManage           GlobalPermission = "global.role.manage"
	GlobalPermissionPermissionManage     GlobalPermission = "global.permission.manage"
	GlobalPermissionOrganizationManage   GlobalPermission = "organization.manage"
	GlobalPermissionContactPointManage   GlobalPermission = "contact_point.manage"
	GlobalPermissionCategorySchemeManage GlobalPermission = "category_scheme.manage"
)

// GroupPermission is a permission evaluated within a group's scope.
type GroupPermission string

const (
	GroupPermissionResourceCreate    GroupPermission = "resource.create"
	GroupPermissionResourceEdit      GroupPermission = "resource.edit"
	GroupPermissionResourceSubmit    GroupPermission = "resource.submit"
	GroupPermissionResourceWithdraw  GroupPermission = "resource.withdraw"
	GroupPermissionResourceReopen    GroupPermission = "resource.reopen"
	GroupPermissionResourceDiscard   GroupPermission = "resource.discard"
	GroupPermissionResourceReject    GroupPermission = "resource.reject"
	GroupPermissionResourcePublish   GroupPermission = "resource.publish"
	GroupPermissionResourceUnpublish GroupPermission = "resource.unpublish"
	GroupPermissionMembershipManage  GroupPermission = "group.membership.manage"
	GroupPermissionRoleManage        GroupPermission = "group.role.manage"
	GroupPermissionPermissionManage  GroupPermission = "group.permission.manage"
	GroupPermissionCategoryManage    GroupPermission = "category.manage"
)

// AuthorizationUser contains the user fields needed for authorization decisions.
type AuthorizationUser struct {
	ID          string
	Status      UserStatus
	GlobalAdmin bool
}

// VersionAuthz contains the resource fields needed for version authorization.
type VersionAuthz struct {
	ResourceID uuid.UUID
	GroupID    uuid.UUID
	Status     Status
}

// DiscoveryAuthorization contains candidates allowed by the discovery policy.
type DiscoveryAuthorization struct {
	AllowedCandidates []DiscoveryCandidate
	CreatedByGroupIDs map[uuid.UUID]struct{}
}
