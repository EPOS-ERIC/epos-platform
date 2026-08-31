// Package health exposes service health HTTP handlers.
package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db"
)

type healthResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type Handler struct {
	db  *db.PostgresDB
	log *slog.Logger
}

// New creates a database health handler.
func New(log *slog.Logger, database *db.PostgresDB) *Handler {
	return &Handler{
		db:  database,
		log: log.With("component", "health-handler"),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		h.log.Error("health check failed", "component", "database", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		if err := json.NewEncoder(w).Encode(healthResponse{Status: "degraded", Reason: "database unreachable"}); err != nil {
			h.log.Error("failed to encode health response", "status", http.StatusServiceUnavailable, "error", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok", Reason: ""}); err != nil {
		h.log.Error("failed to encode health response", "status", http.StatusOK, "error", err)
	}
}
