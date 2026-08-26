// Package identity exposes internal user resource operations.
package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/identity"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("identity-handler")

type service interface {
	GetByID(context.Context, string, string) (identity.User, error)
	List(context.Context, string) ([]identity.UserSummary, error)
	Create(context.Context, identity.Claims) (identity.User, error)
	GrantGlobalPermission(context.Context, string, string, string) error
	RevokeGlobalPermission(context.Context, string, string, string) error
	AssignGlobalRole(context.Context, string, string, string) error
	RemoveGlobalRole(context.Context, string, string, string) error
	BlockUser(context.Context, string, string, string) error
	UnblockUser(context.Context, string, string) error
}

type Handler struct {
	service service
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	targetID := strings.TrimSpace(chi.URLParam(r, "user_id"))
	if targetID == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return
	}

	user, err := h.service.GetByID(r.Context(), actorID, targetID)
	if err != nil {
		if errors.Is(err, identity.ErrForbidden) {
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "user read is not permitted")
			return
		}

		if errors.Is(err, identity.ErrNotFound) {
			transport.WriteProblem(w, http.StatusNotFound, "user_not_found", "user not found")
			return
		}

		log.ErrorContext(r.Context(), "get user failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "get user failed")

		return
	}

	transport.WriteJSON(w, http.StatusOK, response(user))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	users, err := h.service.List(r.Context(), actorID)
	if err != nil {
		if errors.Is(err, identity.ErrForbidden) {
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "user list is not permitted")
			return
		}

		log.ErrorContext(r.Context(), "list users failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "list users failed")

		return
	}

	result := make([]userSummaryResponse, 0, len(users))
	for _, user := range users {
		result = append(result, userSummaryResponse{
			ID:                user.ID,
			Email:             user.Email,
			GivenName:         user.GivenName,
			FamilyName:        user.FamilyName,
			Status:            user.Status,
			GlobalRoles:       append([]string{}, user.GlobalRoles...),
			GlobalPermissions: append([]string{}, user.GlobalPermissions...),
		})
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
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	userID := strings.TrimSpace(chi.URLParam(r, "user_id"))
	var request struct {
		Reason string `json:"reason"`
	}
	if !transport.Decode(w, r, &request) {
		return
	}
	if userID == "" || strings.TrimSpace(request.Reason) == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "user id and non-empty reason are required")
		return
	}

	err := h.service.BlockUser(r.Context(), actorID, userID, request.Reason)
	if err != nil {
		h.writeUserStateError(w, r, err, "block user", "cannot block the last active global administrator")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	userID := strings.TrimSpace(chi.URLParam(r, "user_id"))
	if userID == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return
	}

	err := h.service.UnblockUser(r.Context(), actorID, userID)
	if err != nil {
		h.writeUserStateError(w, r, err, "unblock user", "cannot unblock user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TODO: see if we should make a more generic user error wrapper to reuse in this handler
func (h *Handler) writeUserStateError(w http.ResponseWriter, r *http.Request, err error, operation, conflictMessage string) {
	if errors.Is(err, identity.ErrInvalid) {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid user state request")
		return
	}
	if errors.Is(err, identity.ErrForbidden) {
		transport.WriteProblem(w, http.StatusForbidden, "forbidden", "user state management is not permitted")
		return
	}
	if errors.Is(err, identity.ErrNotFound) {
		transport.WriteProblem(w, http.StatusNotFound, "user_not_found", "user not found")
		return
	}
	if errors.Is(err, identity.ErrConflict) {
		transport.WriteProblem(w, http.StatusConflict, "conflict", conflictMessage)
		return
	}
	log.ErrorContext(r.Context(), operation+" failed", "error", err)
	transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", operation+" failed")
}

func (h *Handler) changeGlobalPermission(w http.ResponseWriter, r *http.Request, grant bool) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	targetID := strings.TrimSpace(chi.URLParam(r, "user_id"))
	permission := strings.TrimSpace(chi.URLParam(r, "permission"))
	if targetID == "" || permission == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid user id or permission")
		return
	}

	var err error
	if grant {
		err = h.service.GrantGlobalPermission(r.Context(), actorID, targetID, permission)
	} else {
		err = h.service.RevokeGlobalPermission(r.Context(), actorID, targetID, permission)
	}

	if err != nil {
		if errors.Is(err, identity.ErrForbidden) {
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "global permission management is not permitted")
			return
		}

		if errors.Is(err, identity.ErrInvalid) {
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid global permission")
			return
		}

		if errors.Is(err, identity.ErrNotFound) {
			transport.WriteProblem(w, http.StatusNotFound, "user_not_found", "user not found")
			return
		}

		log.ErrorContext(r.Context(), "change global permission failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "change global permission failed")

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) changeGlobalRole(w http.ResponseWriter, r *http.Request, assign bool) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}
	targetID := strings.TrimSpace(chi.URLParam(r, "user_id"))
	role := strings.TrimSpace(chi.URLParam(r, "role"))
	if targetID == "" || role == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid user id or role")
		return
	}

	var err error
	if assign {
		err = h.service.AssignGlobalRole(r.Context(), actorID, targetID, role)
	} else {
		err = h.service.RemoveGlobalRole(r.Context(), actorID, targetID, role)
	}
	if err != nil {
		if errors.Is(err, identity.ErrForbidden) {
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "global role management is not permitted")
			return
		}
		if errors.Is(err, identity.ErrInvalid) {
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid global role")
			return
		}
		if errors.Is(err, identity.ErrNotFound) {
			transport.WriteProblem(w, http.StatusNotFound, "user_not_found", "user not found")
			return
		}
		if errors.Is(err, identity.ErrConflict) {
			transport.WriteProblem(w, http.StatusConflict, "conflict", "cannot remove the last active global administrator")
			return
		}

		log.ErrorContext(r.Context(), "change global role failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "change global role failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ID         string `json:"id"`
		Email      string `json:"email"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
	}
	if !transport.Decode(w, r, &request) {
		return
	}

	user, err := h.service.Create(r.Context(), identity.Claims{
		ID:         request.ID,
		Email:      request.Email,
		GivenName:  request.GivenName,
		FamilyName: request.FamilyName,
	})
	if err != nil {
		if errors.Is(err, identity.ErrInvalid) {
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "id, email, given_name, and family_name are required")
			return
		}

		if strings.Contains(err.Error(), "duplicate") {
			transport.WriteProblem(w, http.StatusConflict, "conflict", "user already exists")
			return
		}

		log.ErrorContext(r.Context(), "create user failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "create user failed")

		return
	}

	transport.WriteJSON(w, http.StatusCreated, response(user))
}

type userResponse struct {
	ID                string               `json:"id"`
	Email             string               `json:"email"`
	GivenName         string               `json:"given_name"`
	FamilyName        string               `json:"family_name"`
	Status            string               `json:"status"`
	GlobalRoles       []string             `json:"global_roles"`
	GlobalPermissions []string             `json:"global_permissions"`
	CreatedAt         string               `json:"created_at"`
	UpdatedAt         string               `json:"updated_at"`
	Memberships       []membershipResponse `json:"memberships"`
}

type userSummaryResponse struct {
	ID                string   `json:"id"`
	Email             string   `json:"email"`
	GivenName         string   `json:"given_name"`
	FamilyName        string   `json:"family_name"`
	Status            string   `json:"status"`
	GlobalRoles       []string `json:"global_roles"`
	GlobalPermissions []string `json:"global_permissions"`
}

type membershipResponse struct {
	Group        groupResponse `json:"group"`
	Roles        []string      `json:"roles"`
	Status       string        `json:"status"`
	Permissions  []string      `json:"permissions"`
	RoleRequests []roleRequest `json:"role_requests"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
}

type roleRequest struct {
	ID          uuid.UUID `json:"id"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	RequestedAt string    `json:"requested_at"`
}

type groupResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func response(user identity.User) userResponse {
	memberships := make([]membershipResponse, 0, len(user.Memberships))
	for _, membership := range user.Memberships {
		memberships = append(memberships, membershipResponse{
			Group: groupResponse{
				ID:   membership.GroupID,
				Name: membership.GroupName,
			},
			Roles:        append([]string{}, membership.Roles...),
			Status:       membership.Status,
			Permissions:  append([]string{}, membership.Permissions...),
			RoleRequests: roleRequests(membership.RoleRequests),
			CreatedAt:    membership.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt:    membership.UpdatedAt.Format(time.RFC3339Nano),
		})
	}

	return userResponse{
		ID:                user.ID,
		Email:             user.Email,
		GivenName:         user.GivenName,
		FamilyName:        user.FamilyName,
		Status:            user.Status,
		GlobalRoles:       append([]string{}, user.GlobalRoles...),
		GlobalPermissions: append([]string{}, user.GlobalPermissions...),
		CreatedAt:         user.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:         user.UpdatedAt.Format(time.RFC3339Nano),
		Memberships:       memberships,
	}
}

func roleRequests(requests []identity.RoleRequest) []roleRequest {
	result := make([]roleRequest, 0, len(requests))
	for _, request := range requests {
		result = append(result, roleRequest{
			ID:          request.ID,
			Role:        request.Role,
			Status:      request.Status,
			RequestedAt: request.RequestedAt.Format(time.RFC3339Nano),
		})
	}
	return result
}
