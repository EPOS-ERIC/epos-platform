// Package metadataimport coordinates authorized, atomic metadata imports.
package metadataimport

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

type Authorizer interface {
	AuthorizeAdmin(context.Context, string) (domain.Result, error)
}

type Repository interface {
	Import(context.Context, string, domain.ImportBatch) (domain.ImportResult, error)
}

type Service struct {
	authorizer Authorizer
	repository Repository
}

// New creates an authorized metadata import service.
func New(authorizer Authorizer, repository Repository) *Service {
	return &Service{
		authorizer: authorizer,
		repository: repository,
	}
}

// Import authorizes and persists a validated metadata batch.
func (s *Service) Import(ctx context.Context, actor domain.Actor, batch domain.ImportBatch) (domain.ImportResult, error) {
	decision, err := s.authorizer.AuthorizeAdmin(ctx, string(actor))
	if err != nil {
		return domain.ImportResult{}, operationError("authorize import", err)
	}
	if !decision.Allowed {
		return domain.ImportResult{}, fmt.Errorf("%w: %s", domain.ErrImportForbidden, decision.Reason)
	}

	result, err := s.repository.Import(ctx, string(actor), batch)
	return result, operationError("persist import", err)
}

func operationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrImportForbidden) || errors.Is(err, domain.ErrImportConflict) || errors.Is(err, domain.ErrInvalidImport) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w: %w", operation, domain.ErrImportInternal, err)
}
