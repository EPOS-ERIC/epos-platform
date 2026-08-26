package transport

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
)

var log = logging.Get("transport")

// Decode parses one JSON request body into target.
func Decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid JSON request")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid JSON request")
		return false
	}
	return true
}

// WriteJSON writes value as a JSON response.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Error("encode JSON response failed", "status", status, "error", err)
	}
}

// WriteProblem writes a problem details JSON response.
func WriteProblem(w http.ResponseWriter, status int, code, detail string) {
	WriteJSON(w, status, problem{Title: code, Status: status, Detail: detail})
}

type problem struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}
