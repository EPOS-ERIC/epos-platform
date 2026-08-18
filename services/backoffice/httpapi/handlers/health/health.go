// Package health exposes service health HTTP handlers.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db"
)

var log = logging.Get("health-handler")

type healthResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type handler struct {
	db *db.PostgresDB
}

// New creates a database health handler.
func New(database *db.PostgresDB) *handler {
	return &handler{
		db: database,
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		log.Error("health check failed", "component", "database", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		if err := json.NewEncoder(w).Encode(healthResponse{Status: "degraded", Reason: "database unreachable"}); err != nil {
			log.Error("failed to encode health response", "status", http.StatusServiceUnavailable, "error", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok", Reason: ""}); err != nil {
		log.Error("failed to encode health response", "status", http.StatusOK, "error", err)
	}
}
