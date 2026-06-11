package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/db"
)

type Health struct {
	db *db.PostgresDB
}

func NewHealth(database *db.PostgresDB) *Health {
	return &Health{db: database}
}

func (h *Health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		log.Error("health check failed", "component", "database", "error", err)
		writeHealthDegraded(w)
		return
	}

	writeHealthOK(w)
}
