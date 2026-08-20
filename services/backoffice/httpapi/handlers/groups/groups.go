// Package groups exposes public group metadata endpoints.
package groups

import (
	"context"
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	groupdomain "github.com/EPOS-ERIC/epos-platform/services/backoffice/groups"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/transport"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("groups-handler")

type service interface {
	List(context.Context) ([]groupdomain.Group, error)
	Get(context.Context, uuid.UUID) (groupdomain.Group, error)
}

type Handler struct {
	service service
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.List(r.Context())
	if err != nil {
		log.ErrorContext(r.Context(), "list groups failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "list groups failed")
		return
	}

	result := make([]groupResponse, 0, len(groups))
	for _, group := range groups {
		result = append(result, response(group))
	}

	transport.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group id")
		return
	}

	group, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, groupdomain.ErrNotFound) {
			transport.WriteProblem(w, http.StatusNotFound, "group_not_found", "group not found")
			return
		}

		log.ErrorContext(r.Context(), "get group failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "get group failed")
		return
	}

	transport.WriteJSON(w, http.StatusOK, response(group))
}

type groupResponse struct {
	ID          uuid.UUID `json:"id"`
	SourceIRI   string    `json:"source_iri"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Label       *string   `json:"label"`
	Logo        *string   `json:"logo"`
	Page        *string   `json:"page"`
	Color       *string   `json:"color"`
	Position    int32     `json:"position"`
	Concept     string    `json:"concept"`
}

func response(group groupdomain.Group) groupResponse {
	return groupResponse{
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
