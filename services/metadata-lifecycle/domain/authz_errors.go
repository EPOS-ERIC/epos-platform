package domain

import "errors"

var (
	ErrAuthorizationNotFound = errors.New("authorization resource not found")
	ErrInvalidAuthorization  = errors.New("invalid authorization")
)
