package users

import (
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toDomainUserSummary(row sqlc.ListUsersRow, globalRoles, globalGrantedPermissions, globalRolePermissions []string) (domain.UserSummary, error) {
	roles, err := toDomainGlobalRoles(globalRoles)
	if err != nil {
		return domain.UserSummary{}, fmt.Errorf("map global roles: %w", err)
	}
	globalGrantedPermissionValues, err := domain.ParseGlobalPermissions(globalGrantedPermissions)
	if err != nil {
		return domain.UserSummary{}, fmt.Errorf("map global granted permissions: %w", err)
	}
	globalRolePermissionValues, err := domain.ParseGlobalPermissions(globalRolePermissions)
	if err != nil {
		return domain.UserSummary{}, fmt.Errorf("map global role permissions: %w", err)
	}

	user := domain.UserSummary{
		ID:                       row.ID,
		Email:                    row.Email,
		GivenName:                row.GivenName,
		FamilyName:               row.FamilyName,
		Status:                   domain.UserStatus(row.Status),
		GlobalRoles:              roles,
		GlobalGrantedPermissions: globalGrantedPermissionValues,
		GlobalRolePermissions:    globalRolePermissionValues,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
	if err := user.Normalize(); err != nil {
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
	blockReason *string,
	createdAt,
	updatedAt time.Time,
	globalRoles []string,
	globalGrantedPermissions []string,
	globalRolePermissions []string,
	memberships []domain.Membership,
) (domain.User, error) {
	roles, err := toDomainGlobalRoles(globalRoles)
	if err != nil {
		return domain.User{}, fmt.Errorf("map global roles: %w", err)
	}
	globalGrantedPermissionValues, err := domain.ParseGlobalPermissions(globalGrantedPermissions)
	if err != nil {
		return domain.User{}, fmt.Errorf("map global granted permissions: %w", err)
	}
	globalRolePermissionValues, err := domain.ParseGlobalPermissions(globalRolePermissions)
	if err != nil {
		return domain.User{}, fmt.Errorf("map global role permissions: %w", err)
	}

	user := domain.User{
		ID:                       id,
		Email:                    email,
		GivenName:                givenName,
		FamilyName:               familyName,
		Status:                   domain.UserStatus(status),
		BlockReason:              blockReason,
		GlobalRoles:              roles,
		GlobalGrantedPermissions: globalGrantedPermissionValues,
		GlobalRolePermissions:    globalRolePermissionValues,
		CreatedAt:                createdAt,
		UpdatedAt:                updatedAt,
		Memberships:              memberships,
	}
	if err := user.Normalize(); err != nil {
		return domain.User{}, fmt.Errorf("construct user: %w", err)
	}

	return user, nil
}

func toDomainGlobalRoles(values []string) ([]domain.GlobalRole, error) {
	if values == nil {
		return nil, nil
	}

	roles := make([]domain.GlobalRole, len(values))
	for i, value := range values {
		role := domain.GlobalRole(value)
		if role != domain.GlobalRoleAdmin {
			return nil, fmt.Errorf("unknown global role: %q", value)
		}
		roles[i] = role
	}

	return roles, nil
}

func toDomainRoleRequest(id uuid.UUID, role string, status sqlc.RoleRequestStatus, requestedAt time.Time) (domain.RoleRequest, error) {
	groupRole, err := domain.ParseGroupRole(role)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("invalid role: %w", err)
	}
	requestStatus, err := domain.ParsePersistedRoleRequestStatus(string(status))
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("invalid status: %w", err)
	}

	request := domain.RoleRequest{
		ID:          id,
		Role:        groupRole,
		Status:      requestStatus,
		RequestedAt: requestedAt,
	}
	if err := request.Normalize(); err != nil {
		return domain.RoleRequest{}, fmt.Errorf("construct role request: %w", err)
	}

	return request, nil
}
