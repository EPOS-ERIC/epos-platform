// Package identity contains Gateway user and group identity use-cases.
package identity

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Service struct {
	metadataLifecycle *clients.MetadataLifecycle
	log               *slog.Logger
}

func NewService(log *slog.Logger, metadataLifecycle *clients.MetadataLifecycle) *Service {
	return &Service{
		metadataLifecycle: metadataLifecycle,
		log:               log.With("TODO-service", "identity"),
	}
}

func (s *Service) ListUsers(ctx context.Context, actorID string) ([]domain.UserSummary, error) {
	if s == nil || s.metadataLifecycle == nil {
		return nil, fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	users, err := s.metadataLifecycle.ListUsers(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("list users from metadata-lifecycle: %w", err)
	}

	return users, nil
}

func (s *Service) GrantGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	userID = strings.TrimSpace(userID)
	permission = strings.TrimSpace(permission)
	if userID == "" || permission == "" {
		return fmt.Errorf("invalid user id or permission: %w", domain.ErrBadRequest)
	}

	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	if err := s.metadataLifecycle.GrantGlobalPermission(ctx, actorID, userID, permission); err != nil {
		return fmt.Errorf("grant global permission through metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) RevokeGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	userID = strings.TrimSpace(userID)
	permission = strings.TrimSpace(permission)
	if userID == "" || permission == "" {
		return fmt.Errorf("invalid user id or permission: %w", domain.ErrBadRequest)
	}

	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	if err := s.metadataLifecycle.RevokeGlobalPermission(ctx, actorID, userID, permission); err != nil {
		return fmt.Errorf("revoke global permission through metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) AssignGlobalRole(ctx context.Context, actorID, userID, role string) error {
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)
	if userID == "" || role == "" {
		return fmt.Errorf("invalid user id or role: %w", domain.ErrBadRequest)
	}

	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	if err := s.metadataLifecycle.AssignGlobalRole(ctx, actorID, userID, role); err != nil {
		return fmt.Errorf("assign global role through metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) RemoveGlobalRole(ctx context.Context, actorID, userID, role string) error {
	userID = strings.TrimSpace(userID)
	role = strings.TrimSpace(role)
	if userID == "" || role == "" {
		return fmt.Errorf("invalid user id or role: %w", domain.ErrBadRequest)
	}

	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	if err := s.metadataLifecycle.RemoveGlobalRole(ctx, actorID, userID, role); err != nil {
		return fmt.Errorf("remove global role through metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) BlockUser(ctx context.Context, actorID, userID, reason string) error {
	userID = strings.TrimSpace(userID)
	reason = strings.TrimSpace(reason)
	if userID == "" || reason == "" {
		return fmt.Errorf("invalid user id or reason: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	if err := s.metadataLifecycle.BlockUser(ctx, actorID, userID, reason); err != nil {
		return fmt.Errorf("block user through metadata-lifecycle: %w", err)
	}

	return nil
}

func (s *Service) UnblockUser(ctx context.Context, actorID, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("invalid user id: %w", domain.ErrBadRequest)
	}
	if s == nil || s.metadataLifecycle == nil {
		return fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	if err := s.metadataLifecycle.UnblockUser(ctx, actorID, userID); err != nil {
		return fmt.Errorf("unblock user through metadata-lifecycle: %w", err)
	}

	return nil
}
