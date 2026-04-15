package api

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("api")

type Handlers struct{}

func NewHandlers() *Handlers {
	return &Handlers{}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.healthCheck)
}

func (h *Handlers) healthCheck(w http.ResponseWriter, r *http.Request) {
	log.Debug("checking health")

	// TODO: what to do here? we don't depend on anything except sub services but none is really required

	jsonResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
}
