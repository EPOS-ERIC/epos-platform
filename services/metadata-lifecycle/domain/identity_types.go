package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserClaims contains profile fields supplied by an external identity provider.
type UserClaims struct {
	ID         string
	Email      string
	GivenName  string
	FamilyName string
}

// User contains a complete user profile and access projection.
type User struct {
	ID                string
	Email             string
	GivenName         string
	FamilyName        string
	Status            UserStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	GlobalRoles       []string
	GlobalPermissions []string
	Memberships       []UserMembership
}

// UserSummary contains the user fields needed for directory listings.
type UserSummary struct {
	ID                string
	Email             string
	GivenName         string
	FamilyName        string
	Status            UserStatus
	GlobalRoles       []string
	GlobalPermissions []string
}

// GroupReference identifies the group attached to a user membership.
type GroupReference struct {
	ID   uuid.UUID
	Name string
}

// UserMembership is a membership projection from a user's perspective.
type UserMembership struct {
	Group        GroupReference
	Roles        []GroupRole
	Status       MembershipStatus
	Permissions  []string
	RoleRequests []RoleRequest
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
