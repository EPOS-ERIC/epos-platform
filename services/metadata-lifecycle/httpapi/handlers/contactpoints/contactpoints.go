// Package contactpoints exposes protected canonical contact-point reads.
package contactpoints

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type service interface {
	List(context.Context, domain.Actor) ([]domain.ContactPoint, error)
	Get(context.Context, domain.Actor, uuid.UUID) (domain.ContactPoint, error)
	Create(context.Context, domain.Actor, domain.ContactPointInput) (domain.ContactPoint, error)
	Replace(context.Context, domain.Actor, uuid.UUID, domain.ContactPointInput) (domain.ContactPoint, error)
	Delete(context.Context, domain.Actor, uuid.UUID) error
}

type Handler struct {
	service service
	log     *slog.Logger
}

func New(log *slog.Logger, service service) *Handler {
	return &Handler{
		service: service,
		log: log.With("component", "contact-points-handler"),
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	contactPoints, err := h.service.List(r.Context(), actorID)
	if err != nil {
		writeError(h.log, w, r, err, "list contact points")
		return
	}

	result := make([]contactPointResponse, 0, len(contactPoints))
	for _, contactPoint := range contactPoints {
		result = append(result, toResponse(contactPoint))
	}

	transport.WriteJSON(h.log, w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "contact_point_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid contact point id")
		return
	}

	contactPoint, err := h.service.Get(r.Context(), actorID, id)
	if err != nil {
		writeError(h.log, w, r, err, "get contact point")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusOK, toResponse(contactPoint))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	var body contactPointRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}

	input, err := toDomainContactPointInput(body)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid contact point")
		return
	}

	contactPoint, err := h.service.Create(r.Context(), actorID, input)
	if err != nil {
		writeError(h.log, w, r, err, "create contact point")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusCreated, toResponse(contactPoint))
}

func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "contact_point_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid contact point id")
		return
	}

	var body contactPointRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}

	input, err := toDomainContactPointInput(body)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid contact point")
		return
	}

	contactPoint, err := h.service.Replace(r.Context(), actorID, id, input)
	if err != nil {
		writeError(h.log, w, r, err, "replace contact point")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusOK, toResponse(contactPoint))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "contact_point_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid contact point id")
		return
	}

	if err := h.service.Delete(r.Context(), actorID, id); err != nil {
		writeError(h.log, w, r, err, "delete contact point")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeError(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrInvalidContactPoint):
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid contact point request")
	case errors.Is(err, domain.ErrContactPointForbidden):
		transport.WriteProblem(log, w, http.StatusForbidden, "forbidden", "contact point operation is not permitted")
	case errors.Is(err, domain.ErrContactPointConflict):
		transport.WriteProblem(log, w, http.StatusConflict, "conflict", "contact point already exists")
	case errors.Is(err, domain.ErrContactPointNotFound):
		transport.WriteProblem(log, w, http.StatusNotFound, "contact_point_not_found", "contact point not found")
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		transport.WriteProblem(log, w, http.StatusInternalServerError, "internal_server_error", operation+" failed")
	}
}
