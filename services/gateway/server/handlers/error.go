package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ogen-go/ogen/ogenerrors"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func NewErrorHandler(log *slog.Logger) ogenerrors.ErrorHandler {
	return func(ctx context.Context, w http.ResponseWriter, _ *http.Request, err error) {
		writeError(log, ctx, w, err)
	}
}

// RequestTimeout gives each request a deadline. Handlers and downstream calls
// must observe the request context for the deadline to stop their work.
func RequestTimeout(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeError(log *slog.Logger, ctx context.Context, w http.ResponseWriter, err error) {
	status, problem := problemResponse(ctx, err)
	reportHTTPError(log, ctx, err, status)
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(&problem); err != nil {
		log.ErrorContext(ctx, "failed to encode problem response", "status", status, "error", err)
	}
}

func reportHTTPError(log *slog.Logger, ctx context.Context, err error, status int) {
	if status < http.StatusInternalServerError {
		log.DebugContext(ctx, "gateway request failed", "status", status, "error", err)
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(otelcodes.Error, http.StatusText(status))
	log.ErrorContext(ctx, "gateway request failed", "status", status, "error", err)
}
