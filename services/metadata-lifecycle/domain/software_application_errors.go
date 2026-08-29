package domain

import "errors"

var (
	ErrSoftwareApplicationForbidden = errors.New("software application operation forbidden")
	ErrSoftwareApplicationConflict  = errors.New("software application lifecycle conflict")
	ErrSoftwareApplicationNotFound  = errors.New("software application not found")
	ErrSoftwareApplicationInvalid   = errors.New("invalid software application")
	ErrSoftwareApplicationInternal  = errors.New("software application operation failed")
)
