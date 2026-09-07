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

type UserSummary struct {
	ID                string
	Email             string
	GivenName         string
	FamilyName        string
	Status            UserStatus
	GlobalRoles       []GlobalRole
	GlobalPermissions []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type User struct {
	UserSummary
	Memberships []Membership
}

// GroupSummary identifies the group attached to a user membership.
type GroupSummary struct {
	ID   uuid.UUID
	Name string
}

// Membership is a membership projection from a user's perspective.
type Membership struct {
	Group        GroupSummary
	Roles        []GroupRole
	Status       MembershipStatus
	Permissions  []string
	RoleRequests []RoleRequest
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
