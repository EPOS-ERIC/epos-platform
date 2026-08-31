// Package groups exposes public group metadata endpoints.
package groups

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
	Create(context.Context, domain.Actor, domain.GroupInput) (domain.Group, error)
	Replace(context.Context, domain.Actor, uuid.UUID, domain.GroupInput) (domain.Group, error)
	Delete(context.Context, domain.Actor, uuid.UUID) error
	List(context.Context) ([]domain.Group, error)
	Get(context.Context, uuid.UUID) (domain.Group, error)
}

type Handler struct {
	service service
	log     *slog.Logger
}

func New(log *slog.Logger, service service) *Handler {
	return &Handler{
		service: service,
		log:     log.With("component", "groups-handler"),
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	var body groupRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}

	input, err := toDomainGroupInput(body)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid group")
		return
	}

	group, err := h.service.Create(r.Context(), actorID, input)
	if err != nil {
		writeError(h.log, w, r, err, "create group")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusCreated, toResponse(group))
}

func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	id, ok := groupIDPath(h.log, w, r)
	if !ok {
		return
	}

	var body groupRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}

	input, err := toDomainGroupInput(body)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid group")
		return
	}

	group, err := h.service.Replace(r.Context(), actorID, id, input)
	if err != nil {
		writeError(h.log, w, r, err, "replace group")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusOK, toResponse(group))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	id, ok := groupIDPath(h.log, w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), actorID, id); err != nil {
		writeError(h.log, w, r, err, "delete group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.List(r.Context())
	if err != nil {
		writeError(h.log, w, r, err, "list groups")
		return
	}

	result := make([]groupResponse, 0, len(groups))
	for _, group := range groups {
		result = append(result, toResponse(group))
	}

	transport.WriteJSON(h.log, w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := groupIDPath(h.log, w, r)
	if !ok {
		return
	}

	group, err := h.service.Get(r.Context(), id)
	if err != nil {
		writeError(h.log, w, r, err, "get group")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusOK, toResponse(group))
}

func groupIDPath(log *slog.Logger, w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid group id")
		return uuid.Nil, false
	}
	return id, true
}

func writeError(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrInvalidGroup):
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid group request")
	case errors.Is(err, domain.ErrGroupForbidden):
		transport.WriteProblem(log, w, http.StatusForbidden, "forbidden", "group operation is not permitted")
	case errors.Is(err, domain.ErrGroupConflict):
		transport.WriteProblem(log, w, http.StatusConflict, "conflict", "group operation conflicts with existing data")
	case errors.Is(err, domain.ErrGroupNotFound):
		transport.WriteProblem(log, w, http.StatusNotFound, "group_not_found", "group not found")
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		transport.WriteProblem(log, w, http.StatusInternalServerError, "internal_server_error", operation+" failed")
	}
}
