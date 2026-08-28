// Package organizations exposes public canonical organization endpoints.
package organizations

import (
	"context"
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	organizationservice "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/organizations"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("organizations-handler")

type service interface {
	List(context.Context) ([]domain.Organization, error)
	Get(context.Context, uuid.UUID) (domain.Organization, error)
	Create(context.Context, string, domain.OrganizationInput) (domain.Organization, error)
	Replace(context.Context, string, uuid.UUID, domain.OrganizationInput) (domain.Organization, error)
	Delete(context.Context, string, uuid.UUID) error
}

type Handler struct {
	service service
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	var body organizationRequest
	if !transport.Decode(w, r, &body) {
		return
	}

	input, err := toDomainOrganizationInput(body)
	if err != nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization")
		return
	}
	item, err := h.service.Create(r.Context(), actorID, input)
	if err != nil {
		switch {
		case errors.Is(err, organizationservice.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization")
		case errors.Is(err, organizationservice.ErrForbidden):
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "organization management permission required")
		case errors.Is(err, organizationservice.ErrConflict):
			transport.WriteProblem(w, http.StatusConflict, "conflict", "organization already exists")
		case errors.Is(err, organizationservice.ErrNotFound):
			transport.WriteProblem(w, http.StatusNotFound, "organization_not_found", "referenced organization or contact point not found")
		default:
			log.ErrorContext(r.Context(), "create organization failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "create organization failed")
		}
		return
	}

	transport.WriteJSON(w, http.StatusCreated, toResponse(item))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		// TODO: maybe we could log directly in the write problem function?
		log.ErrorContext(r.Context(), "list organizations failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "list organizations failed")
		return
	}

	result := make([]organizationResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toResponse(item))
	}

	transport.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "organization_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization id")
		return
	}

	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, organizationservice.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization")
		case errors.Is(err, organizationservice.ErrNotFound):
			transport.WriteProblem(w, http.StatusNotFound, "organization_not_found", "organization not found")
		default:
			log.ErrorContext(r.Context(), "get organization failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "get organization failed")
		}
		return
	}

	transport.WriteJSON(w, http.StatusOK, toResponse(item))
}

func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "organization_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization id")
		return
	}

	var body organizationRequest
	if !transport.Decode(w, r, &body) {
		return
	}
	input, err := toDomainOrganizationInput(body)
	if err != nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization")
		return
	}
	item, err := h.service.Replace(r.Context(), actorID, id, input)
	if err != nil {
		switch {
		case errors.Is(err, organizationservice.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization")
		case errors.Is(err, organizationservice.ErrForbidden):
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "organization management permission required")
		case errors.Is(err, organizationservice.ErrNotFound):
			transport.WriteProblem(w, http.StatusNotFound, "organization_not_found", "organization not found")
		case errors.Is(err, organizationservice.ErrConflict):
			transport.WriteProblem(w, http.StatusConflict, "conflict", "organization already exists")
		default:
			log.ErrorContext(r.Context(), "replace organization failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "replace organization failed")
		}
		return
	}

	transport.WriteJSON(w, http.StatusOK, toResponse(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "organization_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization id")
		return
	}

	if err := h.service.Delete(r.Context(), actorID, id); err != nil {
		switch {
		case errors.Is(err, organizationservice.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid organization")
		case errors.Is(err, organizationservice.ErrForbidden):
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "organization management permission required")
		case errors.Is(err, organizationservice.ErrNotFound):
			transport.WriteProblem(w, http.StatusNotFound, "organization_not_found", "organization not found")
		default:
			log.ErrorContext(r.Context(), "delete organization failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "delete organization failed")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
