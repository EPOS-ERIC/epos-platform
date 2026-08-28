package domain

import (
	"slices"
	"strings"

	"github.com/google/uuid"
)

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

// NewMembershipStatus validates and normalizes a membership decision.
func NewMembershipStatus(value string) (MembershipStatus, error) {
	status := MembershipStatus(strings.TrimSpace(value))
	switch status {
	case MembershipActive, MembershipRejected:
		return status, nil
	default:
		return "", ErrInvalidAccess
	}
}

// NewRoleRequestStatus validates and normalizes a role-request decision.
func NewRoleRequestStatus(value string) (RoleRequestStatus, error) {
	status := RoleRequestStatus(strings.TrimSpace(value))
	switch status {
	case RoleRequestAccepted, RoleRequestRejected:
		return status, nil
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
	value.Permissions = slices.Clone(value.Permissions)
	value.RoleRequests = roleRequests
	return value, nil
}
