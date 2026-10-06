package domain

import (
	"slices"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
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
		return UserClaims{}, apperror.Default(apperror.CodeValidationFailed)
	}

	return value, nil
}

// NewGroupReference validates and normalizes a persisted group reference.
func NewGroupReference(value GroupSummary) (result GroupSummary, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	value.Name = strings.TrimSpace(value.Name)
	if value.ID == uuid.Nil || value.Name == "" {
		return GroupSummary{}, apperror.Default(apperror.CodeValidationFailed)
	}

	return value, nil
}

// NewUserMembership validates and normalizes a persisted user membership.
func NewUserMembership(value Membership) (result Membership, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
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
	grantedPermissions, err := normalizeGroupPermissions(value.GrantedPermissions)
	if err != nil {
		return Membership{}, err
	}
	rolePermissions, err := normalizeGroupPermissions(value.RolePermissions)
	if err != nil {
		return Membership{}, err
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
	value.GrantedPermissions = grantedPermissions
	value.RolePermissions = rolePermissions
	value.RoleRequests = roleRequests
	return value, nil
}

// NewUser validates and normalizes a persisted user projection.
func NewUser(value User) (result User, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	value.ID, value.Email, value.GivenName, value.FamilyName = normalizeUserFields(
		value.ID,
		value.Email,
		value.GivenName,
		value.FamilyName,
	)
	if value.ID == "" || value.Email == "" || value.GivenName == "" || value.FamilyName == "" {
		return User{}, apperror.Default(apperror.CodeValidationFailed)
	}

	status, err := NewPersistedUserStatus(string(value.Status))
	if err != nil {
		return User{}, err
	}
	globalGrantedPermissions, err := normalizeGlobalPermissions(value.GlobalGrantedPermissions)
	if err != nil {
		return User{}, err
	}
	globalRolePermissions, err := normalizeGlobalPermissions(value.GlobalRolePermissions)
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
	value.GlobalGrantedPermissions = globalGrantedPermissions
	value.GlobalRolePermissions = globalRolePermissions
	value.Memberships = memberships
	return value, nil
}

// NewUserSummary validates and normalizes a persisted user summary.
func NewUserSummary(value UserSummary) (result UserSummary, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	value.ID, value.Email, value.GivenName, value.FamilyName = normalizeUserFields(
		value.ID,
		value.Email,
		value.GivenName,
		value.FamilyName,
	)
	if value.ID == "" || value.Email == "" || value.GivenName == "" || value.FamilyName == "" {
		return UserSummary{}, apperror.Default(apperror.CodeValidationFailed)
	}

	status, err := NewPersistedUserStatus(string(value.Status))
	if err != nil {
		return UserSummary{}, err
	}
	globalGrantedPermissions, err := normalizeGlobalPermissions(value.GlobalGrantedPermissions)
	if err != nil {
		return UserSummary{}, err
	}
	globalRolePermissions, err := normalizeGlobalPermissions(value.GlobalRolePermissions)
	if err != nil {
		return UserSummary{}, err
	}

	value.Status = status
	value.GlobalRoles = slices.Clone(value.GlobalRoles)
	value.GlobalGrantedPermissions = globalGrantedPermissions
	value.GlobalRolePermissions = globalRolePermissions
	return value, nil
}

func normalizeGlobalPermissions(values []GlobalPermission) ([]GlobalPermission, error) {
	if values == nil {
		return nil, nil
	}

	permissions := make([]GlobalPermission, len(values))
	for i, value := range values {
		permission, err := NewGlobalPermission(string(value))
		if err != nil {
			return nil, err
		}
		permissions[i] = permission
	}
	return permissions, nil
}

func normalizeUserFields(id, email, givenName, familyName string) (string, string, string, string) {
	return strings.TrimSpace(id), strings.TrimSpace(email), strings.TrimSpace(givenName), strings.TrimSpace(familyName)
}
