// Package identity contains Gateway user and group identity use-cases.
package identity

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
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
		return domain.User{}, fmt.Errorf("invalid user id: %w", apperror.Default(apperror.CodeValidationFailed))
	}

	user, err := s.metadataLifecycle.GetUser(ctx, actorID, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user from metadata-lifecycle: %w", err)
	}

	return user, nil
}

func (s *Service) EnsureUser(ctx context.Context, identity domain.ActorIdentity) (domain.User, error) {
	user, err := s.metadataLifecycle.GetUser(ctx, identity.AuthIdentifier, identity.AuthIdentifier)
	if apperror.IsCode(err, apperror.CodeNotFound) {
		err = s.metadataLifecycle.ProvisionUser(ctx, identity)
		if err == nil || apperror.IsCode(err, apperror.CodeConflict) {
			user, err = s.GetUser(ctx, identity.AuthIdentifier, identity.AuthIdentifier)
		}
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("ensure metadata-lifecycle user: %w", err)
	}

	return user, nil
}
