package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/transport"
)

func auditMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := logging.WithRequest(r.Context(), logging.Request{
				Method: r.Method,
				Path:   r.URL.Path,
				Query:  r.URL.RawQuery,
			})

			if actorID := strings.TrimSpace(r.Header.Get(transport.ActorHeader)); actorID != "" {
				ctx = logging.WithActor(ctx, actorID)
			}

			rw := &auditResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
				wroteHeader:    false,
			}
			next.ServeHTTP(rw, r.WithContext(ctx))

			switch rw.status {
			case http.StatusUnauthorized:
				log.WarnContext(ctx, "access denied", "event", "authentication_required", "status", rw.status)
			case http.StatusForbidden:
				log.WarnContext(ctx, "access denied", "event", "authorization_denied", "status", rw.status)
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
