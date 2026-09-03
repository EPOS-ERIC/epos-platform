// Package domain defines shared domain-level errors and contracts.
package domain

import (
	"errors"
	"fmt"
)

// ErrorMetadata contains safe details intended for gateway clients.
type ErrorMetadata struct {
	Title  string
	Detail string
}

type metadataError struct {
	err      error
	metadata ErrorMetadata
}

func (e *metadataError) Error() string {
	return e.err.Error()
}

func (e *metadataError) Unwrap() error {
	return e.err
}

// WithErrorMetadata attaches client-safe details while preserving err's type.
func WithErrorMetadata(err error, title, detail string) error {
	if err == nil {
		return nil
	}
	if title == "" {
		return fmt.Errorf("error metadata title is required: %w", err)
	}
	return &metadataError{err: err, metadata: ErrorMetadata{Title: title, Detail: detail}}
}

// ErrorMetadataFrom returns client-safe details attached to err or its cause.
func ErrorMetadataFrom(err error) (ErrorMetadata, bool) {
	var value *metadataError
	if !errors.As(err, &value) {
		return ErrorMetadata{Title: "", Detail: ""}, false
	}
	return value.metadata, true
}

var (
	ErrNotImplemented  = errors.New("not implemented")
	ErrNotFound        = errors.New("not found")
	ErrBadRequest      = errors.New("bad request")
	ErrConflict        = errors.New("conflict")
	ErrUpstreamTimeout = errors.New("upstream timeout")
	ErrForbidden       = errors.New("forbidden")
	ErrUnauthorized    = errors.New("unauthorized")
)
