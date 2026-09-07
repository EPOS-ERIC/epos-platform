package domain

import (
	"time"
)

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

type Membership struct {
	Group            GroupSummary
	Roles            []GroupRole
	Status           MembershipStatus
	GroupPermissions []string
	RoleRequests     []RoleRequest
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type GroupSummary struct {
	ID   string
	Name string
}

type RoleRequest struct {
	ID          string
	Role        GroupRole
	Status      RoleRequestStatus
	RequestedAt time.Time
}

type UserStatus string

const (
	UserStatusActive  UserStatus = "ACTIVE"
	UserStatusBlocked UserStatus = "BLOCKED"
)

type MembershipStatus string

const (
	MembershipStatusUnspecified MembershipStatus = "UNSPECIFIED"
	MembershipStatusPending     MembershipStatus = "PENDING"
	MembershipStatusActive      MembershipStatus = "ACTIVE"
	MembershipStatusRejected    MembershipStatus = "REJECTED"
)

type GroupRole string

const (
	GroupRoleUnspecified GroupRole = "UNSPECIFIED"
	GroupRoleEditor      GroupRole = "EDITOR"
	GroupRoleReviewer    GroupRole = "REVIEWER"
	GroupRoleAdmin       GroupRole = "ADMIN"
)

type GlobalRole string

const (
	GlobalRoleAdmin GlobalRole = "ADMIN"
)

type RoleRequestStatus string

const (
	RoleRequestStatusUnspecified RoleRequestStatus = "UNSPECIFIED"
	RoleRequestStatusPending     RoleRequestStatus = "PENDING"
	RoleRequestStatusAccepted    RoleRequestStatus = "ACCEPTED"
	RoleRequestStatusRejected    RoleRequestStatus = "REJECTED"
)
