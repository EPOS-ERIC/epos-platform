// Package metadataimport coordinates authorized metadata imports.
package metadataimport

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
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
	log        *slog.Logger
}

// New creates an authorized metadata import service.
func New(log *slog.Logger, repository Repository, authorizer Authorizer) *Service {
	return &Service{
		authorizer: authorizer,
		repository: repository,
		log:        log.With("component", "import-service"),
	}
}

// Import authorizes and persists a validated metadata batch.
func (s *Service) Import(ctx context.Context, actor domain.Actor, batch domain.ImportBatch) (domain.ImportResult, error) {
	decision, err := s.authorizer.AuthorizeAdmin(ctx, string(actor))
	if err != nil {
		return domain.ImportResult{}, operationError("authorize import", err)
	}
	if !decision.Allowed {
		return domain.ImportResult{}, fmt.Errorf("%w: %s", apperror.Default(apperror.CodePermissionDenied), decision.Reason)
	}

	result, err := s.repository.Import(ctx, string(actor), batch)
	return result, operationError("persist import", err)
}

func operationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if apperror.IsCode(err, apperror.CodePermissionDenied) || apperror.IsCode(err, apperror.CodeConflict) || apperror.IsCode(err, apperror.CodeValidationFailed) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w: %w", operation, apperror.Default(apperror.CodeInternal), err)
}
