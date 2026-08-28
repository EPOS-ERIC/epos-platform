package users

import (
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toDomainUserSummary(row sqlc.ListUsersRow, globalRoles, globalPermissions []string) (domain.UserSummary, error) {
	user, err := domain.NewUserSummary(domain.UserSummary{
		ID:                row.ID,
		Email:             row.Email,
		GivenName:         row.GivenName,
		FamilyName:        row.FamilyName,
		Status:            domain.UserStatus(row.Status),
		GlobalRoles:       globalRoles,
		GlobalPermissions: globalPermissions,
	})
	if err != nil {
		return domain.UserSummary{}, fmt.Errorf("construct user summary: %w", err)
	}

	return user, nil
}

func toDomainUser(
	id,
	email,
	givenName,
	familyName string,
	status sqlc.UserStatus,
	createdAt,
	updatedAt time.Time,
	globalRoles []string,
	globalPermissions []string,
	memberships []domain.UserMembership,
) (domain.User, error) {
	user, err := domain.NewUser(domain.User{
		ID:                id,
		Email:             email,
		GivenName:         givenName,
		FamilyName:        familyName,
		Status:            domain.UserStatus(status),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		GlobalRoles:       globalRoles,
		GlobalPermissions: globalPermissions,
		Memberships:       memberships,
	})
	if err != nil {
		return domain.User{}, fmt.Errorf("construct user: %w", err)
	}

	return user, nil
}

func toDomainRoleRequest(id uuid.UUID, role string, status sqlc.RoleRequestStatus, requestedAt time.Time) (domain.RoleRequest, error) {
	groupRole, err := domain.NewGroupRole(role)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("invalid role: %w", err)
	}
	requestStatus, err := domain.NewPersistedRoleRequestStatus(string(status))
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("invalid status: %w", err)
	}

	request, err := domain.NewRoleRequest(domain.RoleRequest{
		ID:          id,
		Role:        groupRole,
		Status:      requestStatus,
		RequestedAt: requestedAt,
	})
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("construct role request: %w", err)
	}

	return request, nil
}
