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

// GlobalRole is a role that can be assigned to a user globally.
type GlobalRole string

const (
	GlobalRoleAdmin GlobalRole = "ADMIN"
)

// MembershipStatus is the current state of a group membership.
type MembershipStatus string

const (
	MembershipPending  MembershipStatus = "PENDING"
	MembershipActive   MembershipStatus = "ACTIVE"
	MembershipRejected MembershipStatus = "REJECTED"
)

// Decision identifies an accept or reject operation on a pending request.
type Decision string

const (
	DecisionAccept Decision = "ACCEPT"
	DecisionReject Decision = "REJECT"
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

// RoleRequestDecision contains the decided request and resulting membership.
type RoleRequestDecision struct {
	Request    RoleRequest
	Membership GroupMembership
}

// GroupMembership is a complete group membership returned by group access operations.
type GroupMembership struct {
	User               MembershipUser
	Status             MembershipStatus
	Roles              []GroupRole
	GrantedPermissions []GroupPermission
	RolePermissions    []GroupPermission
	RoleRequests       []RoleRequest
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
