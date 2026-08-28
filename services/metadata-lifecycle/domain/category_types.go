package domain

import "github.com/google/uuid"

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
