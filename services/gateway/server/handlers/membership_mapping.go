package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func toProtoMembershipRole(value gatewayapi.CreateMembershipRequestRolesItem) (metadatalifecyclev1.GroupRole, error) {
	switch value {
	case gatewayapi.CreateMembershipRequestRolesItemEDITOR:
		return metadatalifecyclev1.GroupRole_GROUP_ROLE_EDITOR, nil
	case gatewayapi.CreateMembershipRequestRolesItemREVIEWER:
		return metadatalifecyclev1.GroupRole_GROUP_ROLE_REVIEWER, nil
	default:
		return 0, fmt.Errorf("invalid membership role %q", value)
	}
}

func toProtoGroupRole(value gatewayapi.GroupRole) (metadatalifecyclev1.GroupRole, error) {
	switch value {
	case gatewayapi.GroupRoleEDITOR:
		return metadatalifecyclev1.GroupRole_GROUP_ROLE_EDITOR, nil
	case gatewayapi.GroupRoleREVIEWER:
		return metadatalifecyclev1.GroupRole_GROUP_ROLE_REVIEWER, nil
	case gatewayapi.GroupRoleGROUPADMIN:
		return metadatalifecyclev1.GroupRole_GROUP_ROLE_ADMIN, nil
	default:
		return 0, fmt.Errorf("invalid group role %q", value)
	}
}

func toProtoRoleRequestRole(value gatewayapi.CreateRoleRequestRole) (metadatalifecyclev1.GroupRole, error) {
	switch value {
	case gatewayapi.CreateRoleRequestRoleEDITOR:
		return metadatalifecyclev1.GroupRole_GROUP_ROLE_EDITOR, nil
	case gatewayapi.CreateRoleRequestRoleREVIEWER:
		return metadatalifecyclev1.GroupRole_GROUP_ROLE_REVIEWER, nil
	default:
		return 0, fmt.Errorf("invalid role request role %q", value)
	}
}

func toProtoDecision(value gatewayapi.Decision) (metadatalifecyclev1.Decision, error) {
	switch value {
	case gatewayapi.DecisionAccept:
		return metadatalifecyclev1.Decision_DECISION_ACCEPT, nil
	case gatewayapi.DecisionReject:
		return metadatalifecyclev1.Decision_DECISION_REJECT, nil
	default:
		return 0, fmt.Errorf("invalid decision %q", value)
	}
}

func toAPIProtoGroupMembership(value *metadatalifecyclev1.GroupMembership) (gatewayapi.GroupMembership, error) {
	if value == nil || value.User == nil {
		return gatewayapi.GroupMembership{}, fmt.Errorf("decode membership: response is empty")
	}
	status, err := toAPIMembershipStatus(value.Status)
	if err != nil {
		return gatewayapi.GroupMembership{}, err
	}
	grantedPermissions, err := toAPIProtoGroupPermissions(value.GrantedPermissions)
	if err != nil {
		return gatewayapi.GroupMembership{}, err
	}
	rolePermissions, err := toAPIProtoGroupPermissions(value.RolePermissions)
	if err != nil {
		return gatewayapi.GroupMembership{}, err
	}
	result := gatewayapi.GroupMembership{User: gatewayapi.MembershipUser{ID: value.User.Id, GivenName: value.User.GivenName, FamilyName: value.User.FamilyName, Email: value.User.Email}, Status: status, GrantedPermissions: grantedPermissions, RolePermissions: rolePermissions}
	for _, role := range value.Roles {
		mapped, err := toAPIGroupMembershipRole(role)
		if err != nil {
			return gatewayapi.GroupMembership{}, err
		}
		result.Roles = append(result.Roles, mapped)
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

func toAPIGroupPermissions(values []domain.GroupPermission) ([]gatewayapi.GroupPermission, error) {
	permissions := make([]gatewayapi.GroupPermission, len(values))
	for i, value := range values {
		permission := gatewayapi.GroupPermission(value)
		if err := permission.Validate(); err != nil {
			return nil, fmt.Errorf("metadata-lifecycle returned unknown group permission %q: %w", value, err)
		}
		permissions[i] = permission
	}
	return permissions, nil
}

func toAPIProtoGroupPermissions(values []metadatalifecyclev1.GroupPermission) ([]gatewayapi.GroupPermission, error) {
	permissions := make([]gatewayapi.GroupPermission, len(values))
	for i, value := range values {
		var permission gatewayapi.GroupPermission
		switch value {
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_CREATE:
			permission = gatewayapi.GroupPermissionResourceCreate
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_EDIT:
			permission = gatewayapi.GroupPermissionResourceEdit
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_SUBMIT:
			permission = gatewayapi.GroupPermissionResourceSubmit
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_WITHDRAW:
			permission = gatewayapi.GroupPermissionResourceWithdraw
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_REOPEN:
			permission = gatewayapi.GroupPermissionResourceReopen
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_DISCARD:
			permission = gatewayapi.GroupPermissionResourceDiscard
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_REJECT:
			permission = gatewayapi.GroupPermissionResourceReject
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_PUBLISH:
			permission = gatewayapi.GroupPermissionResourcePublish
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_RESOURCE_UNPUBLISH:
			permission = gatewayapi.GroupPermissionResourceUnpublish
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_MEMBERSHIP_MANAGE:
			permission = gatewayapi.GroupPermissionGroupMembershipManage
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_ROLE_MANAGE:
			permission = gatewayapi.GroupPermissionGroupRoleManage
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_PERMISSION_MANAGE:
			permission = gatewayapi.GroupPermissionGroupPermissionManage
		case metadatalifecyclev1.GroupPermission_GROUP_PERMISSION_CATEGORY_MANAGE:
			permission = gatewayapi.GroupPermissionCategoryManage
		default:
			return nil, fmt.Errorf("metadata-lifecycle returned unknown group permission: %v", value)
		}
		permissions[i] = permission
	}
	return permissions, nil
}

func toAPIMembershipRole(value domain.GroupRole) (gatewayapi.MembershipRole, error) {
	switch value {
	case domain.GroupRoleEditor:
		return gatewayapi.MembershipRoleEDITOR, nil
	case domain.GroupRoleReviewer:
		return gatewayapi.MembershipRoleREVIEWER, nil
	case domain.GroupRoleAdmin:
		return gatewayapi.MembershipRoleGROUPADMIN, nil
	default:
		return "", fmt.Errorf("unsupported membership role: %s", value)
	}
}

func toAPIGroupMembershipRole(value metadatalifecyclev1.GroupRole) (gatewayapi.MembershipRole, error) {
	switch value {
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_EDITOR:
		return gatewayapi.MembershipRoleEDITOR, nil
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_REVIEWER:
		return gatewayapi.MembershipRoleREVIEWER, nil
	case metadatalifecyclev1.GroupRole_GROUP_ROLE_ADMIN:
		return gatewayapi.MembershipRoleGROUPADMIN, nil
	default:
		return "", fmt.Errorf("unsupported membership role: %s", value.String())
	}
}

func toAPIMembershipStatus(value metadatalifecyclev1.MembershipStatus) (gatewayapi.MembershipStatus, error) {
	switch value {
	case metadatalifecyclev1.MembershipStatus_MEMBERSHIP_STATUS_PENDING:
		return gatewayapi.MembershipStatusPENDING, nil
	case metadatalifecyclev1.MembershipStatus_MEMBERSHIP_STATUS_ACTIVE:
		return gatewayapi.MembershipStatusACTIVE, nil
	case metadatalifecyclev1.MembershipStatus_MEMBERSHIP_STATUS_REJECTED:
		return gatewayapi.MembershipStatusREJECTED, nil
	default:
		return "", fmt.Errorf("unsupported membership status: %s", value.String())
	}
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
