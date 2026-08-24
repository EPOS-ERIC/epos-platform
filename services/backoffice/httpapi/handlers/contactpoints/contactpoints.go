// Package contactpoints exposes protected canonical contact-point reads.
package contactpoints

import (
	"context"
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	contactpointdomain "github.com/EPOS-ERIC/epos-platform/services/backoffice/contactpoints"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/transport"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("contact-points-handler")

type service interface {
	List(context.Context, string) ([]contactpointdomain.ContactPoint, error)
	Get(context.Context, string, uuid.UUID) (contactpointdomain.ContactPoint, error)
	Create(context.Context, string, contactpointdomain.CreateRequest) (contactpointdomain.ContactPoint, error)
}

type Handler struct {
	service service
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	contactPoints, err := h.service.List(r.Context(), actorID)
	if err != nil {
		writeError(w, r, err, "list contact points")
		return
	}

	result := make([]contactPointResponse, 0, len(contactPoints))
	for _, contactPoint := range contactPoints {
		result = append(result, response(contactPoint))
	}

	transport.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "contact_point_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid contact point id")
		return
	}

	contactPoint, err := h.service.Get(r.Context(), actorID, id)
	if err != nil {
		writeError(w, r, err, "get contact point")
		return
	}

	transport.WriteJSON(w, http.StatusOK, response(contactPoint))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	var body createContactPointRequest
	if !transport.Decode(w, r, &body) {
		return
	}

	contactPoint, err := h.service.Create(r.Context(), actorID, contactpointdomain.CreateRequest{
		SourceIRI: body.SourceIRI,
		Email:     body.Email,
		Role:      body.Role,
	})
	if err != nil {
		writeError(w, r, err, "create contact point")
		return
	}

	transport.WriteJSON(w, http.StatusCreated, response(contactPoint))
}

func writeError(w http.ResponseWriter, r *http.Request, err error, operation string) {
	switch {
	case errors.Is(err, contactpointdomain.ErrInvalid):
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid contact point request")
	case errors.Is(err, contactpointdomain.ErrForbidden):
		transport.WriteProblem(w, http.StatusForbidden, "forbidden", "contact point read is not permitted")
	case errors.Is(err, contactpointdomain.ErrConflict):
		transport.WriteProblem(w, http.StatusConflict, "conflict", "contact point already exists")
	case errors.Is(err, contactpointdomain.ErrNotFound):
		transport.WriteProblem(w, http.StatusNotFound, "contact_point_not_found", "contact point not found")
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", operation+" failed")
	}
}

type contactPointResponse struct {
	ID        uuid.UUID `json:"id"`
	SourceIRI string    `json:"source_iri"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
}

type createContactPointRequest struct {
	SourceIRI string `json:"source_iri"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

func response(contactPoint contactpointdomain.ContactPoint) contactPointResponse {
	return contactPointResponse{
		ID:        contactPoint.ID,
		SourceIRI: contactPoint.SourceIRI,
		Email:     contactPoint.Email,
		Role:      contactPoint.Role,
	}
}
