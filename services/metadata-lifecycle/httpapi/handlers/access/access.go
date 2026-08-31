// Package access exposes group membership and permission endpoints.
package access

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type service interface {
	ListMemberships(context.Context, domain.Actor, uuid.UUID) ([]domain.GroupMembership, error)
	RequestMembership(context.Context, domain.Actor, uuid.UUID, []domain.GroupRole) (domain.GroupMembership, error)
	WithdrawMembershipRequest(context.Context, domain.Actor, uuid.UUID) error
	DecideMembership(context.Context, domain.Actor, uuid.UUID, string, domain.MembershipStatus) (domain.GroupMembership, error)
	RevokeMembership(context.Context, domain.Actor, uuid.UUID, string) error
	RequestRole(context.Context, domain.Actor, uuid.UUID, domain.GroupRole) (domain.RoleRequest, error)
	ApplyRoleRequestDecision(context.Context, domain.Actor, uuid.UUID, uuid.UUID, domain.RoleRequestStatus) (domain.GroupMembership, error)
	WithdrawRoleRequest(context.Context, domain.Actor, uuid.UUID, uuid.UUID) error
	AssignRole(context.Context, domain.Actor, uuid.UUID, string, domain.GroupRole) error
	RemoveRole(context.Context, domain.Actor, uuid.UUID, string, domain.GroupRole) error
	GrantPermission(context.Context, domain.Actor, uuid.UUID, string, string) error
	RevokePermission(context.Context, domain.Actor, uuid.UUID, string, string) error
}

type Handler struct {
	service service
	log     *slog.Logger
}

func New(log *slog.Logger, service service) *Handler {
	return &Handler{
		service: service,
		log:     log.With("component", "access-handler"),
	}
}

func (h *Handler) ListMemberships(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(h.log, w, r)
	if !ok {
		return
	}

	memberships, err := h.service.ListMemberships(r.Context(), actorID, groupID)
	if err != nil {
		writeError(h.log, w, r, err, "list group memberships")
		return
	}

	result := make([]membershipResponse, 0, len(memberships))
	for _, membership := range memberships {
		result = append(result, toMembershipResponse(membership))
	}

	transport.WriteJSON(h.log, w, http.StatusOK, result)
}

func (h *Handler) RequestMembership(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(h.log, w, r)
	if !ok {
		return
	}

	var body membershipRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}
	if body.Roles == nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "roles is required")
		return
	}

	roles, err := toRoles(body)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid group access request")
		return
	}

	membership, err := h.service.RequestMembership(r.Context(), actorID, groupID, roles)
	if err != nil {
		writeError(h.log, w, r, err, "request group membership")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusCreated, toMembershipResponse(membership))
}

func (h *Handler) WithdrawMembershipRequest(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(h.log, w, r)
	if !ok {
		return
	}

	if err := h.service.WithdrawMembershipRequest(r.Context(), actorID, groupID); err != nil {
		writeError(h.log, w, r, err, "withdraw group membership request")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DecideMembership(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(h.log, w, r)
	if !ok {
		return
	}

	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return
	}
	userID = strings.TrimSpace(userID)

	var body membershipDecisionRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}
	if body.Status == "" {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "status is required")
		return
	}
	status, err := domain.NewMembershipStatus(body.Status)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid group access request")
		return
	}

	membership, err := h.service.DecideMembership(r.Context(), actorID, groupID, userID, status)
	if err != nil {
		writeError(h.log, w, r, err, "decide group membership")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusOK, toMembershipResponse(membership))
}

func (h *Handler) RevokeMembership(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(h.log, w, r)
	if !ok {
		return
	}

	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return
	}
	userID = strings.TrimSpace(userID)

	if err := h.service.RevokeMembership(r.Context(), actorID, groupID, userID); err != nil {
		writeError(h.log, w, r, err, "revoke group membership")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RequestRole(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(h.log, w, r)
	if !ok {
		return
	}

	var body roleRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}
	if body.Role == "" {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "role is required")
		return
	}
	role, err := domain.NewRequestedGroupRole(body.Role)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid group access request")
		return
	}

	request, err := h.service.RequestRole(r.Context(), actorID, groupID, role)
	if err != nil {
		writeError(h.log, w, r, err, "request group role")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusCreated, toRoleRequestResponse(request))
}

func (h *Handler) ApplyRoleRequestDecision(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(h.log, w, r)
	if !ok {
		return
	}

	requestID, err := uuid.Parse(chi.URLParam(r, "request_id"))
	if err != nil || requestID == uuid.Nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid role request id")
		return
	}

	var body roleRequestDecisionRequest
	if !transport.Decode(h.log, w, r, &body) {
		return
	}
	if body.Status == "" {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "status must be ACCEPTED or REJECTED")
		return
	}
	status, err := domain.NewRoleRequestStatus(body.Status)
	if err != nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "status must be ACCEPTED or REJECTED")
		return
	}

	membership, err := h.service.ApplyRoleRequestDecision(r.Context(), actorID, groupID, requestID, status)
	if err != nil {
		writeError(h.log, w, r, err, "decide group role request")
		return
	}

	transport.WriteJSON(h.log, w, http.StatusOK, toMembershipResponse(membership))
}

func (h *Handler) WithdrawRoleRequest(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(h.log, w, r)
	if !ok {
		return
	}

	requestID, err := uuid.Parse(chi.URLParam(r, "request_id"))
	if err != nil || requestID == uuid.Nil {
		transport.WriteProblem(h.log, w, http.StatusBadRequest, "invalid_request", "invalid role request id")
		return
	}

	if err := h.service.WithdrawRoleRequest(r.Context(), actorID, groupID, requestID); err != nil {
		writeError(h.log, w, r, err, "withdraw group role request")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, userID, role, ok := rolePath(h.log, w, r)
	if !ok {
		return
	}

	if err := h.service.AssignRole(r.Context(), actorID, groupID, userID, role); err != nil {
		writeError(h.log, w, r, err, "assign group role")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, userID, role, ok := rolePath(h.log, w, r)
	if !ok {
		return
	}

	if err := h.service.RemoveRole(r.Context(), actorID, groupID, userID, role); err != nil {
		writeError(h.log, w, r, err, "remove group role")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GrantPermission(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, userID, permission, ok := permissionPath(h.log, w, r)
	if !ok {
		return
	}

	if err := h.service.GrantPermission(r.Context(), actorID, groupID, userID, permission); err != nil {
		writeError(h.log, w, r, err, "grant group permission")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RevokePermission(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, userID, permission, ok := permissionPath(h.log, w, r)
	if !ok {
		return
	}

	if err := h.service.RevokePermission(r.Context(), actorID, groupID, userID, permission); err != nil {
		writeError(h.log, w, r, err, "revoke group permission")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func accessPath(log *slog.Logger, w http.ResponseWriter, r *http.Request) (domain.Actor, uuid.UUID, bool) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return "", uuid.Nil, false
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil || groupID == uuid.Nil {
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid group id")
		return "", uuid.Nil, false
	}

	return actorID, groupID, true
}

func rolePath(log *slog.Logger, w http.ResponseWriter, r *http.Request) (domain.Actor, uuid.UUID, string, domain.GroupRole, bool) {
	actorID, groupID, ok := accessPath(log, w, r)
	if !ok {
		return "", uuid.Nil, "", "", false
	}

	userID := chi.URLParam(r, "user_id")
	rawRole := chi.URLParam(r, "role")
	if userID == "" || rawRole == "" {
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid user id or role")
		return "", uuid.Nil, "", "", false
	}
	userID = strings.TrimSpace(userID)
	role, err := domain.NewGroupRole(rawRole)
	if err != nil {
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid group access request")
		return "", uuid.Nil, "", "", false
	}

	return actorID, groupID, userID, role, true
}

func permissionPath(log *slog.Logger, w http.ResponseWriter, r *http.Request) (domain.Actor, uuid.UUID, string, string, bool) {
	actorID, groupID, ok := accessPath(log, w, r)
	if !ok {
		return "", uuid.Nil, "", "", false
	}

	userID := chi.URLParam(r, "user_id")
	permission := chi.URLParam(r, "permission")
	if userID == "" || permission == "" {
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid user id or permission")
		return "", uuid.Nil, "", "", false
	}
	userID = strings.TrimSpace(userID)
	permission = strings.TrimSpace(permission)

	return actorID, groupID, userID, permission, true
}

func writeError(log *slog.Logger, w http.ResponseWriter, r *http.Request, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrInvalidAccess):
		transport.WriteProblem(log, w, http.StatusBadRequest, "invalid_request", "invalid group access request")
	case errors.Is(err, domain.ErrAccessForbidden):
		transport.WriteProblem(log, w, http.StatusForbidden, "forbidden", "group access operation is not permitted")
	case errors.Is(err, domain.ErrAccessNotFound):
		transport.WriteProblem(log, w, http.StatusNotFound, "not_found", "group or user not found")
	case errors.Is(err, domain.ErrAccessConflict):
		transport.WriteProblem(log, w, http.StatusConflict, "conflict", "group membership request conflicts with existing membership")
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		transport.WriteProblem(log, w, http.StatusInternalServerError, "internal_server_error", operation+" failed")
	}
}
