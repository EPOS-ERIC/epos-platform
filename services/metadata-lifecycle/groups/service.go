// Package groups provides group metadata operations.
package groups

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	List(ctx context.Context) ([]domain.Group, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Group, error)
	Create(ctx context.Context, input domain.GroupInput) (domain.Group, error)
	Replace(ctx context.Context, id uuid.UUID, input domain.GroupInput) (domain.Group, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Authorizer interface {
	AuthorizeGlobal(context.Context, string, domain.GlobalPermission) (domain.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
	log        *slog.Logger
}

func New(log *slog.Logger, store Store, authorizer Authorizer) *Service {
	if log == nil {
		panic("groups: nil logger")
	}
	if store == nil {
		panic("groups: nil store")
	}
	if authorizer == nil {
		panic("groups: nil authorizer")
	}
	return &Service{
		store:      store,
		authorizer: authorizer,
		log:        log.With("component", "groups-service"),
	}
}

func (s *Service) Create(ctx context.Context, actorID string, input domain.GroupInput) (domain.Group, error) {
	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, domain.GlobalPermissionGroupManage)
	if err != nil {
		return domain.Group{}, fmt.Errorf("authorize group creation: %w", err)
	}
	if !allowed.Allowed {
		return domain.Group{}, apperror.Default(apperror.CodePermissionDenied)
	}

	group, err := s.store.Create(ctx, input)
	if err != nil {
		return domain.Group{}, fmt.Errorf("create group: %w", err)
	}

	return group, nil
}

func (s *Service) Replace(ctx context.Context, actorID string, id uuid.UUID, input domain.GroupInput) (domain.Group, error) {
	if id == uuid.Nil {
		return domain.Group{}, apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, domain.GlobalPermissionGroupManage)
	if err != nil {
		return domain.Group{}, fmt.Errorf("authorize group replacement: %w", err)
	}
	if !allowed.Allowed {
		return domain.Group{}, apperror.Default(apperror.CodePermissionDenied)
	}

	group, err := s.store.Replace(ctx, id, input)
	if err != nil {
		return domain.Group{}, fmt.Errorf("replace group: %w", err)
	}

	return group, nil
}

func (s *Service) Delete(ctx context.Context, actorID string, id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, domain.GlobalPermissionGroupManage)
	if err != nil {
		return fmt.Errorf("authorize group deletion: %w", err)
	}
	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	return nil
}

func (s *Service) List(ctx context.Context) ([]domain.Group, error) {
	groups, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	return groups, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.Group, error) {
	if id == uuid.Nil {
		return domain.Group{}, apperror.Default(apperror.CodeValidationFailed)
	}

	group, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Group{}, fmt.Errorf("get group: %w", err)
	}

	return group, nil
}
