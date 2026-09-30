package domain

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// NewPersistedUserStatus validates a user status loaded from storage.
func NewPersistedUserStatus(value string) (UserStatus, error) {
	status := UserStatus(value)
	switch status {
	case UserStatusActive, UserStatusBlocked:
		return status, nil
	default:
		return "", apperror.Default(apperror.CodeInternal)
	}
}

// NewPersistedPermissionScope validates a permission scope loaded from storage.
func NewPersistedPermissionScope(value string) (PermissionScope, error) {
	scope := PermissionScope(value)
	switch scope {
	case PermissionScopeGlobal, PermissionScopeGroup:
		return scope, nil
	default:
		return "", apperror.Default(apperror.CodeInternal)
	}
}

// NewPersistedPermission validates a permission loaded from storage.
func NewPersistedPermission(key, label, description, scope string) (Permission, error) {
	if key == "" || label == "" || description == "" {
		return Permission{}, apperror.Default(apperror.CodeInternal)
	}

	parsedScope, err := NewPersistedPermissionScope(scope)
	if err != nil {
		return Permission{}, err
	}

	return Permission{
		Key:         key,
		Label:       label,
		Description: description,
		Scope:       parsedScope,
	}, nil
}

// NewVersionAuthz creates a valid persisted version authorization projection.
func NewVersionAuthz(resourceID, groupID uuid.UUID, status Status) (VersionAuthz, error) {
	if resourceID == uuid.Nil || groupID == uuid.Nil || !validStatus(status) {
		return VersionAuthz{}, apperror.Default(apperror.CodeInternal)
	}

	return VersionAuthz{
		ResourceID: resourceID,
		GroupID:    groupID,
		Status:     status,
	}, nil
}
