package service

import "strings"

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBlocked  UserStatus = "blocked"
)

func NormalizeUserStatus(v string) UserStatus {
	return UserStatus(strings.ToLower(strings.TrimSpace(v)))
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
	MembershipRoleViewer   MembershipRole = "viewer"
	MembershipRoleEditor   MembershipRole = "editor"
	MembershipRoleReviewer MembershipRole = "reviewer"
	MembershipRoleAdmin    MembershipRole = "admin"
)

func NormalizeMembershipRole(v string) MembershipRole {
	return MembershipRole(strings.ToLower(strings.TrimSpace(v)))
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
	MembershipStatusPending  MembershipStatus = "pending"
	MembershipStatusActive   MembershipStatus = "active"
	MembershipStatusRevoked  MembershipStatus = "revoked"
	MembershipStatusRejected MembershipStatus = "rejected"
)

func NormalizeMembershipStatus(v string) MembershipStatus {
	return MembershipStatus(strings.ToLower(strings.TrimSpace(v)))
}

func (s MembershipStatus) Valid() bool {
	switch s {
	case MembershipStatusPending, MembershipStatusActive, MembershipStatusRevoked, MembershipStatusRejected:
		return true
	default:
		return false
	}
}
