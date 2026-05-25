package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type healthPinger interface {
	PingContext(ctx context.Context) error
}

func NewHealthHandler(pinger healthPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if pinger == nil {
			writeHealthOK(w)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()

		if err := pinger.PingContext(ctx); err != nil {
			log.Error("health check failed", "component", "database", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			if err := json.NewEncoder(w).Encode(map[string]string{"status": "degraded", "reason": "database unreachable"}); err != nil {
				log.Error("failed to encode JSON response", "status", http.StatusServiceUnavailable, "error", err)
			}
			return
		}

		writeHealthOK(w)
	}
}

func writeHealthOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Error("failed to encode JSON response", "status", http.StatusOK, "error", err)
	}
}
