// Package middleware provides HTTP middleware functions for logging, error recovery, and content type handling.
package middleware

import (
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
	return size, err
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		duration := time.Since(startedAt)

		if r.URL.RawQuery == "" {
			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration", duration,
			)
		} else {
			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"status", rw.status,
				"duration", duration,
			)
		}
	})
}

// Recoverer handles panics and returns an internal server error response.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered", "error", err, "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				if _, err := w.Write([]byte(`{"error": "Internal Server Error"}`)); err != nil {
					log.Error("failed to write error response", "error", err)
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}
