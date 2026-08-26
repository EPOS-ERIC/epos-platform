package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/identity"
	"github.com/go-chi/chi/v5"
)

type Users struct {
	service *identity.Service
}

func NewUsers(service *identity.Service) *Users {
	return &Users{service: service}
}

func (h *Users) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := AuthenticatedUserFromContext(r.Context())
	if !ok {
		WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if id != "self" && !validUUID(id) {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid user id", nil)
		return
	}

	resolved, err := h.service.GetUser(r.Context(), user.ID, id)
	if err != nil {
		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid user id", err)
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "user read is not permitted", nil)
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			WriteError(r.Context(), w, http.StatusNotFound, "not_found", "user not found", nil)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "user request timed out", err)
			return
		}
		log.ErrorContext(r.Context(), "get user failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
		return
	}

	writeUserJSON(w, http.StatusOK, ToAPIUser(resolved))
}

func (h *Users) List(w http.ResponseWriter, r *http.Request) {
	user, ok := AuthenticatedUserFromContext(r.Context())
	if !ok {
		WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	resolved, err := h.service.ListUsers(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "user list is not permitted", nil)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "user request timed out", err)
			return
		}

		log.ErrorContext(r.Context(), "list users failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
		return
	}

	writeUserSummariesJSON(w, http.StatusOK, toAPIUserSummaries(resolved))
}

func (h *Users) GrantGlobalPermission(w http.ResponseWriter, r *http.Request) {
	h.changeGlobalPermission(w, r, true)
}

func (h *Users) RevokeGlobalPermission(w http.ResponseWriter, r *http.Request) {
	h.changeGlobalPermission(w, r, false)
}

func (h *Users) AssignGlobalRole(w http.ResponseWriter, r *http.Request) {
	h.changeGlobalRole(w, r, true)
}

func (h *Users) RemoveGlobalRole(w http.ResponseWriter, r *http.Request) {
	h.changeGlobalRole(w, r, false)
}

func (h *Users) BlockUser(w http.ResponseWriter, r *http.Request) {
	user, ok := AuthenticatedUserFromContext(r.Context())
	if !ok {
		WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	if !validUUID(chi.URLParam(r, "id")) {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid user id", nil)
		return
	}

	var request struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(r, w, &request) {
		return
	}
	if strings.TrimSpace(request.Reason) == "" {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "reason is required", nil)
		return
	}

	err := h.service.BlockUser(r.Context(), user.ID, chi.URLParam(r, "id"), request.Reason)
	if h.writeUserStateError(w, r, err, "block user", "cannot block the last active global administrator") {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Users) UnblockUser(w http.ResponseWriter, r *http.Request) {
	user, ok := AuthenticatedUserFromContext(r.Context())
	if !ok {
		WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if !validUUID(id) {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid user id", nil)
		return
	}

	err := h.service.UnblockUser(r.Context(), user.ID, id)
	if h.writeUserStateError(w, r, err, "unblock user", "cannot unblock user") {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TODO: see if we should make a more generic error wrapper for the user handlers
func (h *Users) writeUserStateError(w http.ResponseWriter, r *http.Request, err error, operation, conflictMessage string) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, domain.ErrBadRequest):
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid user state request", err)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "user state management is not permitted", nil)
	case errors.Is(err, domain.ErrNotFound):
		WriteError(r.Context(), w, http.StatusNotFound, "not_found", "user not found", nil)
	case errors.Is(err, domain.ErrConflict):
		WriteError(r.Context(), w, http.StatusConflict, "conflict", conflictMessage, nil)
	case errors.Is(err, domain.ErrUpstreamTimeout):
		WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "user request timed out", err)
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
	}
	return true
}

func (h *Users) changeGlobalPermission(w http.ResponseWriter, r *http.Request, grant bool) {
	user, ok := AuthenticatedUserFromContext(r.Context())
	if !ok {
		WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	permission := strings.TrimSpace(chi.URLParam(r, "permission"))
	if !validUUID(id) || !validGlobalPermission(permission) {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid user id or permission", nil)
		return
	}

	var err error
	if grant {
		err = h.service.GrantGlobalPermission(r.Context(), user.ID, id, permission)
	} else {
		err = h.service.RevokeGlobalPermission(r.Context(), user.ID, id, permission)
	}
	if err != nil {
		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid user id or permission", err)
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "global permission management is not permitted", nil)
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			WriteError(r.Context(), w, http.StatusNotFound, "not_found", "user not found", nil)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "user request timed out", err)
			return
		}

		log.ErrorContext(r.Context(), "change global permission failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Users) changeGlobalRole(w http.ResponseWriter, r *http.Request, assign bool) {
	user, ok := AuthenticatedUserFromContext(r.Context())
	if !ok {
		WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	role := strings.TrimSpace(chi.URLParam(r, "role"))
	if !validUUID(id) || role != "ADMIN" {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid user id or role", nil)
		return
	}

	var err error
	if assign {
		err = h.service.AssignGlobalRole(r.Context(), user.ID, id, role)
	} else {
		err = h.service.RemoveGlobalRole(r.Context(), user.ID, id, role)
	}
	if err != nil {
		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid user id or role", err)
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "global role management is not permitted", nil)
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			WriteError(r.Context(), w, http.StatusNotFound, "not_found", "user not found", nil)
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			WriteError(r.Context(), w, http.StatusConflict, "conflict", "cannot remove the last active global administrator", nil)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "user request timed out", err)
			return
		}

		log.ErrorContext(r.Context(), "change global role failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TODO: use an enum
func validGlobalPermission(permission string) bool {
	switch permission {
	case "global.membership.manage", "global.role.manage", "global.permission.manage", "organization.manage", "contact_point.manage", "category_scheme.manage":
		return true
	default:
		return false
	}
}

func writeUserJSON(w http.ResponseWriter, status int, value api.User) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Error("failed to encode user response", "error", err)
	}
}

func writeUserSummariesJSON(w http.ResponseWriter, status int, value []api.UserSummary) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Error("failed to encode user summaries response", "error", err)
	}
}
