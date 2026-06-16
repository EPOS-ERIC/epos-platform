// Package domain defines shared domain-level errors and contracts.
package domain

import "errors"

var ErrNotImplemented = errors.New("not implemented")

var ErrNotFound = errors.New("not found")

var ErrBadRequest = errors.New("bad request")

var ErrUpstreamTimeout = errors.New("upstream timeout")

var ErrForbidden = errors.New("forbidden")
