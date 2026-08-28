// Package users exposes user resource operations.
package users

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("identity-handler")

type service interface {
	GetByID(context.Context, domain.Actor, string) (domain.User, error)
	List(context.Context, domain.Actor) ([]domain.UserSummary, error)
	Create(context.Context, domain.UserClaims) (domain.User, error)
	GrantGlobalPermission(context.Context, domain.Actor, string, string) error
	RevokeGlobalPermission(context.Context, domain.Actor, string, string) error
	AssignGlobalRole(context.Context, domain.Actor, string, string) error
	RemoveGlobalRole(context.Context, domain.Actor, string, string) error
	BlockUser(context.Context, domain.Actor, string, string) error
	UnblockUser(context.Context, domain.Actor, string) error
}

type Handler struct {
	service service
}

func New(service service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.ActorValue(w, r)
	if !ok {
		return
	}

	targetID, ok := userIDPath(w, r)
	if !ok {
		return
	}

	user, err := h.service.GetByID(r.Context(), actorID, targetID)
	if err != nil {
		writeError(w, r, err, "get user", "")
		return
	}

	transport.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.ActorValue(w, r)
	if !ok {
		return
	}

	users, err := h.service.List(r.Context(), actorID)
	if err != nil {
		writeError(w, r, err, "list users", "")
		return
	}

	result := make([]userSummaryResponse, 0, len(users))
	for _, user := range users {
		result = append(result, toUserSummaryResponse(user))
	}

	transport.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) GrantGlobalPermission(w http.ResponseWriter, r *http.Request) {
	h.changeGlobalPermission(w, r, true)
}

func (h *Handler) RevokeGlobalPermission(w http.ResponseWriter, r *http.Request) {
	h.changeGlobalPermission(w, r, false)
}

func (h *Handler) AssignGlobalRole(w http.ResponseWriter, r *http.Request) {
	h.changeGlobalRole(w, r, true)
}

func (h *Handler) RemoveGlobalRole(w http.ResponseWriter, r *http.Request) {
	h.changeGlobalRole(w, r, false)
}

func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.ActorValue(w, r)
	if !ok {
		return
	}

	userID, ok := userIDPath(w, r)
	if !ok {
		return
	}

	var body blockUserRequest
	if !transport.Decode(w, r, &body) {
		return
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if body.Reason == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "reason is required")
		return
	}

	if err := h.service.BlockUser(r.Context(), actorID, userID, body.Reason); err != nil {
		writeError(w, r, err, "block user", "cannot block the last active global administrator")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.ActorValue(w, r)
	if !ok {
		return
	}

	userID, ok := userIDPath(w, r)
	if !ok {
		return
	}

	if err := h.service.UnblockUser(r.Context(), actorID, userID); err != nil {
		writeError(w, r, err, "unblock user", "cannot unblock user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) changeGlobalPermission(w http.ResponseWriter, r *http.Request, grant bool) {
	actorID, userID, permission, ok := permissionPath(w, r)
	if !ok {
		return
	}

	var err error
	if grant {
		err = h.service.GrantGlobalPermission(r.Context(), actorID, userID, permission)
	} else {
		err = h.service.RevokeGlobalPermission(r.Context(), actorID, userID, permission)
	}

	if err != nil {
		writeError(w, r, err, "change global permission", "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) changeGlobalRole(w http.ResponseWriter, r *http.Request, assign bool) {
	actorID, userID, role, ok := rolePath(w, r)
	if !ok {
		return
	}

	var err error
	if assign {
		err = h.service.AssignGlobalRole(r.Context(), actorID, userID, role)
	} else {
		err = h.service.RemoveGlobalRole(r.Context(), actorID, userID, role)
	}
	if err != nil {
		writeError(w, r, err, "change global role", "cannot remove the last active global administrator")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body internalUserRequest
	if !transport.Decode(w, r, &body) {
		return
	}

	claims, err := toUserClaims(body)
	if err != nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "id, email, given_name, and family_name are required")
		return
	}

	user, err := h.service.Create(r.Context(), claims)
	if err != nil {
		writeError(w, r, err, "create user", "user already exists")
		return
	}

	transport.WriteJSON(w, http.StatusCreated, toUserResponse(user))
}

func userIDPath(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(chi.URLParam(r, "user_id"))
	if userID == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return "", false
	}

	return userID, true
}

func permissionPath(w http.ResponseWriter, r *http.Request) (domain.Actor, string, string, bool) {
	actorID, ok := transport.ActorValue(w, r)
	if !ok {
		return "", "", "", false
	}

	userID, ok := userIDPath(w, r)
	if !ok {
		return "", "", "", false
	}
	permission := strings.TrimSpace(chi.URLParam(r, "permission"))
	if permission == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid permission")
		return "", "", "", false
	}

	return actorID, userID, permission, true
}

func rolePath(w http.ResponseWriter, r *http.Request) (domain.Actor, string, string, bool) {
	actorID, ok := transport.ActorValue(w, r)
	if !ok {
		return "", "", "", false
	}

	userID, ok := userIDPath(w, r)
	if !ok {
		return "", "", "", false
	}
	role := strings.TrimSpace(chi.URLParam(r, "role"))
	if role != "ADMIN" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid global role")
		return "", "", "", false
	}

	return actorID, userID, role, true
}

func writeError(w http.ResponseWriter, r *http.Request, err error, operation, conflictMessage string) {
	switch {
	case errors.Is(err, domain.ErrInvalidUser):
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid user request")
	case errors.Is(err, domain.ErrUserForbidden):
		transport.WriteProblem(w, http.StatusForbidden, "forbidden", "user operation is not permitted")
	case errors.Is(err, domain.ErrUserNotFound):
		transport.WriteProblem(w, http.StatusNotFound, "user_not_found", "user not found")
	case errors.Is(err, domain.ErrUserConflict):
		if conflictMessage == "" {
			conflictMessage = "user operation conflicts with existing data"
		}
		transport.WriteProblem(w, http.StatusConflict, "conflict", conflictMessage)
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", operation+" failed")
	}
}
