package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/ogen-go/ogen/validate"
)

func NewErrorHandler(log *slog.Logger) ogenerrors.ErrorHandler {
	return func(ctx context.Context, w http.ResponseWriter, _ *http.Request, err error) {
		log.DebugContext(ctx, "error handler", "error", err)
		status, code, message := problemDetails(err)
		if code == "request_failed" {
			status = http.StatusBadRequest
			code = "invalid_request"
			message = "invalid request"
		}

		var contentTypeErr *validate.InvalidContentTypeError
		var maxBytesErr *http.MaxBytesError
		switch {
		case errors.As(err, &contentTypeErr):
			status = http.StatusUnsupportedMediaType
			code = "unsupported_media_type"
			message = "unsupported media type"
		case errors.As(err, &maxBytesErr):
			status = http.StatusRequestEntityTooLarge
			code = "request_too_large"
			message = fmt.Sprintf("request body exceeds %d bytes", maxBytesErr.Limit)
		}
		writeError(log, ctx, w, status, code, message)
	}
}

func NewTimeoutHandler(log *slog.Logger, next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := &timeoutResponseWriter{header: make(http.Header), body: bytes.Buffer{}, status: 0}
		finished := make(chan struct{})
		go func(parent context.Context) {
			ctx, cancel := context.WithTimeout(parent, timeout)
			defer cancel()
			next.ServeHTTP(response, r.WithContext(ctx))
			close(finished)
		}(r.Context())

		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-finished:
			response.copyTo(w)
		case <-timer.C:
			writeError(log, r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "request timed out")
		}
	})
}

type timeoutResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (w *timeoutResponseWriter) Header() http.Header {
	return w.header
}

func (w *timeoutResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *timeoutResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, _ := w.body.Write(body)
	return n, nil
}

func (w *timeoutResponseWriter) copyTo(dst http.ResponseWriter) {
	for key, values := range w.header {
		for _, value := range values {
			dst.Header().Add(key, value)
		}
	}
	if w.status != 0 {
		dst.WriteHeader(w.status)
	}
	_, _ = dst.Write(w.body.Bytes())
}

func writeError(log *slog.Logger, ctx context.Context, w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	problem := gatewayapi.Problem{
		Title:    code,
		Status:   status,
		Detail:   gatewayapi.NewOptString(message),
		Instance: gatewayapi.OptString{Value: "", Set: false},
	}
	if err := json.NewEncoder(w).Encode(&problem); err != nil {
		log.ErrorContext(ctx, "failed to encode problem response", "status", status, "error", err)
	}
}
