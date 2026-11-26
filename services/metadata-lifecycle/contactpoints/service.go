// Package contactpoints provides protected canonical contact-point reads.
package contactpoints

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	List(context.Context) ([]domain.ContactPoint, error)
	Get(context.Context, uuid.UUID) (domain.ContactPoint, error)
	Create(context.Context, domain.ContactPointInput) (domain.ContactPoint, error)
	Replace(context.Context, uuid.UUID, domain.ContactPointInput) (domain.ContactPoint, error)
	Delete(context.Context, uuid.UUID) error
}

type Authorizer interface {
	CanReadContactPoint(context.Context, string) (domain.Result, error)
	AuthorizeGlobal(context.Context, string, domain.GlobalPermission) (domain.Result, error)
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
		log:        log.With("component", "contact-points-service"),
	}
}

func (s *Service) List(ctx context.Context, actorID string) ([]domain.ContactPoint, error) {
	if err := s.authorize(ctx, actorID); err != nil {
		return nil, err
	}

	contactPoints, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list contact points: %w", err)
	}

	return contactPoints, nil
}

func (s *Service) Get(ctx context.Context, actorID string, id uuid.UUID) (domain.ContactPoint, error) {
	if id == uuid.Nil {
		return domain.ContactPoint{}, apperror.Default(apperror.CodeValidationFailed)
	}
	if err := s.authorize(ctx, actorID); err != nil {
		return domain.ContactPoint{}, err
	}

	contactPoint, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("get contact point: %w", err)
	}

	return contactPoint, nil
}

func (s *Service) Create(ctx context.Context, actorID string, input domain.ContactPointInput) (domain.ContactPoint, error) {
	if err := s.authorizeManage(ctx, actorID); err != nil {
		return domain.ContactPoint{}, err
	}

	contactPoint, err := s.store.Create(ctx, input)
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("create contact point: %w", err)
	}

	return contactPoint, nil
}

func (s *Service) Replace(ctx context.Context, actorID string, id uuid.UUID, input domain.ContactPointInput) (domain.ContactPoint, error) {
	if id == uuid.Nil {
		return domain.ContactPoint{}, apperror.Default(apperror.CodeValidationFailed)
	}

	if err := s.authorizeManage(ctx, actorID); err != nil {
		return domain.ContactPoint{}, err
	}

	contactPoint, err := s.store.Replace(ctx, id, input)
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("replace contact point: %w", err)
	}

	return contactPoint, nil
}

func (s *Service) Delete(ctx context.Context, actorID string, id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	if err := s.authorizeManage(ctx, actorID); err != nil {
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete contact point: %w", err)
	}

	return nil
}

func (s *Service) authorizeManage(ctx context.Context, actorID string) error {
	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, domain.GlobalPermissionContactPointManage)
	if err != nil {
		return fmt.Errorf("authorize contact point management: %w", err)
	}
	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}
	return nil
}

func (s *Service) authorize(ctx context.Context, actorID string) error {
	allowed, err := s.authorizer.CanReadContactPoint(ctx, actorID)
	if err != nil {
		return fmt.Errorf("authorize contact point read: %w", err)
	}
	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
	}

	return nil
}
