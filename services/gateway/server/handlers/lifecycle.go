package handlers

import (
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/lifecycle"
	"github.com/go-chi/chi/v5"
)

type Lifecycle struct {
	service *lifecycle.Service
}

func NewLifecycle(service *lifecycle.Service) *Lifecycle {
	return &Lifecycle{service: service}
}

func (l *Lifecycle) DraftResource(w http.ResponseWriter, r *http.Request) {
	resourceTypeValue := chi.URLParam(r, "resourceType")
	resourceType, err := api.NewResourceTypeFromValue(resourceTypeValue)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("resource type '%s' not supported", resourceTypeValue), nil)
		return
	}

	id := chi.URLParam(r, "id")
	result, err := l.service.DraftResource(r.Context(), resourceType, id, "TODO: user", "TODO: group")
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "internal_server_error", "resource draft request failed", err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
