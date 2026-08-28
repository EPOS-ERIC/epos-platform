// Package categories provides PostgreSQL persistence for categories.
package categories

import (
	"context"
	"errors"
	"fmt"

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

func (r *Repo) List(ctx context.Context, groupID uuid.UUID) ([]domain.Category, error) {
	_, err := r.GetGroupByID(ctx, groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCategoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check group: %w", err)
	}

	rows, err := r.ListGroupCategories(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group categories: %w", err)
	}

	result := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		category, err := toDomainCategory(row)
		if err != nil {
			return nil, fmt.Errorf("construct category %s: %w", row.ID, err)
		}
		result = append(result, category)
	}

	return result, nil
}

func (r *Repo) Get(ctx context.Context, groupID, categoryID uuid.UUID) (domain.Category, error) {
	row, err := r.GetGroupCategory(ctx, sqlc.GetGroupCategoryParams{
		GroupID: groupID,
		ID:      categoryID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Category{}, domain.ErrCategoryNotFound
	}
	if err != nil {
		return domain.Category{}, fmt.Errorf("get group category: %w", err)
	}

	category, err := toDomainCategory(row)
	if err != nil {
		return domain.Category{}, fmt.Errorf("construct category %s: %w", row.ID, err)
	}

	return category, nil
}

func (r *Repo) Create(ctx context.Context, groupID uuid.UUID, input domain.CategoryInput) (domain.Category, error) {
	id := uuid.New()
	row, err := r.CreateGroupCategory(ctx, toCreateCategoryParams(id, groupID, input))
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.Code {
			case "23505":
				return domain.Category{}, domain.ErrCategoryConflict
			case "23503":
				return domain.Category{}, domain.ErrCategoryNotFound
			case "P0001":
				return domain.Category{}, domain.ErrInvalidCategory
			}
		}
		return domain.Category{}, fmt.Errorf("insert category: %w", err)
	}

	category, err := toDomainCategory(row)
	if err != nil {
		return domain.Category{}, fmt.Errorf("construct category %s: %w", row.ID, err)
	}

	return category, nil
}

func (r *Repo) Replace(ctx context.Context, groupID, categoryID uuid.UUID, input domain.CategoryInput) (domain.Category, error) {
	row, err := r.ReplaceGroupCategory(ctx, toReplaceCategoryParams(groupID, categoryID, input))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Category{}, domain.ErrCategoryNotFound
	}
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.Code {
			case "23505":
				return domain.Category{}, domain.ErrCategoryConflict
			case "23503":
				return domain.Category{}, domain.ErrCategoryNotFound
			case "23514", "P0001":
				return domain.Category{}, domain.ErrInvalidCategory
			}
		}
		return domain.Category{}, fmt.Errorf("update category: %w", err)
	}

	category, err := toDomainCategory(row)
	if err != nil {
		return domain.Category{}, fmt.Errorf("construct category %s: %w", row.ID, err)
	}

	return category, nil
}

func (r *Repo) Delete(ctx context.Context, groupID, categoryID uuid.UUID) error {
	rows, err := r.DeleteGroupCategory(ctx, sqlc.DeleteGroupCategoryParams{
		GroupID: groupID,
		ID:      categoryID,
	})
	if err != nil {
		return fmt.Errorf("delete category row: %w", err)
	}
	if rows == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}
