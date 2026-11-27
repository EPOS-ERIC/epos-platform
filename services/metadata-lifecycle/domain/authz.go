package domain

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

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
	GlobalPermissionMembershipManage    GlobalPermission = "global.membership.manage"
	GlobalPermissionRoleManage          GlobalPermission = "global.role.manage"
	GlobalPermissionPermissionManage    GlobalPermission = "global.permission.manage"
	GlobalPermissionAdminManage         GlobalPermission = "global.admin.manage"
	GlobalPermissionUserBlock           GlobalPermission = "global.user.block"
	GlobalPermissionUserAPITokenRevoke  GlobalPermission = "global.user_api_token.revoke"
	GlobalPermissionNotificationManage  GlobalPermission = "global.notification.manage"
	GlobalPermissionOrganizationManage  GlobalPermission = "global.organization.manage"
	GlobalPermissionContactPointManage  GlobalPermission = "global.contact_point.manage"
	GlobalPermissionGroupManage         GlobalPermission = "global.group.manage"
	GlobalPermissionConceptManage       GlobalPermission = "global.concept.manage"
	GlobalPermissionResourceRead        GlobalPermission = "global.resource.read"
	GlobalPermissionResourceCreate      GlobalPermission = "global.resource.create"
	GlobalPermissionResourceEdit        GlobalPermission = "global.resource.edit"
	GlobalPermissionResourceSubmit      GlobalPermission = "global.resource.submit"
	GlobalPermissionResourceWithdraw    GlobalPermission = "global.resource.withdraw"
	GlobalPermissionResourceReopen      GlobalPermission = "global.resource.reopen"
	GlobalPermissionResourceDiscard     GlobalPermission = "global.resource.discard"
	GlobalPermissionResourceReject      GlobalPermission = "global.resource.reject"
	GlobalPermissionResourcePublish     GlobalPermission = "global.resource.publish"
	GlobalPermissionResourceUnpublish   GlobalPermission = "global.resource.unpublish"
	GlobalPermissionCategoryManage      GlobalPermission = "global.category.manage"
	GlobalPermissionConfigurationManage GlobalPermission = "global.configuration.manage"
	GlobalPermissionConverterManage     GlobalPermission = "global.converter.manage"
	GlobalPermissionMonitoringRead      GlobalPermission = "global.monitoring.read"
)

// GroupPermission is a permission evaluated within a group's scope.
type GroupPermission string

const (
	GroupPermissionResourceCreate     GroupPermission = "group.resource.create"
	GroupPermissionResourceEdit       GroupPermission = "group.resource.edit"
	GroupPermissionResourceSubmit     GroupPermission = "group.resource.submit"
	GroupPermissionResourceWithdraw   GroupPermission = "group.resource.withdraw"
	GroupPermissionResourceReopen     GroupPermission = "group.resource.reopen"
	GroupPermissionResourceDiscard    GroupPermission = "group.resource.discard"
	GroupPermissionResourceReject     GroupPermission = "group.resource.reject"
	GroupPermissionResourcePublish    GroupPermission = "group.resource.publish"
	GroupPermissionResourceUnpublish  GroupPermission = "group.resource.unpublish"
	GroupPermissionMembershipManage   GroupPermission = "group.membership.manage"
	GroupPermissionRoleManage         GroupPermission = "group.role.manage"
	GroupPermissionPermissionManage   GroupPermission = "group.permission.manage"
	GroupPermissionNotificationManage GroupPermission = "group.notification.manage"
	GroupPermissionCategoryManage     GroupPermission = "group.category.manage"
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

// ParsePersistedUserStatus validates a user status loaded from storage.
func ParsePersistedUserStatus(value string) (UserStatus, error) {
	status := UserStatus(value)
	switch status {
	case UserStatusActive, UserStatusBlocked:
		return status, nil
	default:
		return "", apperror.Default(apperror.CodeInternal)
	}
}

// ParsePersistedPermissionScope validates a permission scope loaded from storage.
func ParsePersistedPermissionScope(value string) (PermissionScope, error) {
	scope := PermissionScope(value)
	switch scope {
	case PermissionScopeGlobal, PermissionScopeGroup:
		return scope, nil
	default:
		return "", apperror.Default(apperror.CodeInternal)
	}
}

// Normalize validates a permission loaded from storage.
func (value *Permission) Normalize() error {
	if value.Key == "" || value.Label == "" || value.Description == "" {
		return apperror.Default(apperror.CodeInternal)
	}
	parsedScope, err := ParsePersistedPermissionScope(string(value.Scope))
	if err != nil {
		return err
	}
	value.Scope = parsedScope
	return nil
}

// Normalize validates a persisted version authorization projection.
func (value *VersionAuthz) Normalize() error {
	if value.ResourceID == uuid.Nil || value.GroupID == uuid.Nil || !value.Status.Valid() {
		return apperror.Default(apperror.CodeInternal)
	}
	return nil
}
