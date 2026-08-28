package domain

import "errors"

var (
	ErrInvalidAccess   = errors.New("invalid group access")
	ErrAccessNotFound  = errors.New("group access resource not found")
	ErrAccessForbidden = errors.New("group access operation forbidden")
	ErrAccessConflict  = errors.New("group access operation conflict")
)
