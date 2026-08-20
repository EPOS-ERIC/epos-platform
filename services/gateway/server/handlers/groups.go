package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	groupservice "github.com/EPOS-ERIC/epos-platform/services/gateway/groups"
	"github.com/go-chi/chi/v5"
)

type Groups struct {
	service *groupservice.Service
}

func NewGroups(service *groupservice.Service) *Groups {
	return &Groups{service: service}
}

func (h *Groups) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.List(r.Context())
	if err != nil {
		writeGroupError(r, w, err, "list groups")
		return
	}

	result := make([]api.Group, 0, len(groups))
	for _, group := range groups {
		result = append(result, GroupFromDomain(group))
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Groups) Get(w http.ResponseWriter, r *http.Request) {
	group, err := h.service.Get(r.Context(), chi.URLParam(r, "groupId"))
	if err != nil {
		writeGroupError(r, w, err, "get group")
		return
	}

	writeJSON(w, http.StatusOK, GroupFromDomain(group))
}

func GroupFromDomain(group domain.Group) api.Group {
	return api.Group{
		ID:          group.ID,
		SourceIRI:   group.SourceIRI,
		Name:        group.Name,
		Description: group.Description,
		Label:       group.Label,
		Logo:        group.Logo,
		Page:        group.Page,
		Color:       group.Color,
		Position:    group.Position,
		Concept:     group.Concept,
	}
}

func writeGroupError(r *http.Request, w http.ResponseWriter, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrBadRequest):
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid group id", err)
	case errors.Is(err, domain.ErrNotFound):
		WriteError(r.Context(), w, http.StatusNotFound, "not_found", "group not found", nil)
	case errors.Is(err, domain.ErrUpstreamTimeout):
		WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Error("failed to encode group response", "error", err)
	}
}
