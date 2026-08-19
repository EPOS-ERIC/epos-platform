// Package identity contains Gateway user and group identity use-cases.
package identity

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

type Service struct {
	backoffice *clients.Backoffice
}

func NewService(backoffice *clients.Backoffice) *Service {
	return &Service{backoffice: backoffice}
}

func (s *Service) GetUser(ctx context.Context, actor api.User, id string) (api.User, error) {
	if id == "self" {
		id = actor.ID
	}
	if _, err := uuid.Parse(id); err != nil {
		return api.User{}, fmt.Errorf("invalid user id: %w: %w", domain.ErrBadRequest, err)
	}
	if s == nil || s.backoffice == nil {
		return api.User{}, fmt.Errorf("backoffice identity client is not configured")
	}
	user, err := s.backoffice.GetUserByID(ctx, id, actor.ID)
	if err != nil {
		return api.User{}, fmt.Errorf("get user from backoffice: %w", err)
	}
	return UserFromClient(user), nil
}

func UserFromClient(resolved clients.User) api.User {
	user := api.User{
		ID:                resolved.ID,
		AuthIdentifier:    resolved.AuthIdentifier,
		Email:             resolved.Email,
		GivenName:         resolved.GivenName,
		FamilyName:        resolved.FamilyName,
		Status:            api.UserStatus(resolved.Status),
		GlobalRoles:       append([]string{}, resolved.GlobalRoles...),
		GlobalPermissions: append([]string{}, resolved.GlobalPermissions...),
		CreatedAt:         resolved.CreatedAt,
		UpdatedAt:         resolved.UpdatedAt,
		Memberships:       make([]api.UserMembershipSummary, 0, len(resolved.Memberships)),
	}
	for _, membership := range resolved.Memberships {
		roles := make([]api.MembershipRole, 0, len(membership.Roles))
		for _, role := range membership.Roles {
			roles = append(roles, api.MembershipRole(role))
		}
		user.Memberships = append(user.Memberships, api.UserMembershipSummary{
			Group: api.ResourceRef{
				ID:   membership.Group.ID,
				Name: membership.Group.Name,
				Href: "/v1/groups/" + membership.Group.ID,
			},
			Roles:       roles,
			Status:      api.MembershipStatus(membership.Status),
			Permissions: append([]string{}, membership.Permissions...),
			CreatedAt:   membership.CreatedAt,
			UpdatedAt:   membership.UpdatedAt,
		})
	}
	return user
}
