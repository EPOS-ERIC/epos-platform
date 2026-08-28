package domain

import "github.com/google/uuid"

// Result describes an authorization decision.
type Result struct {
	Allowed bool
	Reason  string
}

// UserStatus is the current state of an authorization user.
type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
	UserStatusBlocked  UserStatus = "BLOCKED"
)

// PermissionScope identifies the resource level covered by a permission.
type PermissionScope string

const (
	PermissionScopeGlobal PermissionScope = "GLOBAL"
	PermissionScopeGroup  PermissionScope = "GROUP"
)

// AuthorizationUser contains the user fields needed for authorization decisions.
type AuthorizationUser struct {
	ID     string
	Status UserStatus
}

// VersionAuthz contains the resource fields needed for version authorization.
type VersionAuthz struct {
	ResourceID uuid.UUID
	GroupID    uuid.UUID
	Status     Status
}
