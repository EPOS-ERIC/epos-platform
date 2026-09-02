package users

import (
	"fmt"
	"slices"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

func toUserClaims(value internalUserRequest) (domain.UserClaims, error) {
	claims, err := domain.NewUserClaims(domain.UserClaims{
		ID:         value.ID,
		Email:      value.Email,
		GivenName:  value.GivenName,
		FamilyName: value.FamilyName,
	})
	if err != nil {
		return domain.UserClaims{}, fmt.Errorf("map user request: %w", err)
	}

	return claims, nil
}

func toUserResponse(value domain.User) userResponse {
	memberships := make([]userMembershipResponse, 0, len(value.Memberships))
	for _, membership := range value.Memberships {
		memberships = append(memberships, toUserMembershipResponse(membership))
	}

	return userResponse{
		ID:         value.ID,
		Email:      value.Email,
		GivenName:  value.GivenName,
		FamilyName: value.FamilyName,
		Status:     string(value.Status),
		// GlobalRoles:       slices.Clone(value.GlobalRoles),
		GlobalPermissions: slices.Clone(value.GlobalPermissions),
		CreatedAt:         value.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:         value.UpdatedAt.Format(time.RFC3339Nano),
		Memberships:       memberships,
	}
}

func toUserSummaryResponse(value domain.UserSummary) userSummaryResponse {
	return userSummaryResponse{
		ID:         value.ID,
		Email:      value.Email,
		GivenName:  value.GivenName,
		FamilyName: value.FamilyName,
		Status:     string(value.Status),
		// GlobalRoles:       slices.Clone(value.GlobalRoles),
		GlobalPermissions: slices.Clone(value.GlobalPermissions),
	}
}

func toUserMembershipResponse(value domain.Membership) userMembershipResponse {
	roles := make([]string, 0, len(value.Roles))
	for _, role := range value.Roles {
		roles = append(roles, string(role))
	}

	roleRequests := make([]roleRequestResponse, 0, len(value.RoleRequests))
	for _, request := range value.RoleRequests {
		roleRequests = append(roleRequests, toRoleRequestResponse(request))
	}

	return userMembershipResponse{
		Group: groupResponse{
			ID:   value.Group.ID,
			Name: value.Group.Name,
		},
		Roles:        roles,
		Status:       string(value.Status),
		Permissions:  slices.Clone(value.Permissions),
		RoleRequests: roleRequests,
		CreatedAt:    value.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:    value.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func toRoleRequestResponse(value domain.RoleRequest) roleRequestResponse {
	return roleRequestResponse{
		ID:          value.ID,
		Role:        string(value.Role),
		Status:      string(value.Status),
		RequestedAt: value.RequestedAt.Format(time.RFC3339Nano),
	}
}
