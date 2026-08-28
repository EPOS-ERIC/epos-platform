// Package organizations provides public canonical organization reads.
package organizations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/authz"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

var (
	ErrInvalid   = errors.New("invalid organization")
	ErrNotFound  = errors.New("organization not found")
	ErrForbidden = errors.New("organization operation forbidden")
	ErrConflict  = errors.New("organization conflict")
)

type Store interface {
	List(context.Context) ([]domain.Organization, error)
	Get(context.Context, uuid.UUID) (domain.Organization, error)
	Create(context.Context, domain.OrganizationInput) (domain.Organization, error)
	Replace(context.Context, uuid.UUID, domain.OrganizationInput) (domain.Organization, error)
	Delete(context.Context, uuid.UUID) error
}

type Authorizer interface {
	AuthorizeGlobal(context.Context, string, string) (authz.Result, error)
}

type Service struct {
	store      Store
	authorizer Authorizer
}

func New(store Store, authorizer Authorizer) *Service {
	return &Service{store: store, authorizer: authorizer}
}

func (s *Service) Create(ctx context.Context, actorID string, input domain.OrganizationInput) (domain.Organization, error) {
	// TODO: maybe we could create a custom type alias for the actor? so that we can skip having to trim and check it every time (parse, don't validate)
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return domain.Organization{}, ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, "organization.manage")
	if err != nil {
		return domain.Organization{}, fmt.Errorf("authorize organization creation: %w", err)
	}
	if !allowed.Allowed {
		return domain.Organization{}, ErrForbidden
	}

	organization, err := s.store.Create(ctx, input)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("create organization: %w", err)
	}

	return organization, nil
}

func (s *Service) Replace(ctx context.Context, actorID string, id uuid.UUID, input domain.OrganizationInput) (domain.Organization, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || id == uuid.Nil {
		return domain.Organization{}, ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, "organization.manage")
	if err != nil {
		return domain.Organization{}, fmt.Errorf("authorize organization replacement: %w", err)
	}
	if !allowed.Allowed {
		return domain.Organization{}, ErrForbidden
	}

	organization, err := s.store.Replace(ctx, id, input)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("replace organization: %w", err)
	}
	return organization, nil
}

func (s *Service) Delete(ctx context.Context, actorID string, id uuid.UUID) error {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" || id == uuid.Nil {
		return ErrInvalid
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actorID, "organization.manage")
	if err != nil {
		return fmt.Errorf("authorize organization deletion: %w", err)
	}
	if !allowed.Allowed {
		return ErrForbidden
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}

	return nil
}

func (s *Service) List(ctx context.Context) ([]domain.Organization, error) {
	organizations, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return organizations, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	if id == uuid.Nil {
		return domain.Organization{}, ErrInvalid
	}

	organization, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("get organization: %w", err)
	}

	return organization, nil
}
