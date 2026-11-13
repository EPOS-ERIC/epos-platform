// Package organizations provides public canonical organization reads.
package organizations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Store interface {
	List(context.Context) ([]domain.Organization, error)
	Get(context.Context, uuid.UUID) (domain.Organization, error)
	Create(context.Context, domain.OrganizationInput) (domain.Organization, error)
	Replace(context.Context, uuid.UUID, domain.OrganizationInput) (domain.Organization, error)
	Delete(context.Context, uuid.UUID) error
	Sync(context.Context, []domain.SyncedOrganization) error
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

// Sync applies a complete remote snapshot. The internal RPC is network restricted.
func (s *Service) Sync(ctx context.Context, values []domain.SyncedOrganization) error {
	seen := make(map[int64]bool, len(values))

	for i := range values {
		value := &values[i]
		if value.NID <= 0 || seen[value.NID] {
			return apperror.Field("organizations", "Organization NIDs must be positive and unique.", nil)
		}

		seen[value.NID] = true
	}

	for _, value := range values {
		parents := make(map[int64]bool, len(value.ParentNIDs))
		for _, parent := range value.ParentNIDs {
			if !seen[parent] || parent == value.NID || parents[parent] {
				return apperror.Field("organizations", "Parents must be unique, other organizations in the snapshot.", nil)
			}

			parents[parent] = true
		}
	}

	if err := s.store.Sync(ctx, values); err != nil {
		return fmt.Errorf("sync organizations: %w", err)
	}

	return nil
}

func (s *Service) Create(ctx context.Context, actorID domain.Actor, input domain.OrganizationInput) (domain.Organization, error) {
	allowed, err := s.authorizer.AuthorizeGlobal(ctx, string(actorID), domain.GlobalPermissionOrganizationManage)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("authorize organization creation: %w", err)
	}
	if !allowed.Allowed {
		return domain.Organization{}, apperror.Default(apperror.CodePermissionDenied)
	}

	organization, err := s.store.Create(ctx, input)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("create organization: %w", err)
	}

	return organization, nil
}

func (s *Service) Replace(ctx context.Context, actorID domain.Actor, id uuid.UUID, input domain.OrganizationInput) (domain.Organization, error) {
	if id == uuid.Nil {
		return domain.Organization{}, apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, string(actorID), domain.GlobalPermissionOrganizationManage)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("authorize organization replacement: %w", err)
	}
	if !allowed.Allowed {
		return domain.Organization{}, apperror.Default(apperror.CodePermissionDenied)
	}

	organization, err := s.store.Replace(ctx, id, input)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("replace organization: %w", err)
	}
	return organization, nil
}

func (s *Service) Delete(ctx context.Context, actorID domain.Actor, id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}

	allowed, err := s.authorizer.AuthorizeGlobal(ctx, string(actorID), domain.GlobalPermissionOrganizationManage)
	if err != nil {
		return fmt.Errorf("authorize organization deletion: %w", err)
	}
	if !allowed.Allowed {
		return apperror.Default(apperror.CodePermissionDenied)
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
		return domain.Organization{}, apperror.Default(apperror.CodeValidationFailed)
	}

	organization, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Organization{}, fmt.Errorf("get organization: %w", err)
	}

	return organization, nil
}
