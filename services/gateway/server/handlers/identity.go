package handlers

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) ListUsers(ctx context.Context) ([]gatewayapi.UserListItem, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	response, err := h.metadataLifecycle.ListUsers(ctx, actor)
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

	return gatewayapi.UserListItem{
		ID:                user.GetId(),
		Email:             user.GetEmail(),
		GivenName:         user.GetGivenName(),
		FamilyName:        user.GetFamilyName(),
		Status:            status,
		GlobalRoles:       globalRoles,
		GlobalPermissions: user.GetGlobalPermissions(),
		CreatedAt:         user.GetCreatedAt().AsTime(),
		UpdatedAt:         user.GetUpdatedAt().AsTime(),
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

func toAPIUser(v domain.User) (gatewayapi.User, error) {
	roles := make([]string, len(v.GlobalRoles))
	for i := range v.GlobalRoles {
		roles[i] = string(v.GlobalRoles[i])
	}
	r := gatewayapi.User{
		ID:                v.ID,
		Email:             v.Email,
		GivenName:         v.GivenName,
		FamilyName:        v.FamilyName,
		Status:            gatewayapi.UserStatus(v.Status),
		GlobalRoles:       roles,
		GlobalPermissions: v.GlobalPermissions,
	}
	for _, m := range v.Memberships {
		membership := gatewayapi.UserMembershipSummary{
			Group: gatewayapi.Reference{
				ID:   m.Group.ID,
				Name: m.Group.Name,
			},
			Status:      gatewayapi.MembershipStatus(m.Status),
			Permissions: m.GroupPermissions,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
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
	a, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
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
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.GrantGlobalPermission(ctx, a, p.ID, string(p.Permission)); err != nil {
		return fmt.Errorf("grant global permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) RevokeGlobalPermission(ctx context.Context, p gatewayapi.RevokeGlobalPermissionParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.RevokeGlobalPermission(ctx, a, p.ID, string(p.Permission)); err != nil {
		return fmt.Errorf("revoke global permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) AssignGlobalRole(ctx context.Context, p gatewayapi.AssignGlobalRoleParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	role, err := toProtoGlobalRole(p.Role)
	if err != nil {
		return err
	}
	if err := h.metadataLifecycle.AssignGlobalRole(ctx, a, p.ID, role); err != nil {
		return fmt.Errorf("assign global role: %w", err)
	}
	return nil
}

func (h *ogenHandler) RemoveGlobalRole(ctx context.Context, p gatewayapi.RemoveGlobalRoleParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	role, err := toProtoGlobalRole(p.Role)
	if err != nil {
		return err
	}
	if err := h.metadataLifecycle.RemoveGlobalRole(ctx, a, p.ID, role); err != nil {
		return fmt.Errorf("remove global role: %w", err)
	}
	return nil
}

func (h *ogenHandler) BlockUser(ctx context.Context, q *gatewayapi.BlockUserRequest, p gatewayapi.BlockUserParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.BlockUser(ctx, a, p.ID, string(q.Reason)); err != nil {
		return fmt.Errorf("block user: %w", err)
	}
	return nil
}

func (h *ogenHandler) UnblockUser(ctx context.Context, p gatewayapi.UnblockUserParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.UnblockUser(ctx, a, p.ID); err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}
	return nil
}
