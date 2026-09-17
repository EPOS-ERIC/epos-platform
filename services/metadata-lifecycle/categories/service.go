// Package categories provides group category metadata operations.
package categories

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	List(context.Context, uuid.UUID) ([]domain.Category, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.Category, error)
	Create(context.Context, uuid.UUID, domain.CategoryInput) (domain.Category, error)
	Replace(context.Context, uuid.UUID, uuid.UUID, domain.CategoryInput) (domain.Category, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}

type Authorizer interface {
	AuthorizeGroup(context.Context, string, uuid.UUID, domain.GroupPermission) (domain.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
	log        *slog.Logger
}

func New(log *slog.Logger, store Store, authorizer Authorizer) *Service {
	return &Service{
		store:      store,
		authorizer: authorizer,
		log:        log.With("component", "categories-service"),
	}
}

func (s *Service) List(ctx context.Context, groupID uuid.UUID) ([]domain.Category, error) {
	if groupID == uuid.Nil {
		return nil, domain.ErrInvalidCategory
	}

	items, err := s.store.List(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	return items, nil
}

func (s *Service) Get(ctx context.Context, groupID, categoryID uuid.UUID) (domain.Category, error) {
	if groupID == uuid.Nil || categoryID == uuid.Nil {
		return domain.Category{}, domain.ErrInvalidCategory
	}

	item, err := s.store.Get(ctx, groupID, categoryID)
	if err != nil {
		return domain.Category{}, fmt.Errorf("get category: %w", err)
	}

	return item, nil
}

func (s *Service) Create(ctx context.Context, actorID domain.Actor, groupID uuid.UUID, input domain.CategoryInput) (domain.Category, error) {
	if groupID == uuid.Nil {
		return domain.Category{}, domain.ErrInvalidCategory
	}

	if err := s.authorize(ctx, actorID, groupID); err != nil {
		return domain.Category{}, err
	}

	item, err := s.store.Create(ctx, groupID, input)
	if err != nil {
		return domain.Category{}, fmt.Errorf("create category: %w", err)
	}

	return item, nil
}

func (s *Service) Replace(ctx context.Context, actorID domain.Actor, groupID, categoryID uuid.UUID, input domain.CategoryInput) (domain.Category, error) {
	if groupID == uuid.Nil || categoryID == uuid.Nil {
		return domain.Category{}, domain.ErrInvalidCategory
	}

	if err := s.authorize(ctx, actorID, groupID); err != nil {
		return domain.Category{}, err
	}

	item, err := s.store.Replace(ctx, groupID, categoryID, input)
	if err != nil {
		return domain.Category{}, fmt.Errorf("replace category: %w", err)
	}

	return item, nil
}

func (s *Service) Delete(ctx context.Context, actorID domain.Actor, groupID, categoryID uuid.UUID) error {
	if groupID == uuid.Nil || categoryID == uuid.Nil {
		return domain.ErrInvalidCategory
	}

	if err := s.authorize(ctx, actorID, groupID); err != nil {
		return err
	}

	if err := s.store.Delete(ctx, groupID, categoryID); err != nil {
		return fmt.Errorf("delete category: %w", err)
	}

	return nil
}

func (s *Service) authorize(ctx context.Context, actorID domain.Actor, groupID uuid.UUID) error {
	allowed, err := s.authorizer.AuthorizeGroup(ctx, string(actorID), groupID, domain.GroupPermissionCategoryManage)
	if err != nil {
		return fmt.Errorf("authorize category management: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrCategoryForbidden
	}
	return nil
}
