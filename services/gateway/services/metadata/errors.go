package metadatalifecycle

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type UpstreamError struct {
	Operation  string
	StatusCode int
	Body       string
	Err        error
}

func (e *UpstreamError) Error() string {
	statusLabel := http.StatusText(e.StatusCode)
	if statusLabel == "" {
		statusLabel = "upstream error"
	}

	if e.Body == "" {
		return fmt.Sprintf("%s: upstream returned %d %s", e.Operation, e.StatusCode, statusLabel)
	}

	return fmt.Sprintf("%s: upstream returned %d %s: %s", e.Operation, e.StatusCode, statusLabel, e.Body)
}

func (e *UpstreamError) Unwrap() error {
	return e.Err
}

type TimeoutError struct {
	Operation string
	Err       error
}

func (e *TimeoutError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: request timed out or canceled", e.Operation)
	}

	return fmt.Sprintf("%s: request timed out or canceled: %v", e.Operation, e.Err)
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}

type TransportError struct {
	Operation string
	Err       error
}

func (e *TransportError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: transport failure", e.Operation)
	}

	return fmt.Sprintf("%s: transport failure: %v", e.Operation, e.Err)
}

func (e *TransportError) Unwrap() error {
	return e.Err
}

func wrapRequestError(operation string, response *http.Response, err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &TimeoutError{Operation: operation, Err: err}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &TimeoutError{Operation: operation, Err: err}
	}

	if response != nil {
		return &UpstreamError{
			Operation:  operation,
			StatusCode: response.StatusCode,
			Body:       extractResponseBody(err),
			Err:        err,
		}
	}

	return &TransportError{Operation: operation, Err: err}
}

func extractResponseBody(err error) string {
	type errorWithBody interface {
		Body() []byte
	}

	var bodyError errorWithBody
	if !errors.As(err, &bodyError) {
		return ""
	}

	return strings.TrimSpace(string(bodyError.Body()))
}
