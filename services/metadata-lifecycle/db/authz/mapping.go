package authz

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toDomainUser(row sqlc.GetAuthorizationUserRow) (domain.User, error) {
	status, err := domain.NewPersistedUserStatus(string(row.Status))
	if err != nil {
		return domain.User{}, fmt.Errorf("invalid user status: %w", err)
	}

	return domain.User{ID: row.ID, Status: status}, nil
}

func toDomainPermissionScope(scope sqlc.PermissionScope) (domain.PermissionScope, error) {
	result, err := domain.NewPersistedPermissionScope(string(scope))
	if err != nil {
		return "", fmt.Errorf("invalid permission scope: %w", err)
	}

	return result, nil
}

func toDomainVersionAuthz(resourceID, groupID uuid.UUID, status sqlc.ResourceStatus) (domain.VersionAuthz, error) {
	result, err := domain.NewVersionAuthz(resourceID, groupID, domain.Status(status))
	if err != nil {
		return domain.VersionAuthz{}, fmt.Errorf("construct version authorization: %w", err)
	}

	return result, nil
}
