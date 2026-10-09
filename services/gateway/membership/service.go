// Package membership contains Gateway group membership use-cases.
package membership

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
)

type Service struct {
	metadataLifecycle *clients.MetadataLifecycle
	log               *slog.Logger
}

func NewService(log *slog.Logger, metadataLifecycle *clients.MetadataLifecycle) *Service {
	return &Service{
		metadataLifecycle: metadataLifecycle,
		log:               log.With("TODO-service", "membership"),
	}
}

func (s *Service) RevokeMembership(ctx context.Context, actorID, groupID, userID string) error {
	// TODO: move the self resolution in the metadataLifecycle and remove this useless service
	if userID == "self" {
		userID = actorID
	}
	if userID == "" {
		return fmt.Errorf("invalid user id: %w", apperror.Default(apperror.CodeValidationFailed))
	}

	if err := s.metadataLifecycle.RevokeMembership(ctx, groupID, userID, actorID); err != nil {
		return fmt.Errorf("revoke membership from metadata-lifecycle: %w", err)
	}
	return nil
}
