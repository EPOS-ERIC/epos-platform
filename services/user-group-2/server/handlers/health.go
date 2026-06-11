package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/db"
)

type Health struct {
	db *db.PostgresDB
}

func NewHealth(database *db.PostgresDB) *Health {
	return &Health{
		db: database,
	}
}

func (h *Health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
