package access

import (
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toDomainRoleRequest(id uuid.UUID, role, status string, requestedAt time.Time) (domain.RoleRequest, error) {
	groupRole, err := domain.NewGroupRole(role)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("invalid role: %w", err)
	}
	roleRequestStatus, err := domain.NewPersistedRoleRequestStatus(status)
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("invalid status: %w", err)
	}

	result, err := domain.NewRoleRequest(domain.RoleRequest{
		ID:          id,
		Role:        groupRole,
		Status:      roleRequestStatus,
		RequestedAt: requestedAt,
	})
	if err != nil {
		return domain.RoleRequest{}, fmt.Errorf("construct role request: %w", err)
	}

	return result, nil
}

func toDomainMembership(
	user domain.MembershipUser,
	membership sqlc.GroupMembership,
	roles []domain.GroupRole,
	permissions []string,
	roleRequests []domain.RoleRequest,
) (domain.Membership, error) {
	result, err := domain.NewMembership(domain.Membership{
		User:         user,
		Status:       domain.MembershipStatus(membership.Status),
		Roles:        roles,
		Permissions:  permissions,
		RoleRequests: roleRequests,
		CreatedAt:    membership.CreatedAt,
		UpdatedAt:    membership.UpdatedAt,
	})
	if err != nil {
		return domain.Membership{}, fmt.Errorf("construct membership: %w", err)
	}

	return result, nil
}
