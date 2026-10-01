package resource

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// invalidCategoryIndex returns the first category ID not found in the group,
// including repeated IDs, or -1 when all references are valid.
func invalidCategoryIndex(categoryIDs, groupCategoryIDs []uuid.UUID) int {
	validIDs := make(map[uuid.UUID]struct{}, len(groupCategoryIDs))
	for _, id := range groupCategoryIDs {
		validIDs[id] = struct{}{}
	}
	seen := make(map[uuid.UUID]struct{}, len(categoryIDs))
	for i, id := range categoryIDs {
		if _, ok := validIDs[id]; !ok {
			return i
		}
		if _, ok := seen[id]; ok {
			return i
		}
		seen[id] = struct{}{}
	}
	return -1
}

// CategoriesBelongToGroup returns the first invalid category index before resource mutation.
func CategoriesBelongToGroup(ctx context.Context, q *sqlc.Queries, groupID uuid.UUID, categoryIDs []uuid.UUID) (int, error) {
	if len(categoryIDs) == 0 {
		return -1, nil
	}

	rows, err := q.ListLifecycleCategoryIDsByGroup(ctx, sqlc.ListLifecycleCategoryIDsByGroupParams{
		GroupID:     groupID,
		CategoryIds: categoryIDs,
	})
	if err != nil {
		return -1, fmt.Errorf("list category references: %w", err)
	}
	return invalidCategoryIndex(categoryIDs, rows), nil
}

// LifecycleError classifies PostgreSQL errors from resource lifecycle writes.
func LifecycleError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.Default(apperror.CodeConflict)
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case "23514":
			return fmt.Errorf("database validation: %w", apperror.Wrap(apperror.CodeValidationFailed, "", err))
		case "23001", "23505", "23P01", "P0001":
			return fmt.Errorf("database invariant %q: %w", pgErr.ConstraintName, apperror.Wrap(apperror.CodeConflict, "", err))
		default:
			if strings.HasPrefix(pgErr.Code, "23") {
				return fmt.Errorf("database integrity error %q: %w", pgErr.Code, apperror.Wrap(apperror.CodeInternal, "", err))
			}
		}
	}
	return err
}
