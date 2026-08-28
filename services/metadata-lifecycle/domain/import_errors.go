package domain

import "errors"

var (
	ErrInvalidImport   = errors.New("invalid metadata import")
	ErrImportForbidden = errors.New("permission denied")
	ErrImportConflict  = errors.New("import conflict")
	ErrImportInternal  = errors.New("metadata import failed")
)
