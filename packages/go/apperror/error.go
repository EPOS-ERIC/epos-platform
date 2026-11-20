// Package apperror defines the shared, client-safe application error contract.
package apperror

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

type Code string

const (
	CodeValidationFailed    Code = "validation_failed"
	CodeUnauthenticated     Code = "unauthenticated"
	CodePermissionDenied    Code = "permission_denied"
	CodeNotFound            Code = "not_found"
	CodeConflict            Code = "conflict"
	CodeInvalidTransition   Code = "invalid_transition"
	CodeInternal            Code = "internal"
	CodeUpstreamTimeout     Code = "upstream_timeout"
	CodeUpstreamUnavailable Code = "upstream_unavailable"
	CodeUpstreamHTTPFailure Code = "upstream_http_failure"
	CodeUnsupportedMedia    Code = "unsupported_media_type"
	CodeRequestTooLarge     Code = "request_too_large"
)

var defaultDetails = map[Code]string{
	CodeValidationFailed:    "The request is invalid.",
	CodeUnauthenticated:     "Authentication is required.",
	CodePermissionDenied:    "You are not permitted to perform this operation.",
	CodeNotFound:            "The requested resource was not found.",
	CodeConflict:            "The operation conflicts with the current state.",
	CodeInvalidTransition:   "The operation is not allowed in the current state.",
	CodeInternal:            "An unexpected error occurred.",
	CodeUpstreamTimeout:     "The request timed out.",
	CodeUpstreamUnavailable: "A required service is unavailable.",
	CodeUpstreamHTTPFailure: "Upstream service returned an HTTP failure.",
	CodeUnsupportedMedia:    "The request media type is not supported.",
	CodeRequestTooLarge:     "The request body is too large.",
}

// AppError marks an error as safe to describe to a caller. Cause remains
// available to server-side error handling and is never part of the wire shape.
type AppError struct {
	Code   Code
	Detail string
	Target string
	cause  error
}

// New constructs a client-safe error. Empty details use the code's safe default.
func New(code Code, detail string) *AppError {
	if !Known(code) {
		code = CodeInternal
	}
	if detail == "" {
		detail = defaultDetails[code]
	}
	return &AppError{Code: code, Detail: detail}
}

// Default constructs a client-safe error with the code's standard detail.
func Default(code Code) *AppError {
	return New(code, "")
}

// Wrap constructs a client-safe error while retaining an internal cause.
func Wrap(code Code, detail string, cause error) *AppError {
	err := New(code, detail)
	err.cause = cause
	return err
}

// Field marks one validation failure for a client-facing request field.
func Field(target, detail string, cause error) *AppError {
	err := Wrap(CodeValidationFailed, detail, cause)
	err.Target = target
	return err
}

// PrefixTarget adds a parent field path without nesting AppErrors.
func PrefixTarget(err error, prefix string) error {
	var appErr *AppError
	if !errors.As(err, &appErr) || prefix == "" {
		return err
	}
	copy := *appErr
	if copy.Target == "" {
		copy.Target = prefix
	} else if strings.HasPrefix(copy.Target, "[") {
		copy.Target = prefix + copy.Target
	} else {
		copy.Target = prefix + "." + copy.Target
	}
	return &copy
}

// Known reports whether code belongs to the shared contract.
func Known(code Code) bool {
	_, ok := defaultDetails[code]
	return ok
}

// Unwrap returns the internal cause.
func (e *AppError) Unwrap() error {
	return e.cause
}

// IsCode reports whether err wraps an AppError with the given code.
func IsCode(err error, code Code) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Code == code
}

// Error returns internal cause text for server-side logging, otherwise safe detail.
func (e *AppError) Error() string {
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.Detail
}

// LogAttrs returns structured diagnostics without exposing the underlying cause.
func (e *AppError) LogAttrs() []slog.Attr {
	if e == nil {
		return nil
	}
	attrs := []slog.Attr{
		slog.String("error_code", string(e.Code)),
		slog.String("error_detail", e.Detail),
	}
	if e.Target != "" {
		attrs = append(attrs, slog.String("error_target", e.Target))
	}
	return attrs
}

// HTTPStatus returns the public HTTP status assigned to code.
func HTTPStatus(code Code) int {
	switch code {
	case CodeValidationFailed:
		return http.StatusBadRequest
	case CodeUnsupportedMedia:
		return http.StatusUnsupportedMediaType
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodePermissionDenied:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict, CodeInvalidTransition:
		return http.StatusConflict
	case CodeUpstreamTimeout:
		return http.StatusGatewayTimeout
	case CodeUpstreamUnavailable, CodeUpstreamHTTPFailure:
		return http.StatusServiceUnavailable
	case CodeRequestTooLarge:
		return http.StatusRequestEntityTooLarge
	default:
		return http.StatusInternalServerError
	}
}
