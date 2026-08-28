package domain

import (
	"slices"
	"strings"

	"github.com/google/uuid"
)

// NewUserClaims validates and normalizes claims supplied by an identity provider.
func NewUserClaims(value UserClaims) (UserClaims, error) {
	value.ID, value.Email, value.GivenName, value.FamilyName = normalizeUserFields(
		value.ID,
		value.Email,
		value.GivenName,
		value.FamilyName,
	)
	if value.ID == "" || value.Email == "" || value.GivenName == "" || value.FamilyName == "" {
		return UserClaims{}, ErrInvalidUser
	}

	return value, nil
}

// NewGroupReference validates and normalizes a persisted group reference.
func NewGroupReference(value GroupReference) (GroupReference, error) {
	value.Name = strings.TrimSpace(value.Name)
	if value.ID == uuid.Nil || value.Name == "" {
		return GroupReference{}, ErrInvalidUser
	}

	return value, nil
}

// NewUserMembership validates and normalizes a persisted user membership.
func NewUserMembership(value UserMembership) (UserMembership, error) {
	group, err := NewGroupReference(value.Group)
	if err != nil {
		return UserMembership{}, err
	}

	status, err := NewPersistedMembershipStatus(string(value.Status))
	if err != nil {
		return UserMembership{}, err
	}

	roles := slices.Clone(value.Roles)
	for i, role := range roles {
		roles[i], err = NewGroupRole(string(role))
		if err != nil {
			return UserMembership{}, err
		}
	}

	roleRequests := make([]RoleRequest, 0, len(value.RoleRequests))
	for _, request := range value.RoleRequests {
		request, err := NewRoleRequest(request)
		if err != nil {
			return UserMembership{}, err
		}
		roleRequests = append(roleRequests, request)
	}

	value.Group = group
	value.Status = status
	value.Roles = roles
	value.Permissions = slices.Clone(value.Permissions)
	value.RoleRequests = roleRequests
	return value, nil
}

// NewUser validates and normalizes a persisted user projection.
func NewUser(value User) (User, error) {
	value.ID, value.Email, value.GivenName, value.FamilyName = normalizeUserFields(
		value.ID,
		value.Email,
		value.GivenName,
		value.FamilyName,
	)
	if value.ID == "" || value.Email == "" || value.GivenName == "" || value.FamilyName == "" {
		return User{}, ErrInvalidUser
	}

	status, err := NewPersistedUserStatus(string(value.Status))
	if err != nil {
		return User{}, err
	}

	memberships := make([]UserMembership, 0, len(value.Memberships))
	for _, membership := range value.Memberships {
		membership, err := NewUserMembership(membership)
		if err != nil {
			return User{}, err
		}
		memberships = append(memberships, membership)
	}

	value.Status = status
	value.GlobalRoles = slices.Clone(value.GlobalRoles)
	value.GlobalPermissions = slices.Clone(value.GlobalPermissions)
	value.Memberships = memberships
	return value, nil
}

// NewUserSummary validates and normalizes a persisted user summary.
func NewUserSummary(value UserSummary) (UserSummary, error) {
	value.ID, value.Email, value.GivenName, value.FamilyName = normalizeUserFields(
		value.ID,
		value.Email,
		value.GivenName,
		value.FamilyName,
	)
	if value.ID == "" || value.Email == "" || value.GivenName == "" || value.FamilyName == "" {
		return UserSummary{}, ErrInvalidUser
	}

	status, err := NewPersistedUserStatus(string(value.Status))
	if err != nil {
		return UserSummary{}, err
	}

	value.Status = status
	value.GlobalRoles = slices.Clone(value.GlobalRoles)
	value.GlobalPermissions = slices.Clone(value.GlobalPermissions)
	return value, nil
}

func normalizeUserFields(id, email, givenName, familyName string) (string, string, string, string) {
	return strings.TrimSpace(id), strings.TrimSpace(email), strings.TrimSpace(givenName), strings.TrimSpace(familyName)
}
