package domain

import "errors"

var (
	ErrSoftwareSourceCodeForbidden = errors.New("software source code operation forbidden")
	ErrSoftwareSourceCodeConflict  = errors.New("software source code lifecycle conflict")
	ErrSoftwareSourceCodeNotFound  = errors.New("software source code not found")
	ErrSoftwareSourceCodeInvalid   = errors.New("invalid software source code")
	ErrSoftwareSourceCodeInternal  = errors.New("software source code operation failed")
)
