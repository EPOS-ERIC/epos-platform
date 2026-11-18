// Package concepts provides PostgreSQL persistence for concepts.
package concepts

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	*sqlc.Queries
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{Queries: sqlc.New(db)}
}

func (r *Repo) ListConcepts(ctx context.Context) ([]domain.Concept, error) {
	rows, err := r.ListConceptRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("list concepts: %w", err)
	}

	result := make([]domain.Concept, 0, len(rows))
	for _, row := range rows {
		value := domain.Concept{
			ID:   row.ID,
			Name: row.Name,
		}
		if err := value.Normalize(); err != nil {
			return nil, fmt.Errorf("construct concept: %w", err)
		}

		result = append(result, value)
	}

	return result, nil
}

func (r *Repo) GetConcept(ctx context.Context, id uuid.UUID) (domain.Concept, error) {
	row, err := r.GetConceptRecord(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Concept{}, apperror.Default(apperror.CodeNotFound)
	}

	if err != nil {
		return domain.Concept{}, fmt.Errorf("get concept: %w", err)
	}

	value := domain.Concept{
		ID:   row.ID,
		Name: row.Name,
	}
	if err := value.Normalize(); err != nil {
		return domain.Concept{}, err
	}
	return value, nil
}

func (r *Repo) CreateConcept(ctx context.Context, input domain.ConceptInput) (domain.Concept, error) {
	row, err := r.CreateConceptRecord(ctx, input.Name)
	if err != nil {
		return domain.Concept{}, conceptError("create", err)
	}

	value := domain.Concept{
		ID:   row.ID,
		Name: row.Name,
	}
	if err := value.Normalize(); err != nil {
		return domain.Concept{}, err
	}
	return value, nil
}

func (r *Repo) ReplaceConcept(ctx context.Context, id uuid.UUID, input domain.ConceptInput) (domain.Concept, error) {
	row, err := r.ReplaceConceptRecord(ctx, sqlc.ReplaceConceptRecordParams{
		ID:   id,
		Name: input.Name,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Concept{}, apperror.Default(apperror.CodeNotFound)
	}

	if err != nil {
		return domain.Concept{}, conceptError("replace", err)
	}

	value := domain.Concept{
		ID:   row.ID,
		Name: row.Name,
	}
	if err := value.Normalize(); err != nil {
		return domain.Concept{}, err
	}
	return value, nil
}

func (r *Repo) DeleteConcept(ctx context.Context, id uuid.UUID) error {
	count, err := r.DeleteConceptRecord(ctx, id)
	if err != nil {
		return conceptError("delete", err)
	}

	if count == 0 {
		return apperror.Default(apperror.CodeNotFound)
	}

	return nil
}

func conceptError(operation string, err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if ok && (pgErr.Code == "23505" || pgErr.Code == "23503" || pgErr.Code == "23001") {
		return apperror.Wrap(apperror.CodeConflict, "", fmt.Errorf("%s concept: %w", operation, err))
	}

	return fmt.Errorf("%s concept: %w", operation, err)
}
