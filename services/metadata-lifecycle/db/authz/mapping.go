package authz

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toDomainUser(row sqlc.GetAuthorizationUserRow) (domain.AuthorizationUser, error) {
	status, err := domain.ParsePersistedUserStatus(string(row.Status))
	if err != nil {
		return domain.AuthorizationUser{}, fmt.Errorf("invalid user status: %w", err)
	}

	return domain.AuthorizationUser{
		ID:          row.ID,
		Status:      status,
		GlobalAdmin: row.GlobalAdmin,
	}, nil
}

func toDomainPermissionScope(scope sqlc.PermissionScope) (domain.PermissionScope, error) {
	result, err := domain.ParsePersistedPermissionScope(string(scope))
	if err != nil {
		return "", fmt.Errorf("invalid permission scope: %w", err)
	}

	return result, nil
}

func toDomainVersionAuthz(resourceID, groupID uuid.UUID, status sqlc.ResourceStatus) (domain.VersionAuthz, error) {
	result := domain.VersionAuthz{
		ResourceID: resourceID,
		GroupID:    groupID,
		Status:     domain.Status(status),
	}
	if err := result.Normalize(); err != nil {
		return domain.VersionAuthz{}, fmt.Errorf("construct version authorization: %w", err)
	}

	return result, nil
}
