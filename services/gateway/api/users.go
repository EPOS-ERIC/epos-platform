package api

import (
	"fmt"
	"slices"
	"strings"
)

type User struct {
	ID                string                  `json:"id"`
	Email             string                  `json:"email"`
	GivenName         string                  `json:"given_name"`
	FamilyName        string                  `json:"family_name"`
	Status            UserStatus              `json:"status"`
	GlobalRoles       []string                `json:"global_roles"`
	GlobalPermissions []string                `json:"global_permissions"`
	Memberships       []UserMembershipSummary `json:"memberships"`
	CreatedAt         string                  `json:"created_at"`
	UpdatedAt         string                  `json:"updated_at"`
}

// UserSummary is gateway response for user list and profile fields.
type UserSummary struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	GivenName         string     `json:"given_name"`
	FamilyName        string     `json:"family_name"`
	Status            UserStatus `json:"status"`
	GlobalRoles       []string   `json:"global_roles"`
	GlobalPermissions []string   `json:"global_permissions"`
	CreatedAt         string     `json:"created_at,omitempty"`
	UpdatedAt         string     `json:"updated_at,omitempty"`
}

type MembershipRole string

const (
	MembershipEditorRole     MembershipRole = "EDITOR"
	MembershipReviewerRole   MembershipRole = "REVIEWER"
	MembershipGroupAdminRole MembershipRole = "GROUP_ADMIN"
)

type MembershipStatus string

const (
	MembershipPendingStatus  MembershipStatus = "PENDING"
	MembershipActiveStatus   MembershipStatus = "ACTIVE"
	MembershipRejectedStatus MembershipStatus = "REJECTED"
)

func (s MembershipStatus) IsDecision() bool {
	return s == MembershipActiveStatus || s == MembershipRejectedStatus
}

type UserMembershipSummary struct {
	Group        ResourceRef      `json:"group"`
	Roles        []MembershipRole `json:"roles"`
	Status       MembershipStatus `json:"status"`
	Permissions  []string         `json:"permissions"`
	RoleRequests []RoleRequest    `json:"role_requests"`
	CreatedAt    string           `json:"created_at"`
	UpdatedAt    string           `json:"updated_at"`
}

type UserStatus string

const (
	ActiveStatus   UserStatus = "ACTIVE"
	InactiveStatus UserStatus = "INACTIVE"
	BlockedStatus  UserStatus = "BLOCKED"
)

func (s UserStatus) IsValid() bool {
	switch s {
	case ActiveStatus, InactiveStatus, BlockedStatus:
		return true
	default:
		return false
	}
}

func NewUserStatusFromValue(value string) (UserStatus, error) {
	status := UserStatus(strings.ToUpper(strings.TrimSpace(value)))
	if status.IsValid() {
		return status, nil
	}

	return "", fmt.Errorf("invalid value '%s' for UserStatus", value)
}

func (u User) HasGlobalRole(role string) bool {
	return slices.Contains(u.GlobalRoles, role)
}

func (u User) IsGlobalAdmin() bool {
	return u.HasGlobalRole("ADMIN")
}

func (u User) HasGlobalPermission(permission string) bool {
	return slices.Contains(u.GlobalPermissions, permission)
}
