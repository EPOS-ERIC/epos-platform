//nolint:dupl,exhaustruct_v5,wrapcheck
package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) ListUsers(ctx context.Context) ([]gatewayapi.UserSummary, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.identity.ListUsers(ctx, actor)
	if err != nil {
		return nil, err
	}
	out := []gatewayapi.UserSummary{}
	for _, x := range v {
		roles := make([]string, len(x.GlobalRoles))
		for i := range x.GlobalRoles {
			roles[i] = string(x.GlobalRoles[i])
		}
		out = append(out, gatewayapi.UserSummary{
			ID:                x.ID,
			Email:             x.Email,
			GivenName:         x.GivenName,
			FamilyName:        x.FamilyName,
			Status:            gatewayapi.UserStatus(x.Status),
			GlobalRoles:       roles,
			GlobalPermissions: x.GlobalPermissions,
		})
	}
	return out, nil
}

func mapUser(v domain.User) (gatewayapi.User, error) {
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
			mapped, err := mapRole(request)
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
		return nil, err
	}
	r, err := mapUser(v)
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
	if err := h.identity.GrantGlobalPermission(ctx, a, p.ID.String(), string(p.Permission)); err != nil {
		return err
	}
	return nil
}

func (h *ogenHandler) RevokeGlobalPermission(ctx context.Context, p gatewayapi.RevokeGlobalPermissionParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.identity.RevokeGlobalPermission(ctx, a, p.ID.String(), string(p.Permission)); err != nil {
		return err
	}
	return nil
}

func (h *ogenHandler) AssignGlobalRole(ctx context.Context, p gatewayapi.AssignGlobalRoleParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.identity.AssignGlobalRole(ctx, a, p.ID.String(), string(p.Role)); err != nil {
		return err
	}
	return nil
}

func (h *ogenHandler) RemoveGlobalRole(ctx context.Context, p gatewayapi.RemoveGlobalRoleParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.identity.RemoveGlobalRole(ctx, a, p.ID.String(), string(p.Role)); err != nil {
		return err
	}
	return nil
}

func (h *ogenHandler) BlockUser(ctx context.Context, q *gatewayapi.BlockUserRequest, p gatewayapi.BlockUserParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.identity.BlockUser(ctx, a, p.ID.String(), string(q.Reason)); err != nil {
		return err
	}
	return nil
}

func (h *ogenHandler) UnblockUser(ctx context.Context, p gatewayapi.UnblockUserParams) error {
	a, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if err := h.identity.UnblockUser(ctx, a, p.ID.String()); err != nil {
		return err
	}
	return nil
}
