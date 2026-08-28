package categories

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toCreateCategoryParams(id, groupID uuid.UUID, input domain.CategoryInput) sqlc.CreateGroupCategoryParams {
	return sqlc.CreateGroupCategoryParams{
		ID:               id,
		GroupID:          groupID,
		ParentCategoryID: input.ParentCategoryID,
		Name:             input.Name,
	}
}

func toReplaceCategoryParams(groupID, categoryID uuid.UUID, input domain.CategoryInput) sqlc.ReplaceGroupCategoryParams {
	return sqlc.ReplaceGroupCategoryParams{
		GroupID:          groupID,
		ID:               categoryID,
		Name:             input.Name,
		ParentCategoryID: input.ParentCategoryID,
	}
}

func toDomainCategory(row sqlc.Category) (domain.Category, error) {
	category, err := domain.NewCategory(row.ID, row.GroupID, domain.CategoryInput{
		Name:             row.Name,
		ParentCategoryID: row.ParentCategoryID,
	})
	if err != nil {
		return domain.Category{}, fmt.Errorf("bad category: %w", err)
	}

	return category, nil
}
