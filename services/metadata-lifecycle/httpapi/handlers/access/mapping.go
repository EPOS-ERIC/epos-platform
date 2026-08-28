package access

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

func toRoles(value membershipRequest) ([]domain.GroupRole, error) {
	roles := make([]domain.GroupRole, 0, len(value.Roles))
	seen := make(map[domain.GroupRole]struct{}, len(value.Roles))
	for _, role := range value.Roles {
		parsedRole, err := domain.NewRequestedGroupRole(role)
		if err != nil {
			return nil, fmt.Errorf("parse requested role: %w", err)
		}
		if _, ok := seen[parsedRole]; ok {
			return nil, fmt.Errorf("duplicate requested role: %w", domain.ErrInvalidAccess)
		}
		seen[parsedRole] = struct{}{}
		roles = append(roles, parsedRole)
	}

	return roles, nil
}

func toMembershipResponse(value domain.GroupMembership) membershipResponse {
	roles := make([]string, 0, len(value.Roles))
	for _, role := range value.Roles {
		roles = append(roles, string(role))
	}

	requests := make([]roleRequestResponse, 0, len(value.RoleRequests))
	for _, request := range value.RoleRequests {
		requests = append(requests, toRoleRequestResponse(request))
	}

	return membershipResponse{
		User: membershipUserResponse{
			ID:         value.User.ID,
			GivenName:  value.User.GivenName,
			FamilyName: value.User.FamilyName,
			Email:      value.User.Email,
		},
		Status:       string(value.Status),
		Roles:        roles,
		Permissions:  value.Permissions,
		RoleRequests: requests,
		CreatedAt:    value.CreatedAt,
		UpdatedAt:    value.UpdatedAt,
	}
}

func toRoleRequestResponse(value domain.RoleRequest) roleRequestResponse {
	return roleRequestResponse{
		ID:          value.ID,
		Role:        string(value.Role),
		Status:      string(value.Status),
		RequestedAt: value.RequestedAt,
	}
}
