// Package middleware provides HTTP middleware functions for logging, error recovery, and content type handling.
package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
)

var log = logging.Get("http")

// TODO: see if this package makes sense as a shared library for different services

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
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			generatedID, err := logging.NewRequestID()
			if err != nil {
				log.ErrorContext(r.Context(), "generate request id failed", "error", err)
			} else {
				requestID = generatedID
			}
		}

		ctx := logging.WithRequest(r.Context(), logging.Request{
			RequestID: requestID,
			Method:    r.Method,
			Path:      r.URL.Path,
			Query:     r.URL.RawQuery,
		})
		r = r.WithContext(ctx)
		if requestID != "" {
			w.Header().Set("X-Request-Id", requestID)
		}

		startedAt := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		duration := time.Since(startedAt)

		log.InfoContext(r.Context(), "http request", "status", rw.status, "duration", duration)
	})
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
