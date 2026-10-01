package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// NewCategoryInput validates and normalizes editable category fields.
func NewCategoryInput(value CategoryInput) (CategoryInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return CategoryInput{}, invalidField("name", "Name must not be blank.", nil)
	}
	if value.ParentCategoryID != nil && *value.ParentCategoryID == uuid.Nil {
		return CategoryInput{}, invalidField("parent_category_id", "Parent category ID must be a valid UUID.", nil)
	}
	return value, nil
}

// NewCategory creates a complete canonical category from its identity and input.
func NewCategory(id, groupID uuid.UUID, value CategoryInput) (result Category, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()

	if id == uuid.Nil || groupID == uuid.Nil {
		return Category{}, apperror.Default(apperror.CodeValidationFailed)
	}
	value, err = NewCategoryInput(value)
	if err != nil {
		return Category{}, err
	}
	return Category{
		ID:               id,
		GroupID:          groupID,
		ParentCategoryID: value.ParentCategoryID,
		Name:             value.Name,
	}, nil
}
