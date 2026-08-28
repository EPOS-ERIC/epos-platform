package categories

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

func toDomainCategoryInput(value categoryRequest) (domain.CategoryInput, error) {
	input, err := domain.NewCategoryInput(domain.CategoryInput{
		Name:             value.Name,
		ParentCategoryID: value.ParentCategoryID,
	})
	if err != nil {
		return domain.CategoryInput{}, fmt.Errorf("map category request: %w", err)
	}

	return input, nil
}

func toResponse(value domain.Category) categoryResponse {
	return categoryResponse{
		ID:               value.ID,
		GroupID:          value.GroupID,
		ParentCategoryID: value.ParentCategoryID,
		Name:             value.Name,
	}
}
