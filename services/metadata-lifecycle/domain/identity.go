package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// UserClaims contains profile fields supplied by an external identity provider.
type UserClaims struct {
	ID         string
	Email      string
	GivenName  string
	FamilyName string
}

type UserSummary struct {
	ID                       string
	Email                    string
	GivenName                string
	FamilyName               string
	Status                   UserStatus
	GlobalRoles              []GlobalRole
	GlobalGrantedPermissions []GlobalPermission
	GlobalRolePermissions    []GlobalPermission
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type User struct {
	UserSummary
	BlockReason *string
	Memberships []Membership
}

// GroupSummary identifies the group attached to a user membership.
type GroupSummary struct {
	ID   uuid.UUID
	Name string
}

// Membership is a membership projection from a user's perspective.
type Membership struct {
	Group              GroupSummary
	Roles              []GroupRole
	Status             MembershipStatus
	GrantedPermissions []GroupPermission
	RolePermissions    []GroupPermission
	RoleRequests       []RoleRequest
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Normalize validates and normalizes claims supplied by an identity provider.
func (value *UserClaims) Normalize() error {
	value.ID, value.Email, value.GivenName, value.FamilyName = normalizeUserFields(
		value.ID,
		value.Email,
		value.GivenName,
		value.FamilyName,
	)
	switch {
	case value.ID == "":
		return invalidField("id", "ID must not be blank.", nil)
	case value.Email == "":
		return invalidField("email", "Email must not be blank.", nil)
	case value.GivenName == "":
		return invalidField("given_name", "Given name must not be blank.", nil)
	case value.FamilyName == "":
		return invalidField("family_name", "Family name must not be blank.", nil)
	}

	return nil
}

// Normalize validates and normalizes a persisted group reference.
func (value *GroupSummary) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	value.Name = strings.TrimSpace(value.Name)
	if value.ID == uuid.Nil || value.Name == "" {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	return nil
}

// Normalize validates and normalizes a persisted user membership.
func (value *Membership) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if err := value.Group.Normalize(); err != nil {
		return err
	}

	status, err := ParsePersistedMembershipStatus(string(value.Status))
	if err != nil {
		return err
	}

	roles := slices.Clone(value.Roles)
	for i, role := range roles {
		roles[i], err = ParseGroupRole(string(role))
		if err != nil {
			return err
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

// Normalize validates and normalizes a persisted user projection.
func (value *User) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if err := value.UserSummary.Normalize(); err != nil {
		return err
	}

	memberships := make([]Membership, 0, len(value.Memberships))
	for _, membership := range value.Memberships {
		if err := membership.Normalize(); err != nil {
			return err
		}
		memberships = append(memberships, membership)
	}

	value.Memberships = memberships
	return nil
}

// Normalize validates and normalizes a persisted user summary.
func (value *UserSummary) Normalize() (err error) {
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
		return apperror.Default(apperror.CodeValidationFailed)
	}

	status, err := ParsePersistedUserStatus(string(value.Status))
	if err != nil {
		return err
	}
	globalGrantedPermissions, err := normalizeGlobalPermissions(value.GlobalGrantedPermissions)
	if err != nil {
		return err
	}
	globalRolePermissions, err := normalizeGlobalPermissions(value.GlobalRolePermissions)
	if err != nil {
		return err
	}

	value.Status = status
	value.GlobalRoles = slices.Clone(value.GlobalRoles)
	value.GlobalGrantedPermissions = globalGrantedPermissions
	value.GlobalRolePermissions = globalRolePermissions
	return nil
}

func normalizeGlobalPermissions(values []GlobalPermission) ([]GlobalPermission, error) {
	if values == nil {
		return nil, nil
	}

	permissions := make([]GlobalPermission, len(values))
	for i, value := range values {
		permission, err := ParseGlobalPermission(string(value))
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
