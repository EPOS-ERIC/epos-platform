package groups

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/groups"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/transport"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type accessService interface {
	ListMemberships(context.Context, string, uuid.UUID) ([]groups.Membership, error)
	RequestMembership(context.Context, string, uuid.UUID, groups.MembershipRequest) (groups.Membership, error)
	WithdrawMembershipRequest(context.Context, string, uuid.UUID) error
	DecideMembership(context.Context, string, uuid.UUID, string, string) (groups.Membership, error)
	RevokeMembership(context.Context, string, uuid.UUID, string) error
	RequestRole(context.Context, string, uuid.UUID, string) (groups.RoleRequest, error)
	ApplyRoleRequestDecision(context.Context, string, uuid.UUID, uuid.UUID, string) (groups.Membership, error)
	WithdrawRoleRequest(context.Context, string, uuid.UUID, uuid.UUID) error
}

type AccessHandler struct {
	service accessService
}

func NewAccess(service accessService) *AccessHandler {
	return &AccessHandler{service: service}
}

func (h *AccessHandler) ListMemberships(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(w, r)
	if !ok {
		return
	}

	memberships, err := h.service.ListMemberships(r.Context(), actorID, groupID)
	if err != nil {
		writeAccessError(w, r, err, "list group memberships")
		return
	}

	// TODO: see if maybe we can create a package with utils functions like Map or something
	result := make([]membershipResponse, 0, len(memberships))
	for _, membership := range memberships {
		result = append(result, toMembershipResponse(membership))
	}

	transport.WriteJSON(w, http.StatusOK, result)
}

func (h *AccessHandler) RequestMembership(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(w, r)
	if !ok {
		return
	}

	var body struct {
		Roles *[]string `json:"roles"`
	}
	if !transport.Decode(w, r, &body) {
		return
	}
	if body.Roles == nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "roles is required")
		return
	}

	membership, err := h.service.RequestMembership(r.Context(), actorID, groupID, groups.MembershipRequest{Roles: *body.Roles})
	if err != nil {
		writeAccessError(w, r, err, "request group membership")
		return
	}

	transport.WriteJSON(w, http.StatusCreated, toMembershipResponse(membership))
}

func (h *AccessHandler) WithdrawMembershipRequest(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(w, r)
	if !ok {
		return
	}

	if err := h.service.WithdrawMembershipRequest(r.Context(), actorID, groupID); err != nil {
		writeAccessError(w, r, err, "withdraw group membership request")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AccessHandler) DecideMembership(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(w, r)
	if !ok {
		return
	}

	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return
	}

	var body struct {
		Status *string `json:"status"`
	}
	if !transport.Decode(w, r, &body) {
		return
	}
	if body.Status == nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "status is required")
		return
	}

	membership, err := h.service.DecideMembership(r.Context(), actorID, groupID, userID, *body.Status)
	if err != nil {
		writeAccessError(w, r, err, "decide group membership")
		return
	}

	transport.WriteJSON(w, http.StatusOK, toMembershipResponse(membership))
}

func (h *AccessHandler) RevokeMembership(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(w, r)
	if !ok {
		return
	}

	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return
	}

	if err := h.service.RevokeMembership(r.Context(), actorID, groupID, userID); err != nil {
		writeAccessError(w, r, err, "revoke group membership")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AccessHandler) RequestRole(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(w, r)
	if !ok {
		return
	}

	var body struct {
		Role *string `json:"role"`
	}
	if !transport.Decode(w, r, &body) {
		return
	}
	if body.Role == nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "role is required")
		return
	}

	request, err := h.service.RequestRole(r.Context(), actorID, groupID, *body.Role)
	if err != nil {
		writeAccessError(w, r, err, "request group role")
		return
	}

	transport.WriteJSON(w, http.StatusCreated, roleRequestResponse{
		ID:          request.ID,
		Role:        request.Role,
		Status:      request.Status,
		RequestedAt: request.RequestedAt,
	})
}

func (h *AccessHandler) ApplyRoleRequestDecision(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(w, r)
	if !ok {
		return
	}

	requestID, err := uuid.Parse(chi.URLParam(r, "request_id"))
	if err != nil || requestID == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid role request id")
		return
	}

	var body struct {
		Status *string `json:"status"`
	}
	if !transport.Decode(w, r, &body) {
		return
	}
	if body.Status == nil || (*body.Status != "ACCEPTED" && *body.Status != "REJECTED") {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "status must be ACCEPTED or REJECTED")
		return
	}

	membership, err := h.service.ApplyRoleRequestDecision(r.Context(), actorID, groupID, requestID, *body.Status)
	if err != nil {
		writeAccessError(w, r, err, "decide group role request")
		return
	}

	transport.WriteJSON(w, http.StatusOK, toMembershipResponse(membership))
}

func (h *AccessHandler) WithdrawRoleRequest(w http.ResponseWriter, r *http.Request) {
	actorID, groupID, ok := accessPath(w, r)
	if !ok {
		return
	}

	requestID, err := uuid.Parse(chi.URLParam(r, "request_id"))
	if err != nil || requestID == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid role request id")
		return
	}

	if err := h.service.WithdrawRoleRequest(r.Context(), actorID, groupID, requestID); err != nil {
		writeAccessError(w, r, err, "withdraw group role request")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func accessPath(w http.ResponseWriter, r *http.Request) (string, uuid.UUID, bool) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return "", uuid.Nil, false
	}

	groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil || groupID == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group id")
		return "", uuid.Nil, false
	}

	return actorID, groupID, true
}

func writeAccessError(w http.ResponseWriter, r *http.Request, err error, operation string) {
	switch {
	case errors.Is(err, groups.ErrInvalid):
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group access request")
	case errors.Is(err, groups.ErrForbidden):
		transport.WriteProblem(w, http.StatusForbidden, "forbidden", "group access operation is not permitted")
	case errors.Is(err, groups.ErrNotFound):
		transport.WriteProblem(w, http.StatusNotFound, "not_found", "group or user not found")
	case errors.Is(err, groups.ErrConflict):
		transport.WriteProblem(w, http.StatusConflict, "conflict", "group membership request conflicts with existing membership")
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", operation+" failed")
	}
}

type membershipResponse struct {
	User         membershipUserResponse `json:"user"`
	Status       string                 `json:"status"`
	Roles        []string               `json:"roles"`
	Permissions  []string               `json:"permissions"`
	RoleRequests []roleRequestResponse  `json:"role_requests"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type membershipUserResponse struct {
	ID         string `json:"id"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Email      string `json:"email"`
}

type roleRequestResponse struct {
	ID          uuid.UUID `json:"id"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	RequestedAt time.Time `json:"requested_at"`
}

func toMembershipResponse(membership groups.Membership) membershipResponse {
	roles := append([]string{}, membership.Roles...)
	permissions := append([]string{}, membership.Permissions...)
	requests := make([]roleRequestResponse, 0, len(membership.RoleRequests))
	for _, request := range membership.RoleRequests {
		requests = append(requests, roleRequestResponse{
			ID:          request.ID,
			Role:        request.Role,
			Status:      request.Status,
			RequestedAt: request.RequestedAt,
		})
	}
	return membershipResponse{
		User: membershipUserResponse{
			ID:         membership.User.ID,
			GivenName:  membership.User.GivenName,
			FamilyName: membership.User.FamilyName,
			Email:      membership.User.Email,
		},
		Status:       membership.Status,
		Roles:        roles,
		Permissions:  permissions,
		RoleRequests: requests,
		CreatedAt:    membership.CreatedAt,
		UpdatedAt:    membership.UpdatedAt,
	}
}
