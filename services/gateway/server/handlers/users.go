package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/usergroup"
	"github.com/go-chi/chi/v5"
)

type Users struct {
	service *usergroup.Service
	log     *slog.Logger
}

func NewUsers(service *usergroup.Service) *Users {
	return &Users{
		service: service,
		log:     logging.Get("users-handler"),
	}
}

func (h *Users) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	user, err := h.service.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			WriteError(ctx, w, http.StatusNotFound, "user_not_found", "user not found", nil)
			return
		}

		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "user request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "user request failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "user request failed", err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Users) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := h.service.ListUsers(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "user request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "user list request failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "user request failed", err)
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *Users) PatchUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	var patch api.PatchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain only mutable user fields", nil)
		return
	}

	if patch.IsEmpty() {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain only mutable user fields", nil)
		return
	}

	user, err := h.service.PatchUser(ctx, id, patch)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			WriteError(ctx, w, http.StatusNotFound, "user_not_found", "user not found", nil)
			return
		}

		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "user request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "user patch request failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "user request failed", err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}
