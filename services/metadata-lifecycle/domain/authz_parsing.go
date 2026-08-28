package domain

import "github.com/google/uuid"

// NewPersistedUserStatus validates a user status loaded from storage.
func NewPersistedUserStatus(value string) (UserStatus, error) {
	status := UserStatus(value)
	switch status {
	case UserStatusActive, UserStatusInactive, UserStatusBlocked:
		return status, nil
	default:
		return "", ErrInvalidAuthorization
	}
}

// NewPersistedPermissionScope validates a permission scope loaded from storage.
func NewPersistedPermissionScope(value string) (PermissionScope, error) {
	scope := PermissionScope(value)
	switch scope {
	case PermissionScopeGlobal, PermissionScopeGroup:
		return scope, nil
	default:
		return "", ErrInvalidAuthorization
	}
}

// NewVersionAuthz creates a valid persisted version authorization projection.
func NewVersionAuthz(resourceID, groupID uuid.UUID, status Status) (VersionAuthz, error) {
	if resourceID == uuid.Nil || groupID == uuid.Nil || !validStatus(status) {
		return VersionAuthz{}, ErrInvalidAuthorization
	}

	return VersionAuthz{
		ResourceID: resourceID,
		GroupID:    groupID,
		Status:     status,
	}, nil
}
