// Package handlers holds HTTP handlers for user-group.
package handlers

import (
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("failed to encode response", "status", status, "error", err)
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
