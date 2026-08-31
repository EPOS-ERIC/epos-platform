package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Discovery struct {
	service *discovery.Service
	log     *slog.Logger
}

func NewDiscovery(log *slog.Logger, service *discovery.Service) *Discovery {
	return &Discovery{
		service: service,
		log:     log.With("component", "discovery-handler"),
	}
}

func (h *Discovery) Search(w http.ResponseWriter, r *http.Request) {
	filter, err := toDomainDiscoveryFilter(r.URL.Query())
	if err != nil {
		WriteError(h.log, r.Context(), w, http.StatusBadRequest, "invalid_request", err.Error(), err)
		return
	}

	actorID := ""
	if actor, ok := AuthenticatedUserFromContext(r.Context()); ok {
		actorID = actor.ID
	}
	items, err := h.service.Search(r.Context(), actorID, filter)
	if err != nil {
		writeDiscoveryError(h.log, r, w, err)
		return
	}

	result := mapSlice(items, toAPIDiscoveryItem)
	writeJSON(h.log, w, http.StatusOK, result)
}

func writeDiscoveryError(log *slog.Logger, r *http.Request, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrBadRequest):
		WriteError(log, r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid discovery request", err)
	case errors.Is(err, domain.ErrUnauthorized):
		WriteError(log, r.Context(), w, http.StatusUnauthorized, "unauthorized", "authentication is required for non-published discovery results", nil)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(log, r.Context(), w, http.StatusForbidden, "forbidden", "discovery access denied", nil)
	case errors.Is(err, domain.ErrUpstreamTimeout):
		WriteError(log, r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "discovery request timed out", err)
	default:
		log.ErrorContext(r.Context(), "discovery request failed", "error", err)
		WriteError(log, r.Context(), w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
	}
}
