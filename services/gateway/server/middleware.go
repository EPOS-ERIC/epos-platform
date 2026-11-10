package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/server/handlers"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	if err != nil {
		return size, fmt.Errorf("write response body: %w", err)
	}
	return size, nil
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	log = log.With("component", "request")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(rw, r)
			duration := time.Since(startedAt)

			if shouldSkipHealthLog(r, rw.status) {
				return
			}

			userAgent := []rune(r.UserAgent())
			if len(userAgent) > 256 {
				userAgent = userAgent[:256]
			}
			log.InfoContext(
				r.Context(),
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration", duration.String(),
				"user_agent", string(userAgent),
			)
		})
	}
}

func shouldSkipHealthLog(r *http.Request, status int) bool {
	if r.Method != http.MethodGet {
		return false
	}

	if r.URL.Path != "/health" && r.URL.Path != "/healthz" {
		return false
	}

	return status < http.StatusBadRequest
}

func corsMiddleware(allowOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			next.ServeHTTP(w, r)
		})
	}
}

func recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	log = log.With("component", "recovery")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() { //nolint:contextcheck // Request context is captured from r.
				if err := recover(); err != nil {
					log.ErrorContext(r.Context(), "panic recovered", "error", err)
					if err := handlers.WritePanicProblem(w, r); err != nil {
						log.ErrorContext(r.Context(), "failed to write error response", "error", err)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
