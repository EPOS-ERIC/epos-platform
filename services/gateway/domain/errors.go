// Package domain defines shared domain-level errors and contracts.
package domain

import (
	"errors"
	"fmt"
)

// TODO: we should use the domain errors only for non proxied endpoints? or not? define this well and clean it up

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

// ErrorMetadataFrom returns client-safe details attached to err or its cause.
func ErrorMetadataFrom(err error) (ErrorMetadata, bool) {
	var value *metadataError
	if !errors.As(err, &value) {
		return ErrorMetadata{}, false
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

// UpstreamError represents a failed request to a downstream service.
// Response bodies stay inside the gateway and are not part of the public API.
// TODO: why do we have these two error structs? can't we only use one?
type UpstreamError struct {
	StatusCode   int
	WrappedError error
}

func (e *UpstreamError) Error() string {
	if e.WrappedError != nil {
		return fmt.Sprintf("upstream request failed: %s", e.WrappedError.Error())
	}
	return fmt.Sprintf("upstream request failed: %d", e.StatusCode)
}
