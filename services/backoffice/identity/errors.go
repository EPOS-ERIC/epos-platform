package identity

import "errors"

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrInvalid    = errors.New("invalid")
	ErrRuleNotHit = errors.New("rule not applicable")
)
