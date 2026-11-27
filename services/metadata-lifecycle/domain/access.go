package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// GroupRole is a role that can be assigned to a group member.
type GroupRole string

const (
	GroupRoleEditor   GroupRole = "EDITOR"
	GroupRoleReviewer GroupRole = "REVIEWER"
	GroupRoleAdmin    GroupRole = "GROUP_ADMIN"
)

// GlobalRole is a role that can be assigned to a user globally.
type GlobalRole string

const (
	GlobalRoleAdmin GlobalRole = "ADMIN"
)

// MembershipStatus is the current state of a group membership.
type MembershipStatus string

const (
	MembershipPending  MembershipStatus = "PENDING"
	MembershipActive   MembershipStatus = "ACTIVE"
	MembershipRejected MembershipStatus = "REJECTED"
)

// Decision identifies an accept or reject operation on a pending request.
type Decision string

const (
	DecisionAccept Decision = "ACCEPT"
	DecisionReject Decision = "REJECT"
)

// RoleRequestStatus is the current state of a role request.
type RoleRequestStatus string

const (
	RoleRequestPending  RoleRequestStatus = "PENDING"
	RoleRequestAccepted RoleRequestStatus = "ACCEPTED"
	RoleRequestRejected RoleRequestStatus = "REJECTED"
)

// MembershipUser identifies the user attached to a group membership.
type MembershipUser struct {
	ID         string
	GivenName  string
	FamilyName string
	Email      string
}

// RoleRequest describes a requested group role.
type RoleRequest struct {
	ID          uuid.UUID
	Role        GroupRole
	Status      RoleRequestStatus
	RequestedAt time.Time
}

// RoleRequestDecision contains the decided request and resulting membership.
type RoleRequestDecision struct {
	Request    RoleRequest
	Membership GroupMembership
}

// GroupMembership is a complete group membership returned by group access operations.
type GroupMembership struct {
	User               MembershipUser
	Status             MembershipStatus
	Roles              []GroupRole
	GrantedPermissions []GroupPermission
	RolePermissions    []GroupPermission
	RoleRequests       []RoleRequest
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ParseGlobalPermission validates and normalizes a global permission key.
func ParseGlobalPermission(value string) (GlobalPermission, error) {
	permission := GlobalPermission(strings.TrimSpace(value))
	switch permission {
	case GlobalPermissionMembershipManage,
		GlobalPermissionRoleManage,
		GlobalPermissionPermissionManage,
		GlobalPermissionAdminManage,
		GlobalPermissionUserBlock,
		GlobalPermissionUserAPITokenRevoke,
		GlobalPermissionNotificationManage,
		GlobalPermissionOrganizationManage,
		GlobalPermissionContactPointManage,
		GlobalPermissionGroupManage,
		GlobalPermissionConceptManage,
		GlobalPermissionResourceRead,
		GlobalPermissionResourceCreate,
		GlobalPermissionResourceEdit,
		GlobalPermissionResourceSubmit,
		GlobalPermissionResourceWithdraw,
		GlobalPermissionResourceReopen,
		GlobalPermissionResourceDiscard,
		GlobalPermissionResourceReject,
		GlobalPermissionResourcePublish,
		GlobalPermissionResourceUnpublish,
		GlobalPermissionCategoryManage,
		GlobalPermissionConfigurationManage,
		GlobalPermissionConverterManage,
		GlobalPermissionMonitoringRead:
		return permission, nil
	default:
		return "", apperror.Default(apperror.CodeInternal)
	}
}

// ParseGlobalPermissions validates and normalizes global permission keys.
func ParseGlobalPermissions(values []string) ([]GlobalPermission, error) {
	if values == nil {
		return nil, nil
	}

	permissions := make([]GlobalPermission, len(values))
	for i, value := range values {
		permission, err := ParseGlobalPermission(value)
		if err != nil {
			return nil, err
		}
		permissions[i] = permission
	}
	return permissions, nil
}

// ParseGroupPermission validates and normalizes a group permission key.
func ParseGroupPermission(value string) (GroupPermission, error) {
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
		return "", apperror.Default(apperror.CodeInternal)
	}
}

// ParseGroupRole validates and normalizes a role path value.
func ParseGroupRole(value string) (GroupRole, error) {
	role := GroupRole(strings.TrimSpace(value))
	if role != GroupRoleEditor && role != GroupRoleReviewer && role != GroupRoleAdmin {
		return "", apperror.Default(apperror.CodeInternal)
	}
	return role, nil
}

// ParseRequestedGroupRole validates and normalizes a self-requested role.
func ParseRequestedGroupRole(value string) (GroupRole, error) {
	role, err := ParseGroupRole(value)
	if err != nil || role == GroupRoleAdmin {
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
	return role, nil
}

// ParsePersistedMembershipStatus validates a status loaded from storage.
func ParsePersistedMembershipStatus(value string) (MembershipStatus, error) {
	status := MembershipStatus(value)
	switch status {
	case MembershipPending, MembershipActive, MembershipRejected:
		return status, nil
	default:
		return "", apperror.Default(apperror.CodeInternal)
	}
}

// ParsePersistedRoleRequestStatus validates a status loaded from storage.
func ParsePersistedRoleRequestStatus(value string) (RoleRequestStatus, error) {
	status := RoleRequestStatus(value)
	switch status {
	case RoleRequestPending, RoleRequestAccepted, RoleRequestRejected:
		return status, nil
	default:
		return "", apperror.Default(apperror.CodeInternal)
	}
}

// Normalize validates a membership user projection.
func (value *MembershipUser) Normalize() error {
	value.ID = strings.TrimSpace(value.ID)
	if value.ID == "" {
		return apperror.Default(apperror.CodeInternal)
	}
	return nil
}

// Normalize validates a role request projection.
func (value *RoleRequest) Normalize() error {
	if value.ID == uuid.Nil {
		return apperror.Default(apperror.CodeInternal)
	}
	if value.Role != GroupRoleEditor && value.Role != GroupRoleReviewer && value.Role != GroupRoleAdmin {
		return apperror.Default(apperror.CodeInternal)
	}
	if _, err := ParsePersistedRoleRequestStatus(string(value.Status)); err != nil {
		return err
	}
	return nil
}

// Normalize validates a group membership projection.
func (value *GroupMembership) Normalize() error {
	if err := value.User.Normalize(); err != nil {
		return err
	}
	status, err := ParsePersistedMembershipStatus(string(value.Status))
	if err != nil {
		return err
	}

	roles := slices.Clone(value.Roles)
	for _, role := range roles {
		if role != GroupRoleEditor && role != GroupRoleReviewer && role != GroupRoleAdmin {
			return apperror.Default(apperror.CodeInternal)
		}
	}
	grantedPermissions, err := normalizeGroupPermissions(value.GrantedPermissions)
	if err != nil {
		return err
	}
	rolePermissions, err := normalizeGroupPermissions(value.RolePermissions)
	if err != nil {
		return err
	}

	roleRequests := make([]RoleRequest, 0, len(value.RoleRequests))
	for _, request := range value.RoleRequests {
		if err := request.Normalize(); err != nil {
			return err
		}
		roleRequests = append(roleRequests, request)
	}

	value.Status = status
	value.Roles = roles
	value.GrantedPermissions = grantedPermissions
	value.RolePermissions = rolePermissions
	value.RoleRequests = roleRequests
	return nil
}

func normalizeGroupPermissions(values []GroupPermission) ([]GroupPermission, error) {
	if values == nil {
		return nil, nil
	}

	permissions := make([]GroupPermission, len(values))
	for i, value := range values {
		permission, err := ParseGroupPermission(string(value))
		if err != nil {
			return nil, err
		}

		permissions[i] = permission
	}

	return permissions, nil
}
