package resource

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/google/uuid"
)

// CategoriesBelongToGroup verifies category references before resource mutation.
func CategoriesBelongToGroup(ctx context.Context, q *sqlc.Queries, groupID uuid.UUID, categoryIDs []uuid.UUID) (bool, error) {
	if len(categoryIDs) == 0 {
		return true, nil
	}

	rows, err := q.ListLifecycleCategoryIDsByGroup(ctx, sqlc.ListLifecycleCategoryIDsByGroupParams{
		GroupID:     groupID,
		CategoryIds: categoryIDs,
	})
	if err != nil {
		return false, fmt.Errorf("list category references: %w", err)
	}
	return len(rows) == len(categoryIDs), nil
}
