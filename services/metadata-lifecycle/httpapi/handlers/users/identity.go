// Package users exposes user resource operations.
package users

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/go-chi/chi/v5"
)

type service interface {
	GetByID(context.Context, domain.Actor, string) (domain.User, error)
	Create(context.Context, domain.UserClaims) (domain.User, error)
}

type Handler struct {
	service service
	log     *slog.Logger
}

func New(log *slog.Logger, service service) *Handler {
	return &Handler{
		service: service,
		log:     log.With("component", "users-handler"),
	}
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	targetID, ok := userIDPath(h.log, w, r)
	if !ok {
		return
	}

	user, err := h.service.GetByID(r.Context(), actorID, targetID)
	if err != nil {
		writeError(h.log, w, r, err, "get user", "")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusOK, toUserResponse(user))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body internalUserRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}

	claims, err := toUserClaims(body)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "id, email, given_name, and family_name are required")
		return
	}

	user, err := h.service.Create(r.Context(), claims)
	if err != nil {
		writeError(h.log, w, r, err, "create user", "user already exists")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusCreated, toUserResponse(user))
}

func userIDPath(log *slog.Logger, w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(chi.URLParam(r, "user_id"))
	if userID == "" {
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return "", false
	}

	return userID, true
}

func writeError(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error, operation, conflictMessage string) {
	switch {
	case errors.Is(err, domain.ErrInvalidUser):
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid user request")
	case errors.Is(err, domain.ErrUserForbidden):
		transport.WriteProblem(log, w, http.StatusForbidden, "forbidden", "user operation is not permitted")
	case errors.Is(err, domain.ErrUserNotFound):
		transport.WriteProblem(log, w, http.StatusNotFound, "user_not_found", "user not found")
	case errors.Is(err, domain.ErrUserConflict):
		if conflictMessage == "" {
			conflictMessage = "user operation conflicts with existing data"
		}
		transport.WriteProblem(log, w, http.StatusConflict, "conflict", conflictMessage)
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		transport.WriteProblem(log, w, http.StatusInternalServerError, "internal_server_error", operation+" failed")
	}
}
