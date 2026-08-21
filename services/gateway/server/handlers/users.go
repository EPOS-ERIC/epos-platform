package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

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
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}

	writeUserJSON(w, http.StatusOK, UserFromDomain(resolved))
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
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}

	writeUserSummariesJSON(w, http.StatusOK, UserSummariesFromDomain(resolved))
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

func (h *Users) changeGlobalPermission(w http.ResponseWriter, r *http.Request, grant bool) {
	user, ok := AuthenticatedUserFromContext(r.Context())
	if !ok {
		WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	permission := strings.TrimSpace(chi.URLParam(r, "permission"))
	if id == "" || permission == "" {
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
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
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
	if id == "" || role == "" {
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
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func UserFromDomain(resolved domain.User) api.User {
	user := api.User{
		ID:                resolved.ID,
		Email:             resolved.Email,
		GivenName:         resolved.GivenName,
		FamilyName:        resolved.FamilyName,
		Status:            api.UserStatus(resolved.Status),
		GlobalRoles:       append([]string{}, resolved.GlobalRoles...),
		GlobalPermissions: append([]string{}, resolved.GlobalPermissions...),
		CreatedAt:         resolved.CreatedAt,
		UpdatedAt:         resolved.UpdatedAt,
		Memberships:       make([]api.UserMembershipSummary, 0, len(resolved.Memberships)),
	}

	for _, membership := range resolved.Memberships {
		roles := make([]api.MembershipRole, 0, len(membership.Roles))
		for _, role := range membership.Roles {
			roles = append(roles, api.MembershipRole(role))
		}

		user.Memberships = append(user.Memberships, api.UserMembershipSummary{
			Group: api.ResourceRef{
				ID:   membership.Group.ID,
				Name: membership.Group.Name,
				Href: "/v1/groups/" + membership.Group.ID,
			},
			Roles:        roles,
			Status:       api.MembershipStatus(membership.Status),
			Permissions:  append([]string{}, membership.Permissions...),
			RoleRequests: roleRequestsFromDomain(membership.RoleRequests),
			CreatedAt:    membership.CreatedAt,
			UpdatedAt:    membership.UpdatedAt,
		})
	}

	return user
}

func roleRequestsFromDomain(requests []domain.RoleRequest) []api.RoleRequest {
	result := make([]api.RoleRequest, 0, len(requests))
	for _, request := range requests {
		result = append(result, api.RoleRequest{
			ID:          request.ID,
			Role:        request.Role,
			Status:      request.Status,
			RequestedAt: request.RequestedAt.Format(time.RFC3339Nano),
		})
	}
	return result
}

func UserSummariesFromDomain(resolved []domain.UserSummary) []api.UserSummary {
	result := make([]api.UserSummary, 0, len(resolved))
	for _, user := range resolved {
		result = append(result, api.UserSummary{
			ID:                user.ID,
			Email:             user.Email,
			GivenName:         user.GivenName,
			FamilyName:        user.FamilyName,
			Status:            api.UserStatus(user.Status),
			GlobalRoles:       append([]string{}, user.GlobalRoles...),
			GlobalPermissions: append([]string{}, user.GlobalPermissions...),
			CreatedAt:         user.CreatedAt,
			UpdatedAt:         user.UpdatedAt,
		})
	}

	return result
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
