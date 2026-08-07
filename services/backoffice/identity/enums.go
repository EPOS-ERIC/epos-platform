// Package identity manages users, groups, and group memberships.
package identity

import "strings"

type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
	UserStatusBlocked  UserStatus = "BLOCKED"
)

func NormalizeUserStatus(v string) UserStatus {
	return UserStatus(strings.ToUpper(strings.TrimSpace(v)))
}

func (s UserStatus) Valid() bool {
	switch s {
	case UserStatusActive, UserStatusInactive, UserStatusBlocked:
		return true
	default:
		return false
	}
}

type MembershipRole string

const (
	MembershipRoleViewer   MembershipRole = "VIEWER"
	MembershipRoleEditor   MembershipRole = "EDITOR"
	MembershipRoleReviewer MembershipRole = "REVIEWER"
	MembershipRoleAdmin    MembershipRole = "ADMIN"
)

func NormalizeMembershipRole(v string) MembershipRole {
	return MembershipRole(strings.ToUpper(strings.TrimSpace(v)))
}

func (r MembershipRole) Valid() bool {
	switch r {
	case MembershipRoleViewer, MembershipRoleEditor, MembershipRoleReviewer, MembershipRoleAdmin:
		return true
	default:
		return false
	}
}

type MembershipStatus string

const (
	MembershipStatusPending  MembershipStatus = "PENDING"
	MembershipStatusActive   MembershipStatus = "ACTIVE"
	MembershipStatusRevoked  MembershipStatus = "REVOKED"
	MembershipStatusRejected MembershipStatus = "REJECTED"
)

func NormalizeMembershipStatus(v string) MembershipStatus {
	return MembershipStatus(strings.ToUpper(strings.TrimSpace(v)))
}

func (s MembershipStatus) Valid() bool {
	switch s {
	case MembershipStatusPending, MembershipStatusActive, MembershipStatusRevoked, MembershipStatusRejected:
		return true
	default:
		return false
	}
}
