package handlers

import (
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Discovery struct {
	service *discovery.Service
}

func NewDiscovery(service *discovery.Service) *Discovery {
	return &Discovery{service: service}
}

func (h *Discovery) Search(w http.ResponseWriter, r *http.Request) {
	filter, err := toDomainDiscoveryFilter(r.URL.Query())
	if err != nil {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", err.Error(), err)
		return
	}

	actorID := ""
	if actor, ok := AuthenticatedUserFromContext(r.Context()); ok {
		actorID = actor.ID
	}
	items, err := h.service.Search(r.Context(), actorID, filter)
	if err != nil {
		writeDiscoveryError(r, w, err)
		return
	}

	result := mapSlice(items, toAPIDiscoveryItem)
	writeJSON(w, http.StatusOK, result)
}

func writeDiscoveryError(r *http.Request, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrBadRequest):
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid discovery request", err)
	case errors.Is(err, domain.ErrUnauthorized):
		WriteError(r.Context(), w, http.StatusUnauthorized, "unauthorized", "authentication is required for non-published discovery results", nil)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "discovery access denied", nil)
	case errors.Is(err, domain.ErrUpstreamTimeout):
		WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "discovery request timed out", err)
	default:
		log.ErrorContext(r.Context(), "discovery request failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
	}
}
