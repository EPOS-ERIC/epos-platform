package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	groupservice "github.com/EPOS-ERIC/epos-platform/services/gateway/groups"
	"github.com/go-chi/chi/v5"
)

type Groups struct {
	service *groupservice.Service
}

func NewGroups(service *groupservice.Service) *Groups {
	return &Groups{service: service}
}

func (h *Groups) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.List(r.Context())
	if err != nil {
		writeGroupError(r, w, err, "list groups")
		return
	}

	result := make([]api.Group, 0, len(groups))
	for _, group := range groups {
		result = append(result, GroupFromDomain(group))
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Groups) Get(w http.ResponseWriter, r *http.Request) {
	group, err := h.service.Get(r.Context(), chi.URLParam(r, "groupId"))
	if err != nil {
		writeGroupError(r, w, err, "get group")
		return
	}

	writeJSON(w, http.StatusOK, GroupFromDomain(group))
}

// TODO: since the handlers here and the ones in the backoffice currently are doing basically the same thing, maybe we should make them behave in a similar way for consistency
func (h *Groups) WithdrawMembershipRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID, ok := uuidParam(ctx, w, r, "groupId")
	if !ok {
		return
	}

	if err := h.service.WithdrawMembershipRequest(ctx, groupID, actor.ID); err != nil {
		writeMembershipError(r, w, err, "withdraw membership request")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Groups) DecideMembership(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID, ok := uuidParam(ctx, w, r, "groupId")
	if !ok {
		return
	}
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid userId", nil)
		return
	}

	body, ok := decodeJSONBody(ctx, w, r)
	if !ok {
		return
	}

	var request api.MembershipDecisionRequest
	err := json.Unmarshal(body, &request)
	if err != nil || !request.Status.IsDecision() {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "status must be ACTIVE or REJECTED", err)
		return
	}

	membership, err := h.service.DecideMembership(ctx, groupID, userID, actor.ID, string(request.Status))
	if err != nil {
		writeMembershipError(r, w, err, "decide membership")
		return
	}

	writeJSON(w, http.StatusOK, GroupMembershipFromDomain(membership))
}

func GroupFromDomain(group domain.Group) api.Group {
	return api.Group{
		ID:          group.ID,
		SourceIRI:   group.SourceIRI,
		Name:        group.Name,
		Description: group.Description,
		Label:       group.Label,
		Logo:        group.Logo,
		Page:        group.Page,
		Color:       group.Color,
		Position:    group.Position,
		Concept:     group.Concept,
	}
}

func GroupMembershipFromDomain(membership domain.GroupMembership) api.GroupMembership {
	roles := make([]api.MembershipRole, 0, len(membership.Roles))
	for _, role := range membership.Roles {
		roles = append(roles, api.MembershipRole(role))
	}

	requests := make([]api.RoleRequest, 0, len(membership.RoleRequests))
	for _, request := range membership.RoleRequests {
		requests = append(requests, api.RoleRequest{
			ID:          request.ID,
			Role:        request.Role,
			Status:      request.Status,
			RequestedAt: request.RequestedAt.Format(time.RFC3339Nano),
		})
	}

	return api.GroupMembership{
		User: api.MembershipUser{
			ID:         membership.User.ID,
			GivenName:  membership.User.GivenName,
			FamilyName: membership.User.FamilyName,
			Email:      membership.User.Email,
		},
		Status:       api.MembershipStatus(membership.Status),
		Roles:        roles,
		Permissions:  append([]string{}, membership.Permissions...),
		RoleRequests: requests,
		CreatedAt:    membership.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:    membership.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func writeGroupError(r *http.Request, w http.ResponseWriter, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrBadRequest):
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid group id", err)
	case errors.Is(err, domain.ErrNotFound):
		WriteError(r.Context(), w, http.StatusNotFound, "not_found", "group not found", nil)
	case errors.Is(err, domain.ErrUpstreamTimeout):
		WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
	}
}

func writeMembershipError(r *http.Request, w http.ResponseWriter, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrBadRequest):
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid membership request", err)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "membership operation is not permitted", nil)
	case errors.Is(err, domain.ErrNotFound):
		WriteError(r.Context(), w, http.StatusNotFound, "not_found", "group or membership not found", nil)
	case errors.Is(err, domain.ErrConflict):
		WriteError(r.Context(), w, http.StatusConflict, "conflict", "membership request conflicts with existing membership", nil)
	case errors.Is(err, domain.ErrUpstreamTimeout):
		WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "membership request timed out", err)
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Error("failed to encode group response", "error", err)
	}
}
