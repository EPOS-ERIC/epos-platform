package handlers

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) ListUsers(ctx context.Context) ([]gatewayapi.UserListItem, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	response, err := h.metadataLifecycleClient.ListUsers(ctx, actor)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("list users: response is empty")
	}

	out := make([]gatewayapi.UserListItem, len(response.GetUsers()))
	for i, user := range response.GetUsers() {
		out[i], err = toAPIUserListItem(user)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func toAPIUserListItem(user *metadatalifecyclev1.UserListItem) (gatewayapi.UserListItem, error) {
	if user == nil {
		return gatewayapi.UserListItem{}, fmt.Errorf("list users: response contains nil user")
	}

	status, err := toAPIUserStatus(user.GetStatus())
	if err != nil {
		return gatewayapi.UserListItem{}, err
	}
	globalRoles, err := toAPIGlobalRoles(user.GetGlobalRoles())
	if err != nil {
		return gatewayapi.UserListItem{}, err
	}
	globalGrantedPermissions, err := toAPIProtoGlobalPermissions(user.GetGlobalGrantedPermissions())
	if err != nil {
		return gatewayapi.UserListItem{}, err
	}
	globalRolePermissions, err := toAPIProtoGlobalPermissions(user.GetGlobalRolePermissions())
	if err != nil {
		return gatewayapi.UserListItem{}, err
	}

	return gatewayapi.UserListItem{
		ID:                       user.GetId(),
		Email:                    user.GetEmail(),
		GivenName:                user.GetGivenName(),
		FamilyName:               user.GetFamilyName(),
		Status:                   status,
		GlobalRoles:              globalRoles,
		GlobalGrantedPermissions: globalGrantedPermissions,
		GlobalRolePermissions:    globalRolePermissions,
		CreatedAt:                user.GetCreatedAt().AsTime(),
		UpdatedAt:                user.GetUpdatedAt().AsTime(),
	}, nil
}

func toProtoGlobalRole(value gatewayapi.GlobalRole) (metadatalifecyclev1.GlobalRole, error) {
	switch value {
	case gatewayapi.GlobalRoleADMIN:
		return metadatalifecyclev1.GlobalRole_GLOBAL_ROLE_ADMIN, nil
	default:
		return 0, fmt.Errorf("invalid global role %q", value)
	}
}

func toAPIUserStatus(value metadatalifecyclev1.UserStatus) (gatewayapi.UserStatus, error) {
	switch value {
	case metadatalifecyclev1.UserStatus_USER_STATUS_ACTIVE:
		return gatewayapi.UserStatus("ACTIVE"), nil
	case metadatalifecyclev1.UserStatus_USER_STATUS_BLOCKED:
		return gatewayapi.UserStatus("BLOCKED"), nil
	default:
		return "", fmt.Errorf("metadata-lifecycle returned unknown user status: %v", value)
	}
}

func toAPIGlobalRoles(values []metadatalifecyclev1.GlobalRole) ([]string, error) {
	roles := make([]string, len(values))
	for i, value := range values {
		switch value {
		case metadatalifecyclev1.GlobalRole_GLOBAL_ROLE_ADMIN:
			roles[i] = "ADMIN"
		default:
			return nil, fmt.Errorf("metadata-lifecycle returned unknown global role: %v", value)
		}
	}
	return roles, nil
}

func toAPIGlobalPermissions(values []domain.GlobalPermission) ([]gatewayapi.GlobalPermission, error) {
	permissions := make([]gatewayapi.GlobalPermission, len(values))
	for i, value := range values {
		permission := gatewayapi.GlobalPermission(value)
		if err := permission.Validate(); err != nil {
			return nil, fmt.Errorf("metadata-lifecycle returned unknown global permission %q: %w", value, err)
		}
		permissions[i] = permission
	}
	return permissions, nil
}

func toAPIProtoGlobalPermissions(values []metadatalifecyclev1.GlobalPermission) ([]gatewayapi.GlobalPermission, error) {
	permissions := make([]gatewayapi.GlobalPermission, len(values))
	for i, value := range values {
		var permission gatewayapi.GlobalPermission
		switch value {
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_MEMBERSHIP_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalMembershipManage
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_ROLE_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalRoleManage
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_PERMISSION_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalPermissionManage
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_USER_API_TOKEN_REVOKE:
			permission = gatewayapi.GlobalPermissionGlobalUserAPITokenRevoke
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_NOTIFICATION_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalNotificationManage
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_ORGANIZATION_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalOrganizationManage
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CONTACT_POINT_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalContactPointManage
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_GROUP_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalGroupManage
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CONCEPT_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalConceptManage
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_CREATE:
			permission = gatewayapi.GlobalPermissionGlobalResourceCreate
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_EDIT:
			permission = gatewayapi.GlobalPermissionGlobalResourceEdit
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_SUBMIT:
			permission = gatewayapi.GlobalPermissionGlobalResourceSubmit
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_WITHDRAW:
			permission = gatewayapi.GlobalPermissionGlobalResourceWithdraw
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_REOPEN:
			permission = gatewayapi.GlobalPermissionGlobalResourceReopen
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_DISCARD:
			permission = gatewayapi.GlobalPermissionGlobalResourceDiscard
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_REJECT:
			permission = gatewayapi.GlobalPermissionGlobalResourceReject
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_PUBLISH:
			permission = gatewayapi.GlobalPermissionGlobalResourcePublish
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_RESOURCE_UNPUBLISH:
			permission = gatewayapi.GlobalPermissionGlobalResourceUnpublish
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CATEGORY_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalCategoryManage
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CONFIGURATION_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalConfigurationManage
		case metadatalifecyclev1.GlobalPermission_GLOBAL_PERMISSION_CONVERTER_MANAGE:
			permission = gatewayapi.GlobalPermissionGlobalConverterManage
		default:
			return nil, fmt.Errorf("metadata-lifecycle returned unknown global permission: %v", value)
		}
		permissions[i] = permission
	}
	return permissions, nil
}

func toAPIUser(v domain.User) (gatewayapi.User, error) {
	roles := make([]string, len(v.GlobalRoles))
	for i := range v.GlobalRoles {
		roles[i] = string(v.GlobalRoles[i])
	}
	globalGrantedPermissions, err := toAPIGlobalPermissions(v.GlobalGrantedPermissions)
	if err != nil {
		return gatewayapi.User{}, err
	}
	globalRolePermissions, err := toAPIGlobalPermissions(v.GlobalRolePermissions)
	if err != nil {
		return gatewayapi.User{}, err
	}
	r := gatewayapi.User{
		ID:                       v.ID,
		Email:                    v.Email,
		GivenName:                v.GivenName,
		FamilyName:               v.FamilyName,
		Status:                   gatewayapi.UserStatus(v.Status),
		GlobalRoles:              roles,
		GlobalGrantedPermissions: globalGrantedPermissions,
		GlobalRolePermissions:    globalRolePermissions,
		BlockReason:              optionalString(v.BlockReason),
		CreatedAt:                v.CreatedAt,
		UpdatedAt:                v.UpdatedAt,
	}
	for _, m := range v.Memberships {
		grantedPermissions, err := toAPIGroupPermissions(m.GrantedPermissions)
		if err != nil {
			return gatewayapi.User{}, err
		}
		rolePermissions, err := toAPIGroupPermissions(m.RolePermissions)
		if err != nil {
			return gatewayapi.User{}, err
		}
		membership := gatewayapi.UserMembershipSummary{
			Group: gatewayapi.Reference{
				ID:   m.Group.ID,
				Name: m.Group.Name,
			},
			Status:             gatewayapi.MembershipStatus(m.Status),
			GrantedPermissions: grantedPermissions,
			RolePermissions:    rolePermissions,
			CreatedAt:          m.CreatedAt,
			UpdatedAt:          m.UpdatedAt,
		}
		for _, role := range m.Roles {
			mapped, err := toAPIMembershipRole(role)
			if err != nil {
				return gatewayapi.User{}, err
			}
			membership.Roles = append(membership.Roles, mapped)
		}
		for _, request := range m.RoleRequests {
			mapped, err := toAPIRoleRequest(request)
			if err != nil {
				return gatewayapi.User{}, err
			}
			membership.RoleRequests = append(membership.RoleRequests, mapped)
		}
		r.Memberships = append(r.Memberships, membership)
	}
	return r, nil
}

func (h *ogenHandler) GetUser(ctx context.Context, p gatewayapi.GetUserParams) (*gatewayapi.User, error) {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	v, err := h.identity.GetUser(ctx, a, p.ID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	r, err := toAPIUser(v)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (h *ogenHandler) GrantGlobalPermission(ctx context.Context, p gatewayapi.GrantGlobalPermissionParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	permission, err := domain.NewGlobalPermission(string(p.Permission))
	if err != nil {
		return fmt.Errorf("map global permission: %w", err)
	}
	if err := h.metadataLifecycleClient.GrantGlobalPermission(ctx, a, p.ID, permission); err != nil {
		return fmt.Errorf("grant global permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) RevokeGlobalPermission(ctx context.Context, p gatewayapi.RevokeGlobalPermissionParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	permission, err := domain.NewGlobalPermission(string(p.Permission))
	if err != nil {
		return fmt.Errorf("map global permission: %w", err)
	}
	if err := h.metadataLifecycleClient.RevokeGlobalPermission(ctx, a, p.ID, permission); err != nil {
		return fmt.Errorf("revoke global permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) AssignGlobalRole(ctx context.Context, p gatewayapi.AssignGlobalRoleParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	role, err := toProtoGlobalRole(p.Role)
	if err != nil {
		return err
	}
	if err := h.metadataLifecycleClient.AssignGlobalRole(ctx, a, p.ID, role); err != nil {
		return fmt.Errorf("assign global role: %w", err)
	}
	return nil
}

func (h *ogenHandler) RemoveGlobalRole(ctx context.Context, p gatewayapi.RemoveGlobalRoleParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	role, err := toProtoGlobalRole(p.Role)
	if err != nil {
		return err
	}
	if err := h.metadataLifecycleClient.RemoveGlobalRole(ctx, a, p.ID, role); err != nil {
		return fmt.Errorf("remove global role: %w", err)
	}
	return nil
}

func (h *ogenHandler) BlockUser(ctx context.Context, q *gatewayapi.BlockUserRequest, p gatewayapi.BlockUserParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	if err := h.metadataLifecycleClient.BlockUser(ctx, a, p.ID, q.Reason); err != nil {
		return fmt.Errorf("block user: %w", err)
	}
	return nil
}

func (h *ogenHandler) UnblockUser(ctx context.Context, p gatewayapi.UnblockUserParams) error {
	a, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	if err := h.metadataLifecycleClient.UnblockUser(ctx, a, p.ID); err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}
	return nil
}
