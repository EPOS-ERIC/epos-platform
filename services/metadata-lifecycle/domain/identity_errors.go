package domain

import "errors"

var (
	ErrInvalidUser   = errors.New("invalid user")
	ErrUserNotFound  = errors.New("user not found")
	ErrUserForbidden = errors.New("user operation forbidden")
	ErrUserConflict  = errors.New("user operation conflict")
)
