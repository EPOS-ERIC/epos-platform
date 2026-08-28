package domain

import "errors"

var (
	ErrInvalidCategory   = errors.New("invalid category")
	ErrCategoryNotFound  = errors.New("category not found")
	ErrCategoryForbidden = errors.New("category operation forbidden")
	ErrCategoryConflict  = errors.New("category conflict")
)
