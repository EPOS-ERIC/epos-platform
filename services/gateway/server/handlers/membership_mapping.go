package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func toAPIProtoGroupMembership(value *metadatalifecyclev1.GroupMembership) (gatewayapi.GroupMembership, error) {
	if value == nil || value.User == nil {
		return gatewayapi.GroupMembership{}, fmt.Errorf("decode membership: response is empty")
	}
	userID, err := parseUUID("membership.user.id", value.User.Id)
	if err != nil {
		return gatewayapi.GroupMembership{}, err
	}
	result := gatewayapi.GroupMembership{User: gatewayapi.MembershipUser{ID: userID.String(), GivenName: value.User.GivenName, FamilyName: value.User.FamilyName, Email: value.User.Email}, Status: gatewayapi.MembershipStatus(value.Status), Permissions: value.Permissions}
	for _, role := range value.Roles {
		result.Roles = append(result.Roles, gatewayapi.MembershipRole(role))
	}
	for _, request := range value.RoleRequests {
		id, err := parseUUID("membership.role_request.id", request.Id)
		if err != nil {
			return gatewayapi.GroupMembership{}, err
		}
		role, err := toAPIRoleRequestRole(request.GroupRole)
		if err != nil {
			return gatewayapi.GroupMembership{}, err
		}
		status, err := toAPIRoleRequestStatus(request.Status)
		if err != nil {
			return gatewayapi.GroupMembership{}, err
		}
		result.RoleRequests = append(result.RoleRequests, gatewayapi.RoleRequest{ID: id, Role: role, Status: status, RequestedAt: request.RequestedAt.AsTime()})
	}
	if value.CreatedAt != nil {
		result.CreatedAt = value.CreatedAt.AsTime()
	}
	if value.UpdatedAt != nil {
		result.UpdatedAt = value.UpdatedAt.AsTime()
	}
	return result, nil
}

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

func toAPIProtoRoleRequest(v *metadatalifecyclev1.RoleRequest) (gatewayapi.RoleRequest, error) {
	if v == nil {
		return gatewayapi.RoleRequest{}, fmt.Errorf("decode role request: response is empty")
	}
	id, err := parseUUID("role_request.id", v.Id)
	if err != nil {
		return gatewayapi.RoleRequest{}, err
	}
	role, err := toAPIRoleRequestRole(v.GroupRole)
	if err != nil {
		return gatewayapi.RoleRequest{}, err
	}
	status, err := toAPIRoleRequestStatus(v.Status)
	if err != nil {
		return gatewayapi.RoleRequest{}, err
	}
	return gatewayapi.RoleRequest{ID: id, Role: role, Status: status, RequestedAt: v.RequestedAt.AsTime()}, nil
}

func toAPIRoleRequestRole(value metadatalifecyclev1.GroupRole) (gatewayapi.RoleRequestRole, error) {
	switch value {
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_EDITOR:
		return gatewayapi.RoleRequestRoleEDITOR, nil
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_REVIEWER:
		return gatewayapi.RoleRequestRoleREVIEWER, nil
	default:
		return "", fmt.Errorf("unsupported role request role: %s", value.String())
	}
}

func toAPIRoleRequestStatus(value metadatalifecyclev1.RoleRequestStatus) (gatewayapi.RoleRequestStatus, error) {
	switch value {
	case metadatalifecyclev1.RoleRequestStatus_ROLE_REQUEST_STATUS_PENDING:
		return gatewayapi.RoleRequestStatusPENDING, nil
	case metadatalifecyclev1.RoleRequestStatus_ROLE_REQUEST_STATUS_ACCEPTED:
		return gatewayapi.RoleRequestStatusACCEPTED, nil
	case metadatalifecyclev1.RoleRequestStatus_ROLE_REQUEST_STATUS_REJECTED:
		return gatewayapi.RoleRequestStatusREJECTED, nil
	default:
		return "", fmt.Errorf("unsupported role request status: %s", value.String())
	}
}
