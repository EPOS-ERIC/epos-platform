// Package organizations provides public canonical organization reads.
package organizations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	List(context.Context) ([]domain.Organization, error)
	Get(context.Context, uuid.UUID) (domain.Organization, error)
	Create(context.Context, domain.OrganizationInput) (domain.Organization, error)
	Replace(context.Context, uuid.UUID, domain.OrganizationInput) (domain.Organization, error)
	Delete(context.Context, uuid.UUID) error
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
	return &Service{
		store:      store,
		authorizer: authorizer,
		log:        log.With("component", "organizations-service"),
	}
}

func (s *Service) Create(ctx context.Context, actorID domain.Actor, input domain.OrganizationInput) (domain.Organization, error) {
	allowed, err := s.authorizer.AuthorizeGlobal(ctx, string(actorID), domain.GlobalPermissionOrganizationManage)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("authorize organization creation: %w", err)
	}
	if !allowed.Allowed {
		return domain.Organization{}, domain.ErrOrganizationForbidden
	}

	organization, err := s.store.Create(ctx, input)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("create organization: %w", err)
	}

	return organization, nil
}

func (s *Service) Replace(ctx context.Context, actorID domain.Actor, id uuid.UUID, input domain.OrganizationInput) (domain.Organization, error) {
	if id == uuid.Nil {
		return domain.Organization{}, domain.ErrInvalidOrganization
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, string(actorID), domain.GlobalPermissionOrganizationManage)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("authorize organization replacement: %w", err)
	}
	if !allowed.Allowed {
		return domain.Organization{}, domain.ErrOrganizationForbidden
	}

	organization, err := s.store.Replace(ctx, id, input)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("replace organization: %w", err)
	}
	return organization, nil
}

func (s *Service) Delete(ctx context.Context, actorID domain.Actor, id uuid.UUID) error {
	if id == uuid.Nil {
		return domain.ErrInvalidOrganization
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, string(actorID), domain.GlobalPermissionOrganizationManage)
	if err != nil {
		return fmt.Errorf("authorize organization deletion: %w", err)
	}
	if !allowed.Allowed {
		return domain.ErrOrganizationForbidden
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
		return domain.Organization{}, domain.ErrInvalidOrganization
	}

	organization, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("get organization: %w", err)
	}

	return organization, nil
}
