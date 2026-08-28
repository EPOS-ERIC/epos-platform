package domain

import (
	"time"

	"github.com/google/uuid"
)

// GroupRole is a role that can be assigned to a group member.
type GroupRole string

const (
	GroupRoleEditor   GroupRole = "EDITOR"
	GroupRoleReviewer GroupRole = "REVIEWER"
	GroupRoleAdmin    GroupRole = "GROUP_ADMIN"
)

// MembershipStatus is the current state of a group membership.
type MembershipStatus string

const (
	MembershipPending  MembershipStatus = "PENDING"
	MembershipActive   MembershipStatus = "ACTIVE"
	MembershipRejected MembershipStatus = "REJECTED"
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

// Membership is a complete group membership returned by the domain.
type Membership struct {
	User         MembershipUser
	Status       MembershipStatus
	Roles        []GroupRole
	Permissions  []string
	RoleRequests []RoleRequest
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
