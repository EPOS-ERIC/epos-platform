package domain

import "time"

type MembershipUser struct {
	ID         string
	GivenName  string
	FamilyName string
	Email      string
}

type RoleRequest struct {
	ID          string
	Role        GroupRole
	Status      RoleRequestStatus
	RequestedAt time.Time
}

type GroupMembership struct {
	User         MembershipUser
	Status       string
	Roles        []string
	Permissions  []string
	RoleRequests []RoleRequest
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
