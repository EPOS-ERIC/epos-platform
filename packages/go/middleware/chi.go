// Package middleware provides shared HTTP middleware for logging, recovery, and CORS.
package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
)

var log = logging.Get("http")

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

func Logger(next http.Handler) http.Handler {
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
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.ErrorContext(r.Context(), "panic recovered", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				if _, err := w.Write([]byte(`{"error": "Internal Server Error"}`)); err != nil {
					log.ErrorContext(r.Context(), "failed to write error response", "error", err)
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}
