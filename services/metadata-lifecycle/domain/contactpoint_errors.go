package domain

import "errors"

var (
	ErrInvalidContactPoint   = errors.New("invalid contact point")
	ErrContactPointNotFound  = errors.New("contact point not found")
	ErrContactPointForbidden = errors.New("contact point operation forbidden")
	ErrContactPointConflict  = errors.New("contact point conflict")
)
