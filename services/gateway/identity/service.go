// Package identity contains Gateway user and group identity use-cases.
package identity

import (
	"context"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Service struct {
	backoffice *clients.Backoffice
}

func NewService(backoffice *clients.Backoffice) *Service {
	return &Service{backoffice: backoffice}
}

func (s *Service) GetUser(ctx context.Context, actorID, id string) (domain.User, error) {
	if id == "self" {
		id = actorID
	}

	if id == "" {
		return domain.User{}, fmt.Errorf("invalid user id: %w", domain.ErrBadRequest)
	}

	if s == nil || s.backoffice == nil {
		return domain.User{}, fmt.Errorf("backoffice identity client is not configured")
	}

	user, err := s.backoffice.GetUserByID(ctx, id, actorID)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user from backoffice: %w", err)
	}

	return user, nil
}

func (s *Service) ListUsers(ctx context.Context, actorID string) ([]domain.UserSummary, error) {
	if s == nil || s.backoffice == nil {
		return nil, fmt.Errorf("backoffice identity client is not configured")
	}

	users, err := s.backoffice.ListUsers(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("list users from backoffice: %w", err)
	}

	return users, nil
}

func (s *Service) GrantGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	userID = strings.TrimSpace(userID)
	permission = strings.TrimSpace(permission)
	if userID == "" || permission == "" {
		return fmt.Errorf("invalid user id or permission: %w", domain.ErrBadRequest)
	}

	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice identity client is not configured")
	}

	if err := s.backoffice.GrantGlobalPermission(ctx, actorID, userID, permission); err != nil {
		return fmt.Errorf("grant global permission through backoffice: %w", err)
	}

	return nil
}

func (s *Service) RevokeGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	userID = strings.TrimSpace(userID)
	permission = strings.TrimSpace(permission)
	if userID == "" || permission == "" {
		return fmt.Errorf("invalid user id or permission: %w", domain.ErrBadRequest)
	}

	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice identity client is not configured")
	}

	if err := s.backoffice.RevokeGlobalPermission(ctx, actorID, userID, permission); err != nil {
		return fmt.Errorf("revoke global permission through backoffice: %w", err)
	}

	return nil
}
