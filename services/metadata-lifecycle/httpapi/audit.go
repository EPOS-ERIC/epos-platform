package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func auditMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &auditResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
				wroteHeader:    false,
			}
			next.ServeHTTP(rw, r)

			span := trace.SpanFromContext(r.Context())

			if actorID := strings.TrimSpace(r.Header.Get(transport.ActorHeader)); actorID != "" {
				span.SetAttributes(
					// "enduser" is an otel-specific naming
					attribute.String("enduser.id", actorID),
				)
			}

			switch rw.status {
			case http.StatusUnauthorized:
				log.WarnContext(r.Context(), "access denied", "event", "authentication_required", "status", rw.status)
			case http.StatusForbidden:
				log.WarnContext(r.Context(), "access denied", "event", "authorization_denied", "status", rw.status)
			}
		})
	}
}

type auditResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *auditResponseWriter) WriteHeader(status int) {
	if rw.wroteHeader {
		return
	}

	rw.status = status
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *auditResponseWriter) Write(body []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	size, err := rw.ResponseWriter.Write(body)
	if err != nil {
		return size, fmt.Errorf("write response body: %w", err)
	}

	return size, nil
}
