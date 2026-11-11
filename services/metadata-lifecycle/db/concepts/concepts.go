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
		value, err := domain.NewConcept(row.ID, row.Name)
		if err != nil {
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

	return domain.NewConcept(row.ID, row.Name)
}

func (r *Repo) CreateConcept(ctx context.Context, name string) (domain.Concept, error) {
	row, err := r.CreateConceptRecord(ctx, name)
	if err != nil {
		return domain.Concept{}, conceptError("create", err)
	}

	return domain.NewConcept(row.ID, row.Name)
}

func (r *Repo) ReplaceConcept(ctx context.Context, id uuid.UUID, name string) (domain.Concept, error) {
	row, err := r.ReplaceConceptRecord(ctx, sqlc.ReplaceConceptRecordParams{
		ID:   id,
		Name: name,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Concept{}, apperror.Default(apperror.CodeNotFound)
	}

	if err != nil {
		return domain.Concept{}, conceptError("replace", err)
	}

	return domain.NewConcept(row.ID, row.Name)
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
