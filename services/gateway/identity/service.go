// Package identity contains Gateway user and group identity use-cases.
package identity

import (
	"context"
	"fmt"
	"log/slog"

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

func (s *Service) GetUser(ctx context.Context, actorID, id string) (domain.User, error) {
	if id == "self" {
		id = actorID
	}

	if id == "" {
		return domain.User{}, fmt.Errorf("invalid user id: %w", domain.ErrBadRequest)
	}

	if s == nil || s.metadataLifecycle == nil || s.metadataLifecycle.Identity == nil {
		return domain.User{}, fmt.Errorf("metadata-lifecycle identity client is not configured")
	}

	user, err := s.metadataLifecycle.Identity.GetUserForAuth(ctx, actorID, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user from metadata-lifecycle: %w", err)
	}

	return user, nil
}
