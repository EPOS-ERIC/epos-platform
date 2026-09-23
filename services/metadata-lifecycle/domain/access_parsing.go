package domain

import (
	"slices"
	"strings"

	"github.com/google/uuid"
)

// NewGlobalPermission validates and normalizes a global permission key.
func NewGlobalPermission(value string) (GlobalPermission, error) {
	permission := GlobalPermission(strings.TrimSpace(value))
	switch permission {
	case GlobalPermissionMembershipManage,
		GlobalPermissionRoleManage,
		GlobalPermissionPermissionManage,
		GlobalPermissionNotificationManage,
		GlobalPermissionOrganizationManage,
		GlobalPermissionContactPointManage,
		GlobalPermissionCategorySchemeManage,
		GlobalPermissionResourceCreate,
		GlobalPermissionResourceEdit,
		GlobalPermissionResourceSubmit,
		GlobalPermissionResourceWithdraw,
		GlobalPermissionResourceReopen,
		GlobalPermissionResourceDiscard,
		GlobalPermissionResourceReject,
		GlobalPermissionResourcePublish,
		GlobalPermissionResourceUnpublish,
		GlobalPermissionCategoryManage:
		return permission, nil
	default:
		return "", ErrInvalidAccess
	}
}

// NewGlobalPermissions validates and normalizes global permission keys.
func NewGlobalPermissions(values []string) ([]GlobalPermission, error) {
	if values == nil {
		return nil, nil
	}

	permissions := make([]GlobalPermission, len(values))
	for i, value := range values {
		permission, err := NewGlobalPermission(value)
		if err != nil {
			return nil, err
		}
		permissions[i] = permission
	}
	return permissions, nil
}

// NewGroupPermission validates and normalizes a group permission key.
func NewGroupPermission(value string) (GroupPermission, error) {
	permission := GroupPermission(strings.TrimSpace(value))
	switch permission {
	case GroupPermissionResourceCreate,
		GroupPermissionResourceEdit,
		GroupPermissionResourceSubmit,
		GroupPermissionResourceWithdraw,
		GroupPermissionResourceReopen,
		GroupPermissionResourceDiscard,
		GroupPermissionResourceReject,
		GroupPermissionResourcePublish,
		GroupPermissionResourceUnpublish,
		GroupPermissionMembershipManage,
		GroupPermissionRoleManage,
		GroupPermissionPermissionManage,
		GroupPermissionNotificationManage,
		GroupPermissionCategoryManage:
		return permission, nil
	default:
		return "", ErrInvalidAccess
	}
}

// NewGroupPermissions validates and normalizes group permission keys.
func NewGroupPermissions(values []string) ([]GroupPermission, error) {
	if values == nil {
		return nil, nil
	}

	permissions := make([]GroupPermission, len(values))
	for i, value := range values {
		permission, err := NewGroupPermission(value)
		if err != nil {
			return nil, err
		}
		permissions[i] = permission
	}
	return permissions, nil
}

// NewGroupRole validates and normalizes a role path value.
func NewGroupRole(value string) (GroupRole, error) {
	role := GroupRole(strings.TrimSpace(value))
	if role != GroupRoleEditor && role != GroupRoleReviewer && role != GroupRoleAdmin {
		return "", ErrInvalidAccess
	}
	return role, nil
}

// NewRequestedGroupRole validates and normalizes a self-requested role.
func NewRequestedGroupRole(value string) (GroupRole, error) {
	role, err := NewGroupRole(value)
	if err != nil || role == GroupRoleAdmin {
		return "", ErrInvalidAccess
	}
	return role, nil
}

// NewDecision validates and normalizes a request decision.
func NewDecision(value string) (Decision, error) {
	decision := Decision(strings.TrimSpace(value))
	switch decision {
	case DecisionAccept, DecisionReject:
		return decision, nil
	default:
		return "", ErrInvalidAccess
	}
}

// NewPersistedMembershipStatus validates a status loaded from storage.
func NewPersistedMembershipStatus(value string) (MembershipStatus, error) {
	status := MembershipStatus(value)
	switch status {
	case MembershipPending, MembershipActive, MembershipRejected:
		return status, nil
	default:
		return "", ErrInvalidAccess
	}
}

// NewPersistedRoleRequestStatus validates a status loaded from storage.
func NewPersistedRoleRequestStatus(value string) (RoleRequestStatus, error) {
	status := RoleRequestStatus(value)
	switch status {
	case RoleRequestPending, RoleRequestAccepted, RoleRequestRejected:
		return status, nil
	default:
		return "", ErrInvalidAccess
	}
}

// NewMembershipUser creates a valid membership user projection.
func NewMembershipUser(value MembershipUser) (MembershipUser, error) {
	value.ID = strings.TrimSpace(value.ID)
	if value.ID == "" {
		return MembershipUser{}, ErrInvalidAccess
	}
	return value, nil
}

// NewRoleRequest creates a valid role request projection.
func NewRoleRequest(value RoleRequest) (RoleRequest, error) {
	if value.ID == uuid.Nil {
		return RoleRequest{}, ErrInvalidAccess
	}
	if value.Role != GroupRoleEditor && value.Role != GroupRoleReviewer && value.Role != GroupRoleAdmin {
		return RoleRequest{}, ErrInvalidAccess
	}
	if _, err := NewPersistedRoleRequestStatus(string(value.Status)); err != nil {
		return RoleRequest{}, err
	}
	return value, nil
}

// NewGroupMembership creates a valid group membership projection.
func NewGroupMembership(value GroupMembership) (GroupMembership, error) {
	user, err := NewMembershipUser(value.User)
	if err != nil {
		return GroupMembership{}, err
	}
	status, err := NewPersistedMembershipStatus(string(value.Status))
	if err != nil {
		return GroupMembership{}, err
	}

	roles := slices.Clone(value.Roles)
	for _, role := range roles {
		if role != GroupRoleEditor && role != GroupRoleReviewer && role != GroupRoleAdmin {
			return GroupMembership{}, ErrInvalidAccess
		}
	}
	grantedPermissions, err := normalizeGroupPermissions(value.GrantedPermissions)
	if err != nil {
		return GroupMembership{}, err
	}
	rolePermissions, err := normalizeGroupPermissions(value.RolePermissions)
	if err != nil {
		return GroupMembership{}, err
	}

	roleRequests := make([]RoleRequest, 0, len(value.RoleRequests))
	for _, request := range value.RoleRequests {
		request, err := NewRoleRequest(request)
		if err != nil {
			return GroupMembership{}, err
		}
		roleRequests = append(roleRequests, request)
	}

	value.User = user
	value.Status = status
	value.Roles = roles
	value.GrantedPermissions = grantedPermissions
	value.RolePermissions = rolePermissions
	value.RoleRequests = roleRequests
	return value, nil
}

func normalizeGroupPermissions(values []GroupPermission) ([]GroupPermission, error) {
	if values == nil {
		return nil, nil
	}

	permissions := make([]GroupPermission, len(values))
	for i, value := range values {
		permission, err := NewGroupPermission(string(value))
		if err != nil {
			return nil, err
		}
		permissions[i] = permission
	}
	return permissions, nil
}
