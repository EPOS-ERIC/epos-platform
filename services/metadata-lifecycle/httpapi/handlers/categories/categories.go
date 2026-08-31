// Package categories exposes group category endpoints.
package categories

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
	List(context.Context, uuid.UUID) ([]domain.Category, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.Category, error)
	Create(context.Context, domain.Actor, uuid.UUID, domain.CategoryInput) (domain.Category, error)
	Replace(context.Context, domain.Actor, uuid.UUID, uuid.UUID, domain.CategoryInput) (domain.Category, error)
	Delete(context.Context, domain.Actor, uuid.UUID, uuid.UUID) error
}

type Handler struct {
	service service
	log     *slog.Logger
}

func New(log *slog.Logger, service service) *Handler {
	return &Handler{
		service: service,
		log:     log.With("component", "categories-handler"),
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	groupID, ok := groupIDPath(h.log, w, r)
	if !ok {
		return
	}

	items, err := h.service.List(r.Context(), groupID)
	if err != nil {
		writeError(h.log, w, r, err, "list group categories")
		return
	}

	result := make([]categoryResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toResponse(item))
	}

	transport.WriteJSON(h.log, w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	groupID, categoryID, ok := categoryPath(h.log, w, r)
	if !ok {
		return
	}

	item, err := h.service.Get(r.Context(), groupID, categoryID)
	if err != nil {
		writeError(h.log, w, r, err, "get group category")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusOK, toResponse(item))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	groupID, ok := groupIDPath(h.log, w, r)
	if !ok {
		return
	}
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	var body categoryRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}

	input, err := toDomainCategoryInput(body)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid category")
		return
	}

	item, err := h.service.Create(r.Context(), actorID, groupID, input)
	if err != nil {
		writeError(h.log, w, r, err, "create group category")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusCreated, toResponse(item))
}

func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	groupID, categoryID, ok := categoryPath(h.log, w, r)
	if !ok {
		return
	}
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	var body categoryRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}

	input, err := toDomainCategoryInput(body)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid category")
		return
	}

	item, err := h.service.Replace(r.Context(), actorID, groupID, categoryID, input)
	if err != nil {
		writeError(h.log, w, r, err, "replace group category")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusOK, toResponse(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	groupID, categoryID, ok := categoryPath(h.log, w, r)
	if !ok {
		return
	}
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), actorID, groupID, categoryID); err != nil {
		writeError(h.log, w, r, err, "delete group category")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func groupIDPath(log *slog.Logger, w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	// TODO: we should probably create a shared package for these kind of functions tha are reused in many handlers
	id, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid group id")
		return uuid.Nil, false
	}
	return id, true
}

func categoryPath(log *slog.Logger, w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	groupID, ok := groupIDPath(log, w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	categoryID, err := uuid.Parse(chi.URLParam(r, "category_id"))
	if err != nil || categoryID == uuid.Nil {
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid category id")
		return uuid.Nil, uuid.Nil, false
	}

	return groupID, categoryID, true
}

func writeError(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrInvalidCategory):
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid category request")
	case errors.Is(err, domain.ErrCategoryForbidden):
		transport.WriteProblem(log, w, http.StatusForbidden, "forbidden", "category management permission required")
	case errors.Is(err, domain.ErrCategoryNotFound):
		transport.WriteProblem(log, w, http.StatusNotFound, "category_not_found", "category not found")
	case errors.Is(err, domain.ErrCategoryConflict):
		transport.WriteProblem(log, w, http.StatusConflict, "conflict", "category conflicts with existing category")
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		transport.WriteProblem(log, w, http.StatusInternalServerError, "internal_server_error", operation+" failed")
	}
}
