package domain

import "errors"

var (
	ErrInvalidOrganization   = errors.New("invalid organization")
	ErrOrganizationNotFound  = errors.New("organization not found")
	ErrOrganizationForbidden = errors.New("organization operation forbidden")
	ErrOrganizationConflict  = errors.New("organization conflict")
)
