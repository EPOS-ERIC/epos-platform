package handlers

import (
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func toAPIRoleRequest(v domain.RoleRequest) (gatewayapi.RoleRequest, error) {
	id, err := parseUUID("role_request.id", v.ID)
	if err != nil {
		return gatewayapi.RoleRequest{}, err
	}
	return gatewayapi.RoleRequest{
		ID:          id,
		Role:        gatewayapi.RoleRequestRole(v.Role),
		Status:      gatewayapi.RoleRequestStatus(v.Status),
		RequestedAt: v.RequestedAt,
	}, nil
}

func toAPIMembership(value metadatalifecycleclient.GroupMembership) (gatewayapi.GroupMembership, error) {
	userID, err := parseUUID("membership.user.id", value.User.Id)
	if err != nil {
		return gatewayapi.GroupMembership{}, err
	}
	result := gatewayapi.GroupMembership{
		User: gatewayapi.MembershipUser{
			ID:         userID.String(),
			GivenName:  value.User.GivenName,
			FamilyName: value.User.FamilyName,
			Email:      value.User.Email,
		},
		Status:      gatewayapi.MembershipStatus(value.Status),
		Permissions: value.Permissions,
		CreatedAt:   value.CreatedAt,
		UpdatedAt:   value.UpdatedAt,
	}
	for _, role := range value.Roles {
		result.Roles = append(result.Roles, gatewayapi.MembershipRole(role))
	}
	for _, request := range value.RoleRequests {
		requestID, err := parseUUID("membership.role_request.id", request.Id)
		if err != nil {
			return gatewayapi.GroupMembership{}, err
		}
		result.RoleRequests = append(result.RoleRequests, gatewayapi.RoleRequest{
			ID:          requestID,
			Role:        gatewayapi.RoleRequestRole(request.Role),
			Status:      gatewayapi.RoleRequestStatus(request.Status),
			RequestedAt: request.RequestedAt,
		})
	}
	return result, nil
}

func toAPIRole(value metadatalifecycleclient.RoleRequest) (gatewayapi.RoleRequest, error) {
	id, err := parseUUID("role_request.id", value.Id)
	if err != nil {
		return gatewayapi.RoleRequest{}, err
	}
	return gatewayapi.RoleRequest{
		ID:          id,
		Role:        gatewayapi.RoleRequestRole(value.Role),
		Status:      gatewayapi.RoleRequestStatus(value.Status),
		RequestedAt: value.RequestedAt,
	}, nil
}
