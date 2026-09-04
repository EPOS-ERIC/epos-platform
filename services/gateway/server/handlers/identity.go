package handlers

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func (h *ogenHandler) ListUsers(ctx context.Context) ([]gatewayapi.UserSummary, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	users, err := h.metadataLifecycle.ListUsers(ctx, actor)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	out := make([]gatewayapi.UserSummary, len(users))
	for i, user := range users {
		out[i] = toAPIUserSummary(user)
	}
	return out, nil
}

func toAPIUserSummary(user metadatalifecycleclient.UserSummary) gatewayapi.UserSummary {
	return gatewayapi.UserSummary{
		ID:                user.Id,
		Email:             user.Email,
		GivenName:         user.GivenName,
		FamilyName:        user.FamilyName,
		Status:            gatewayapi.UserStatus(user.Status),
		GlobalRoles:       user.GlobalRoles,
		GlobalPermissions: user.GlobalPermissions,
	}
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
			membership.Roles = append(membership.Roles, gatewayapi.MembershipRole(role))
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
	if err := h.metadataLifecycle.GrantGlobalPermission(ctx, a, p.ID.String(), string(p.Permission)); err != nil {
		return fmt.Errorf("grant global permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) RevokeGlobalPermission(ctx context.Context, p gatewayapi.RevokeGlobalPermissionParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.RevokeGlobalPermission(ctx, a, p.ID.String(), string(p.Permission)); err != nil {
		return fmt.Errorf("revoke global permission: %w", err)
	}
	return nil
}

func (h *ogenHandler) AssignGlobalRole(ctx context.Context, p gatewayapi.AssignGlobalRoleParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.AssignGlobalRole(ctx, a, p.ID.String(), string(p.Role)); err != nil {
		return fmt.Errorf("assign global role: %w", err)
	}
	return nil
}

func (h *ogenHandler) RemoveGlobalRole(ctx context.Context, p gatewayapi.RemoveGlobalRoleParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.RemoveGlobalRole(ctx, a, p.ID.String(), string(p.Role)); err != nil {
		return fmt.Errorf("remove global role: %w", err)
	}
	return nil
}

func (h *ogenHandler) BlockUser(ctx context.Context, q *gatewayapi.BlockUserRequest, p gatewayapi.BlockUserParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.BlockUser(ctx, a, p.ID.String(), string(q.Reason)); err != nil {
		return fmt.Errorf("block user: %w", err)
	}
	return nil
}

func (h *ogenHandler) UnblockUser(ctx context.Context, p gatewayapi.UnblockUserParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.metadataLifecycle.UnblockUser(ctx, a, p.ID.String()); err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}
	return nil
}
