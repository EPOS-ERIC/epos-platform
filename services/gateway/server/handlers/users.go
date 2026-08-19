package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

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
	resolved, err := h.service.GetUser(r.Context(), user, id)
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
	writeUserJSON(w, http.StatusOK, resolved)
}

func writeUserJSON(w http.ResponseWriter, status int, value api.User) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Error("failed to encode user response", "error", err)
	}
}
