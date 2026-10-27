// Package middleware provides shared HTTP middleware for logging, recovery, and CORS.
package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// TODO: we don't need a dedicated package for this since only the gateway has an http server, we can move the middleware there
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
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

func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	log = log.With("component", "request")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: 200, size: 0}
			next.ServeHTTP(rw, r)
			duration := time.Since(startedAt)

			if shouldSkipHealthLog(r, rw.status) {
				return
			}

			log.InfoContext(r.Context(), "http request", "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration", duration.String())
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

func CORSAllowOrigin(allowOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			next.ServeHTTP(w, r)
		})
	}
}

// Recoverer handles panics and returns an internal server error response.
func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return RecovererWithResponse(log, func(w http.ResponseWriter, _ *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(`{"error": "Internal Server Error"}`))
		return err
	})
}

// RecovererWithResponse handles panics and writes an application-specific response.
func RecovererWithResponse(log *slog.Logger, writeResponse func(http.ResponseWriter, *http.Request) error) func(http.Handler) http.Handler {
	log = log.With("component", "recovery")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.ErrorContext(r.Context(), "panic recovered", "error", err)
					if err := writeResponse(w, r); err != nil {
						log.ErrorContext(r.Context(), "failed to write error response", "error", err)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
