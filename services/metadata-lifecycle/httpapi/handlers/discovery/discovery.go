// Package discovery exposes resource discovery HTTP handlers.
package discovery

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
)

type Service interface {
	Search(context.Context, *domain.Actor, domain.DiscoveryFilter) ([]domain.DiscoveryItem, error)
}

type Handler struct {
	service Service
	log     *slog.Logger
}

func New(log *slog.Logger, service Service) *Handler {
	return &Handler{
		service: service,
		log:     log.With("component", "discovery-handler"),
	}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	actor, ok := transport.OptionalActor(w, r)
	if !ok {
		return
	}

	filter, err := filterFromRequest(r)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	h.log.DebugContext(r.Context(), "discovery search", "filter", filter)
	items, err := h.service.Search(r.Context(), actor, filter)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidDiscovery):
			h.log.DebugContext(r.Context(), "discovery search invalid", "error", err)
			transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", err.Error())
		case errors.Is(err, domain.ErrDiscoveryDenied):
			h.log.DebugContext(r.Context(), "discovery search denied", "error", err)
			transport.WriteProblem(h.log, w, http.StatusUnauthorized, "unauthorized", "authentication is required for non-published discovery results")
		default:
			h.log.ErrorContext(r.Context(), "discovery search failed", "error", err)
			transport.WriteProblem(h.log, w, http.StatusInternalServerError, "internal_server_error", "discovery search failed")
		}
		return
	}
	h.log.DebugContext(r.Context(), "discovery search succeeded", "items", len(items))

	transport.WriteJSON(h.log, w, http.StatusOK, toResponse(items))
}
