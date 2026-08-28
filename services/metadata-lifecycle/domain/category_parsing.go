package domain

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrInvalidCategory = errors.New("invalid category")

// NewCategoryInput validates and normalizes editable category fields.
func NewCategoryInput(value CategoryInput) (CategoryInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return CategoryInput{}, ErrInvalidCategory
	}
	if value.ParentCategoryID != nil && *value.ParentCategoryID == uuid.Nil {
		return CategoryInput{}, ErrInvalidCategory
	}
	return value, nil
}

// NewCategory creates a complete canonical category from its identity and input.
func NewCategory(id, groupID uuid.UUID, value CategoryInput) (Category, error) {
	if id == uuid.Nil || groupID == uuid.Nil {
		return Category{}, ErrInvalidCategory
	}
	value, err := NewCategoryInput(value)
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
