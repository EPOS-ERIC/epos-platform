package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// Category is a complete canonical category returned by the domain.
type Category struct {
	ID               uuid.UUID
	GroupID          uuid.UUID
	ParentCategoryID *uuid.UUID
	Name             string
}

// CategoryInput contains fields editable through category CRUD.
type CategoryInput struct {
	Name             string
	ParentCategoryID *uuid.UUID
}

// Normalize validates and normalizes editable category fields.
func (value *CategoryInput) Normalize() error {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return apperror.Field("name", "Name must not be blank.", nil)
	}
	if value.ParentCategoryID != nil && *value.ParentCategoryID == uuid.Nil {
		return apperror.Field("parent_category_id", "Parent category ID must be a valid UUID.", nil)
	}
	return nil
}

// Normalize validates a persisted category.
func (value *Category) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()

	if value.ID == uuid.Nil || value.GroupID == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}
	input := CategoryInput{Name: value.Name, ParentCategoryID: value.ParentCategoryID}
	if err := input.Normalize(); err != nil {
		return err
	}
	value.Name = input.Name
	return nil
}
