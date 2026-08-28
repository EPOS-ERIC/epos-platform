package domain

import "errors"

var (
	ErrInvalidGroup   = errors.New("invalid group")
	ErrGroupNotFound  = errors.New("group not found")
	ErrGroupForbidden = errors.New("action forbidden on group")
	ErrGroupConflict  = errors.New("group operation conflict")
)
