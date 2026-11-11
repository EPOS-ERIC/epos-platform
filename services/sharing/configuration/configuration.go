package configuration

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/domain"
	"github.com/google/uuid"
)

type ListFilter struct {
	Scope   domain.ListScope
	OwnerID domain.Actor
	Caller  domain.Caller
}

type Repository interface {
	Create(context.Context, domain.Actor, domain.ConfigurationInput) (domain.Configuration, error)
	GetByID(context.Context, uuid.UUID) (domain.Configuration, error)
	List(context.Context, ListFilter) ([]domain.ConfigurationSummary, error)
	Update(context.Context, uuid.UUID, domain.Caller, domain.ConfigurationInput) (domain.Configuration, error)
	Delete(context.Context, uuid.UUID, domain.Caller) error
	SetOfficialStatus(context.Context, uuid.UUID, domain.Caller, bool) (domain.Configuration, error)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func requireAuthenticated(caller domain.Caller) error {
	if caller.Actor == "" {
		return apperror.Default(apperror.CodeUnauthenticated)
	}
	return nil
}

func (s *Service) Create(ctx context.Context, caller domain.Caller, input domain.ConfigurationInput) (domain.Configuration, error) {
	if err := requireAuthenticated(caller); err != nil {
		return domain.Configuration{}, err
	}
	return s.repo.Create(ctx, caller.Actor, input)
}

func (s *Service) Get(ctx context.Context, caller domain.Caller, id uuid.UUID) (domain.Configuration, error) {
	record, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Configuration{}, err
	}
	if !canRead(caller, record) {
		return domain.Configuration{}, apperror.Default(apperror.CodeNotFound)
	}
	return record, nil
}

func (s *Service) List(ctx context.Context, caller domain.Caller, filter ListFilter) ([]domain.ConfigurationSummary, error) {
	if _, err := domain.NewListScope(string(filter.Scope)); err != nil {
		return nil, err
	}
	if filter.Scope == domain.ListScopeOwner {
		if filter.OwnerID == "" {
			return nil, apperror.Field("owner_id", "Owner ID is required for this list scope.", nil)
		}
	} else if filter.OwnerID != "" {
		return nil, apperror.Field("owner_id", "Owner ID is only valid with the owner list scope.", nil)
	}

	switch filter.Scope {
	case domain.ListScopeOwn:
		if err := requireAuthenticated(caller); err != nil {
			return nil, err
		}
	case domain.ListScopeOwner, domain.ListScopeAll:
		if err := requireAuthenticated(caller); err != nil {
			return nil, err
		}
		if !caller.CanManage {
			return nil, apperror.Default(apperror.CodePermissionDenied)
		}
	case domain.ListScopeOfficial:
	default:
	}

	filter.Caller = caller

	return s.repo.List(ctx, filter)
}

func (s *Service) Update(ctx context.Context, caller domain.Caller, id uuid.UUID, input domain.ConfigurationInput) (domain.Configuration, error) {
	if err := requireAuthenticated(caller); err != nil {
		return domain.Configuration{}, err
	}
	record, err := s.repo.Update(ctx, id, caller, input)
	if apperror.IsCode(err, apperror.CodeNotFound) {
		return domain.Configuration{}, s.mutationDenied(ctx, id, caller, input.IsPublic)
	}
	return record, err
}

func (s *Service) Delete(ctx context.Context, caller domain.Caller, id uuid.UUID) error {
	if err := requireAuthenticated(caller); err != nil {
		return err
	}
	err := s.repo.Delete(ctx, id, caller)
	if apperror.IsCode(err, apperror.CodeNotFound) {
		return s.mutationDenied(ctx, id, caller, nil)
	}
	return err
}

func (s *Service) SetOfficialStatus(ctx context.Context, caller domain.Caller, id uuid.UUID, isOfficial bool) (domain.Configuration, error) {
	if err := requireAuthenticated(caller); err != nil {
		return domain.Configuration{}, err
	}
	if !caller.CanManage {
		return domain.Configuration{}, apperror.Default(apperror.CodePermissionDenied)
	}
	return s.repo.SetOfficialStatus(ctx, id, caller, isOfficial)
}

func canRead(caller domain.Caller, record domain.Configuration) bool {
	return record.IsPublic || (caller.Actor != "" && (caller.Actor == record.OwnerID || caller.CanManage))
}

func canWrite(caller domain.Caller, record domain.Configuration) bool {
	return caller.CanManage || (!record.IsOfficial && caller.Actor != "" && caller.Actor == record.OwnerID)
}

// The mutation itself enforces authorization. This read only classifies failure.
func (s *Service) mutationDenied(ctx context.Context, id uuid.UUID, caller domain.Caller, isPublic *bool) error {
	record, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if record.IsOfficial && isPublic != nil && !*isPublic {
		return apperror.Field("is_public", "Official configurations must remain public.", nil)
	}
	if !canRead(caller, record) || canWrite(caller, record) {
		return apperror.Default(apperror.CodeNotFound)
	}
	return apperror.Default(apperror.CodePermissionDenied)
}
