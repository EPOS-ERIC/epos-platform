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
func NewGroupReference(value GroupSummary) (GroupSummary, error) {
	value.Name = strings.TrimSpace(value.Name)
	if value.ID == uuid.Nil || value.Name == "" {
		return GroupSummary{}, ErrInvalidUser
	}

	return value, nil
}

// NewUserMembership validates and normalizes a persisted user membership.
func NewUserMembership(value Membership) (Membership, error) {
	group, err := NewGroupReference(value.Group)
	if err != nil {
		return Membership{}, err
	}

	status, err := NewPersistedMembershipStatus(string(value.Status))
	if err != nil {
		return Membership{}, err
	}

	roles := slices.Clone(value.Roles)
	for i, role := range roles {
		roles[i], err = NewGroupRole(string(role))
		if err != nil {
			return Membership{}, err
		}
	}

	roleRequests := make([]RoleRequest, 0, len(value.RoleRequests))
	for _, request := range value.RoleRequests {
		request, err := NewRoleRequest(request)
		if err != nil {
			return Membership{}, err
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

	memberships := make([]Membership, 0, len(value.Memberships))
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
