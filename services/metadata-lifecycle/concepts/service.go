// Package concepts provides concept metadata operations.
package concepts

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	ListConcepts(context.Context) ([]domain.Concept, error)
	GetConcept(context.Context, uuid.UUID) (domain.Concept, error)
	CreateConcept(context.Context, domain.ConceptInput) (domain.Concept, error)
	ReplaceConcept(context.Context, uuid.UUID, domain.ConceptInput) (domain.Concept, error)
	DeleteConcept(context.Context, uuid.UUID) error
}

type Authorizer interface {
	AuthorizeGlobal(context.Context, string, domain.GlobalPermission) (domain.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
}

func New(store Store, authorizer Authorizer) *Service {
	if store == nil {
		panic("concepts: nil store")
	}
	if authorizer == nil {
		panic("concepts: nil authorizer")
	}
	return &Service{store: store, authorizer: authorizer}
}

func (s *Service) ListConcepts(ctx context.Context) ([]domain.Concept, error) {
	return s.store.ListConcepts(ctx)
}

func (s *Service) GetConcept(ctx context.Context, id uuid.UUID) (domain.Concept, error) {
	if id == uuid.Nil {
		return domain.Concept{}, apperror.Default(apperror.CodeValidationFailed)
	}

	return s.store.GetConcept(ctx, id)
}

func (s *Service) authorizeConcept(ctx context.Context, actor string) error {
	result, err := s.authorizer.AuthorizeGlobal(ctx, actor, domain.GlobalPermissionConceptManage)
	if err != nil {
		return fmt.Errorf("authorize concept management: %w", err)
	}

	if !result.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}

	return nil
}

func (s *Service) CreateConcept(ctx context.Context, actor string, input domain.ConceptInput) (domain.Concept, error) {
	if err := s.authorizeConcept(ctx, actor); err != nil {
		return domain.Concept{}, err
	}

	return s.store.CreateConcept(ctx, input)
}

func (s *Service) ReplaceConcept(ctx context.Context, actor string, id uuid.UUID, input domain.ConceptInput) (domain.Concept, error) {
	if id == uuid.Nil {
		return domain.Concept{}, apperror.Default(apperror.CodeValidationFailed)
	}

	if err := s.authorizeConcept(ctx, actor); err != nil {
		return domain.Concept{}, err
	}

	return s.store.ReplaceConcept(ctx, id, input)
}

func (s *Service) DeleteConcept(ctx context.Context, actor string, id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	if err := s.authorizeConcept(ctx, actor); err != nil {
		return err
	}

	return s.store.DeleteConcept(ctx, id)
}
