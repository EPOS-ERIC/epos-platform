package transport

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

// Decode parses one JSON request body into target.
func Decode(log *slog.Logger, w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		log.ErrorContext(r.Context(), "decode JSON request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid JSON request")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		log.ErrorContext(r.Context(), "decode JSON request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid JSON request")
		return false
	}
	return true
}

// WriteJSON writes value as a JSON response.
func WriteJSON(log *slog.Logger, w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Error("encode JSON response failed", "status", status, "error", err)
	}
}

// WriteProblem writes a problem details JSON response.
func WriteProblem(log *slog.Logger, w http.ResponseWriter, status int, code, detail string) {
	WriteJSON(log, w, status, problem{Title: code, Status: status, Detail: detail})
}

type problem struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}
