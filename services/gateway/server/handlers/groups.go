package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/usergroup"
	"github.com/go-chi/chi/v5"
)

type Groups struct {
	service *usergroup.Service
	log     *slog.Logger
}

func NewGroups(service *usergroup.Service) *Groups {
	return &Groups{
		service: service,
		log:     logging.Get("groups-handler"),
	}
}

func (h *Groups) CreateGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticatedUser, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	var req api.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain slug and name", nil)
		return
	}
	if strings.TrimSpace(req.Slug) == "" || strings.TrimSpace(req.Name) == "" {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain slug and name", nil)
		return
	}

	group, err := h.service.CreateGroup(ctx, authenticatedUser, req)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			h.log.WarnContext(ctx, "group access denied", "user_id", authenticatedUser.ID)
			WriteError(ctx, w, http.StatusForbidden, "forbidden", "group access denied", nil)
			return
		}

		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrConflict) {
			WriteError(ctx, w, http.StatusConflict, "conflict", "group already exists", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "group create request failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "group request failed", err)
		return
	}

	writeJSON(w, http.StatusCreated, group)
}

func (h *Groups) ListGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticatedUser, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groups, err := h.service.ListGroups(ctx, authenticatedUser)
	if err != nil {
		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "group list request failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "group request failed", err)
		return
	}

	writeJSON(w, http.StatusOK, groups)
}

func (h *Groups) GetGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticatedUser, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID := chi.URLParam(r, "groupId")
	group, err := h.service.GetGroup(ctx, authenticatedUser, groupID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			WriteError(ctx, w, http.StatusNotFound, "group_not_found", "group not found", nil)
			return
		}

		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "group get request failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "group request failed", err)
		return
	}

	writeJSON(w, http.StatusOK, group)
}

func (h *Groups) ListMemberships(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticatedUser, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID := chi.URLParam(r, "groupId")
	memberships, err := h.service.ListGroupMemberships(ctx, authenticatedUser, groupID)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			h.log.WarnContext(ctx, "group access denied", "user_id", authenticatedUser.ID, "group_id", groupID)
			WriteError(ctx, w, http.StatusForbidden, "forbidden", "group access denied", nil)
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			WriteError(ctx, w, http.StatusNotFound, "group_not_found", "group not found", nil)
			return
		}

		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "group membership list failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "group request failed", err)
		return
	}

	writeJSON(w, http.StatusOK, memberships)
}

func (h *Groups) PatchMembership(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticatedUser, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID := chi.URLParam(r, "groupId")
	userID := chi.URLParam(r, "userId")

	var req api.PatchMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain role or status", nil)
		return
	}
	if req.IsEmpty() {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain role or status", nil)
		return
	}
	if req.Role != nil && !req.Role.IsValid() {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain valid role", nil)
		return
	}
	if req.Status != nil && !req.Status.IsValid() {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain valid status", nil)
		return
	}

	membership, err := h.service.PatchMembership(ctx, authenticatedUser, groupID, userID, req)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			h.log.WarnContext(ctx, "group access denied", "user_id", authenticatedUser.ID, "group_id", groupID, "membership_user_id", userID)
			WriteError(ctx, w, http.StatusForbidden, "forbidden", "group access denied", nil)
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			WriteError(ctx, w, http.StatusNotFound, "membership_not_found", "membership not found", nil)
			return
		}

		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "group membership patch failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "group request failed", err)
		return
	}

	writeJSON(w, http.StatusOK, membership)
}

func (h *Groups) DeleteMembership(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticatedUser, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID := chi.URLParam(r, "groupId")
	userID := chi.URLParam(r, "userId")

	if err := h.service.DeleteMembership(ctx, authenticatedUser, groupID, userID); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			h.log.WarnContext(ctx, "group access denied", "user_id", authenticatedUser.ID, "group_id", groupID, "membership_user_id", userID)
			WriteError(ctx, w, http.StatusForbidden, "forbidden", "group access denied", nil)
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			WriteError(ctx, w, http.StatusNotFound, "membership_not_found", "membership not found", nil)
			return
		}

		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "group membership delete failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "group request failed", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Groups) PatchGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticatedUser, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID := chi.URLParam(r, "groupId")
	var req api.PatchGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain mutable group fields", nil)
		return
	}
	if req.IsEmpty() {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain mutable group fields", nil)
		return
	}

	group, err := h.service.PatchGroup(ctx, authenticatedUser, groupID, req)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			h.log.WarnContext(ctx, "group access denied", "user_id", authenticatedUser.ID, "group_id", groupID)
			WriteError(ctx, w, http.StatusForbidden, "forbidden", "group access denied", nil)
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			WriteError(ctx, w, http.StatusNotFound, "group_not_found", "group not found", nil)
			return
		}

		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrConflict) {
			WriteError(ctx, w, http.StatusConflict, "conflict", "group already exists", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "group patch request failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "group request failed", err)
		return
	}

	writeJSON(w, http.StatusOK, group)
}

func (h *Groups) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticatedUser, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID := chi.URLParam(r, "groupId")
	if err := h.service.DeleteGroup(ctx, authenticatedUser, groupID); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			h.log.WarnContext(ctx, "group access denied", "user_id", authenticatedUser.ID, "group_id", groupID)
			WriteError(ctx, w, http.StatusForbidden, "forbidden", "group access denied", nil)
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			WriteError(ctx, w, http.StatusNotFound, "group_not_found", "group not found", nil)
			return
		}

		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "group delete request failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "group request failed", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Groups) RequestMembership(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticatedUser, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID := chi.URLParam(r, "groupId")
	var req api.CreateMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain role", nil)
		return
	}
	if !req.Role.IsValid() {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request body must contain role", nil)
		return
	}

	membership, err := h.service.RequestMembership(ctx, authenticatedUser, groupID, req)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			h.log.WarnContext(ctx, "group access denied", "user_id", authenticatedUser.ID, "group_id", groupID)
			WriteError(ctx, w, http.StatusForbidden, "forbidden", "group access denied", nil)
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			WriteError(ctx, w, http.StatusNotFound, "group_not_found", "group not found", nil)
			return
		}

		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by user service", nil)
			return
		}

		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
			return
		}

		h.log.ErrorContext(ctx, "group membership request failed", "error", err)
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "group request failed", err)
		return
	}

	writeJSON(w, http.StatusCreated, membership)
}
