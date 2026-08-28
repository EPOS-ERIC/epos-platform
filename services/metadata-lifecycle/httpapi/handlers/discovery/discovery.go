// Package discovery exposes resource discovery HTTP handlers.
package discovery

import (
	"context"
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
)

type Service interface {
	Search(context.Context, *domain.Actor, domain.DiscoveryFilter) ([]domain.DiscoveryItem, error)
}

type Handler struct {
	service Service
}

var log = logging.Get("discovery-handler")

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	actor, ok := transport.OptionalActorValue(w, r)
	if !ok {
		return
	}

	filter, err := filterFromRequest(r)
	if err != nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	items, err := h.service.Search(r.Context(), actor, filter)
	if err != nil {
		switch {
		case errors.Is(err, discovery.ErrInvalid):
			log.DebugContext(r.Context(), "discovery search invalid", "error", err)
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		case errors.Is(err, discovery.ErrDenied):
			log.DebugContext(r.Context(), "discovery search denied", "error", err)
			transport.WriteProblem(w, http.StatusUnauthorized, "unauthorized", "authentication is required for non-published discovery results")
		default:
			log.ErrorContext(r.Context(), "discovery search failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "discovery search failed")
		}
		return
	}

	transport.WriteJSON(w, http.StatusOK, toResponse(items))
}
