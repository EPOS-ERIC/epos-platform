// Package activity provides authenticated access to reconstructed activity events.
package activity

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

type Repository interface {
	ListActivity(context.Context, domain.ActivityFilter) ([]domain.ActivityEvent, error)
}

type Authorizer interface {
	AuthorizeGlobal(context.Context, string, domain.GlobalPermission) (domain.Result, error)
}

type Service struct {
	repository Repository
	authorizer Authorizer
	log        *slog.Logger
}

func New(log *slog.Logger, repository Repository, authorizer Authorizer) *Service {
	return &Service{
		repository: repository,
		authorizer: authorizer,
		log:        log.With("component", "activity-service"),
	}
}

func (s *Service) List(ctx context.Context, actor domain.Actor, filter domain.ActivityFilter) ([]domain.ActivityEvent, error) {
	filter, err := domain.NewActivityFilter(filter)
	if err != nil {
		return nil, err
	}

	decision, err := s.authorizer.AuthorizeGlobal(ctx, string(actor), domain.GlobalPermissionActivityRead)
	if err != nil {
		return nil, fmt.Errorf("authorize activity read: %w", err)
	}
	if !decision.Allowed {
		return nil, apperror.Default(apperror.CodePermissionDenied)
	}

	events, err := s.repository.ListActivity(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list activity: %w", err)
	}

	return events, nil
}
