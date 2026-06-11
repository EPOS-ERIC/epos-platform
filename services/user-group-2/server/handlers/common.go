package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("handlers")

type messageResponse struct {
	Message string `json:"message"`
}

type healthResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func writeMessage(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(messageResponse{Message: message}); err != nil {
		log.Error("failed to encode message response", "status", status, "error", err)
	}
}

func writeHealthOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok", Reason: ""}); err != nil {
		log.Error("failed to encode health response", "status", http.StatusOK, "error", err)
	}
}

func writeHealthDegraded(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "degraded", Reason: "database unreachable"}); err != nil {
		log.Error("failed to encode health response", "status", http.StatusServiceUnavailable, "error", err)
	}
}

func parseUUIDParam(r *http.Request, name string) (uuid.UUID, error) {
	raw := strings.TrimSpace(chi.URLParam(r, name))
	if raw == "" {
		return uuid.Nil, fmt.Errorf("%s is required", name)
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", name)
	}

	return id, nil
}

func nullString(v string) sql.NullString {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return sql.NullString{String: "", Valid: false}
	}

	return sql.NullString{String: trimmed, Valid: true}
}
