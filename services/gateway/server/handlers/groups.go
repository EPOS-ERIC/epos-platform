package handlers

import (
	"context"
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
	groupID, ok := uuidParam(r.Context(), w, r, "groupId")
	if !ok {
		return
	}

	group, err := h.service.Get(r.Context(), groupID)
	if err != nil {
		writeGroupError(r, w, err, "get group")
		return
	}

	writeJSON(w, http.StatusOK, GroupFromDomain(group))
}

func (h *Groups) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}
	var request api.CreateGroupRequest
	if !decodeJSON(r, w, &request) {
		return
	}
	group, err := h.service.Create(ctx, actor.ID, domain.CreateGroupInput{
		SourceIRI: request.SourceIRI, Name: request.Name, Description: request.Description,
		Label: request.Label, Logo: request.Logo, Page: request.Page, Color: request.Color,
		Position: request.Position, Concept: request.Concept,
	})
	if err != nil {
		writeGroupError(r, w, err, "create group")
		return
	}
	writeJSON(w, http.StatusCreated, GroupFromDomain(group))
}

func (h *Groups) Replace(w http.ResponseWriter, r *http.Request) {
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
	var request api.ReplaceGroupRequest
	if !decodeJSON(r, w, &request) {
		return
	}
	group, err := h.service.Replace(ctx, actor.ID, groupID, domain.ReplaceGroupInput{
		Name: request.Name, Description: request.Description, Label: request.Label,
		Logo: request.Logo, Page: request.Page, Color: request.Color,
		Position: request.Position, Concept: request.Concept,
	})
	if err != nil {
		writeGroupError(r, w, err, "replace group")
		return
	}
	writeJSON(w, http.StatusOK, GroupFromDomain(group))
}

func (h *Groups) Delete(w http.ResponseWriter, r *http.Request) {
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
	if err := h.service.Delete(ctx, actor.ID, groupID); err != nil {
		writeGroupError(r, w, err, "delete group")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Groups) ListMemberships(w http.ResponseWriter, r *http.Request) {
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
	memberships, err := h.service.ListMemberships(ctx, groupID, actor.ID)
	if err != nil {
		writeMembershipError(r, w, err, "list group memberships")
		return
	}
	result := make([]api.GroupMembership, 0, len(memberships))
	for _, membership := range memberships {
		result = append(result, GroupMembershipFromDomain(membership))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Groups) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	groupID, ok := uuidParam(ctx, w, r, "groupId")
	if !ok {
		return
	}
	categories, err := h.service.ListCategories(ctx, groupID)
	if err != nil {
		writeGroupError(r, w, err, "list categories")
		return
	}
	result := make([]api.Category, 0, len(categories))
	for _, category := range categories {
		result = append(result, CategoryFromDomain(category))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Groups) GetCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	groupID, ok := uuidParam(ctx, w, r, "groupId")
	if !ok {
		return
	}
	categoryID, ok := uuidParam(ctx, w, r, "categoryId")
	if !ok {
		return
	}
	category, err := h.service.GetCategory(ctx, groupID, categoryID)
	if err != nil {
		writeGroupError(r, w, err, "get category")
		return
	}
	writeJSON(w, http.StatusOK, CategoryFromDomain(category))
}

func (h *Groups) CreateCategory(w http.ResponseWriter, r *http.Request) {
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
	var request api.CreateCategoryRequest
	if !decodeJSON(r, w, &request) {
		return
	}
	category, err := h.service.CreateCategory(ctx, actor.ID, groupID, domain.CreateCategoryInput{
		SourceIRI: request.SourceIRI, Name: request.Name, ParentCategoryID: request.ParentCategoryID,
	})
	if err != nil {
		writeGroupError(r, w, err, "create category")
		return
	}
	writeJSON(w, http.StatusCreated, CategoryFromDomain(category))
}

func (h *Groups) ReplaceCategory(w http.ResponseWriter, r *http.Request) {
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
	categoryID, ok := uuidParam(ctx, w, r, "categoryId")
	if !ok {
		return
	}
	var request api.ReplaceCategoryRequest
	if !decodeJSON(r, w, &request) {
		return
	}
	category, err := h.service.ReplaceCategory(ctx, actor.ID, groupID, categoryID, domain.ReplaceCategoryInput{
		Name: request.Name, ParentCategoryID: request.ParentCategoryID,
	})
	if err != nil {
		writeGroupError(r, w, err, "replace category")
		return
	}
	writeJSON(w, http.StatusOK, CategoryFromDomain(category))
}

func (h *Groups) DeleteCategory(w http.ResponseWriter, r *http.Request) {
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
	categoryID, ok := uuidParam(ctx, w, r, "categoryId")
	if !ok {
		return
	}
	if err := h.service.DeleteCategory(ctx, actor.ID, groupID, categoryID); err != nil {
		writeGroupError(r, w, err, "delete category")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// WithdrawMembershipRequest deletes the authenticated user's pending membership request.
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

func (h *Groups) RequestMembership(w http.ResponseWriter, r *http.Request) {
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

	var request api.CreateMembershipRequest
	if !decodeJSON(r, w, &request) || request.Roles == nil {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "roles is required", nil)
		return
	}

	membership, err := h.service.RequestMembership(ctx, groupID, actor.ID, request.Roles)
	if err != nil {
		writeMembershipError(r, w, err, "request membership")
		return
	}

	writeJSON(w, http.StatusCreated, GroupMembershipFromDomain(membership))
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
	userID, ok := uuidParam(ctx, w, r, "userId")
	if !ok {
		return
	}

	var request api.MembershipDecisionRequest
	if !decodeJSON(r, w, &request) || !request.Status.IsDecision() {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "status must be ACTIVE or REJECTED", nil)
		return
	}

	membership, err := h.service.DecideMembership(ctx, groupID, userID, actor.ID, string(request.Status))
	if err != nil {
		writeMembershipError(r, w, err, "decide membership")
		return
	}

	writeJSON(w, http.StatusOK, GroupMembershipFromDomain(membership))
}

func (h *Groups) DeleteMembership(w http.ResponseWriter, r *http.Request) {
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

	userID, ok := uuidParam(ctx, w, r, "userId")
	if !ok {
		return
	}

	if err := h.service.RevokeMembership(ctx, groupID, userID, actor.ID); err != nil {
		writeMembershipError(r, w, err, "revoke membership")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Groups) AssignGroupRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID, userID, role, ok := groupRoleParams(ctx, w, r)
	if !ok {
		return
	}

	if err := h.service.AssignGroupRole(ctx, groupID, userID, role, actor.ID); err != nil {
		writeMembershipError(r, w, err, "assign group role")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Groups) RemoveGroupRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID, userID, role, ok := groupRoleParams(ctx, w, r)
	if !ok {
		return
	}

	if err := h.service.RemoveGroupRole(ctx, groupID, userID, role, actor.ID); err != nil {
		writeMembershipError(r, w, err, "remove group role")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Groups) GrantGroupPermission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID, userID, permission, ok := groupPermissionParams(ctx, w, r)
	if !ok {
		return
	}

	if err := h.service.GrantGroupPermission(ctx, groupID, userID, permission, actor.ID); err != nil {
		writeMembershipError(r, w, err, "grant group permission")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Groups) RevokeGroupPermission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	groupID, userID, permission, ok := groupPermissionParams(ctx, w, r)
	if !ok {
		return
	}

	if err := h.service.RevokeGroupPermission(ctx, groupID, userID, permission, actor.ID); err != nil {
		writeMembershipError(r, w, err, "revoke group permission")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Groups) RequestRole(w http.ResponseWriter, r *http.Request) {
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

	var request api.CreateRoleRequest
	// TODO: use an enum
	if !decodeJSON(r, w, &request) || (request.Role != "EDITOR" && request.Role != "REVIEWER") {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "role must be EDITOR or REVIEWER", nil)
		return
	}

	roleRequest, err := h.service.RequestRole(ctx, groupID, request.Role, actor.ID)
	if err != nil {
		writeMembershipError(r, w, err, "request role")
		return
	}

	writeJSON(w, http.StatusCreated, api.RoleRequest{
		ID:          roleRequest.ID,
		Role:        roleRequest.Role,
		Status:      roleRequest.Status,
		RequestedAt: roleRequest.RequestedAt.Format(time.RFC3339Nano),
	})
}

func (h *Groups) ApplyRoleRequestDecision(w http.ResponseWriter, r *http.Request) {
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
	requestID, ok := uuidParam(ctx, w, r, "requestId")
	if !ok {
		return
	}

	var request api.RoleRequestDecisionRequest
	if !decodeJSON(r, w, &request) || (request.Status != "ACCEPTED" && request.Status != "REJECTED") {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "status must be ACCEPTED or REJECTED", nil)
		return
	}

	membership, err := h.service.ApplyRoleRequestDecision(ctx, groupID, requestID, request.Status, actor.ID)
	if err != nil {
		writeMembershipError(r, w, err, "decide role request")
		return
	}

	writeJSON(w, http.StatusOK, GroupMembershipFromDomain(membership))
}

func (h *Groups) WithdrawRoleRequest(w http.ResponseWriter, r *http.Request) {
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
	requestID, ok := uuidParam(ctx, w, r, "requestId")
	if !ok {
		return
	}

	if err := h.service.WithdrawRoleRequest(ctx, groupID, requestID, actor.ID); err != nil {
		writeMembershipError(r, w, err, "withdraw role request")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func CategoryFromDomain(category domain.Category) api.Category {
	return api.Category{ID: category.ID, GroupID: category.GroupID, ParentCategoryID: category.ParentCategoryID, SourceIRI: category.SourceIRI, Name: category.Name}
}

func groupRoleParams(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, string, string, bool) {
	groupID, ok := uuidParam(ctx, w, r, "groupId")
	if !ok {
		return "", "", "", false
	}

	userID := chi.URLParam(r, "userId")
	role := api.MembershipRole(chi.URLParam(r, "role"))
	if !validUUID(userID) || !role.IsValid() {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid userId or role", nil)
		return "", "", "", false
	}

	return groupID, userID, string(role), true
}

func groupPermissionParams(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, string, string, bool) {
	groupID, ok := uuidParam(ctx, w, r, "groupId")
	if !ok {
		return "", "", "", false
	}

	userID := chi.URLParam(r, "userId")
	permission := chi.URLParam(r, "permission")
	if !validUUID(userID) || !validGroupPermission(permission) {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid userId or permission", nil)
		return "", "", "", false
	}

	return groupID, userID, permission, true
}

func validGroupPermission(permission string) bool {
	switch permission {
	case "resource.create", "resource.edit", "resource.submit", "resource.withdraw", "resource.reopen", "resource.discard", "resource.reject", "resource.publish", "resource.unpublish", "group.membership.manage", "group.role.manage", "group.permission.manage", "category.manage":
		return true
	default:
		return false
	}
}

//nolint:dupl // Error mapping intentionally mirrors other gateway handler boundaries.
func writeGroupError(r *http.Request, w http.ResponseWriter, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrBadRequest):
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid group id", err)
	case errors.Is(err, domain.ErrNotFound):
		WriteError(r.Context(), w, http.StatusNotFound, "not_found", "group not found", nil)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "group operation is not permitted", nil)
	case errors.Is(err, domain.ErrConflict):
		WriteError(r.Context(), w, http.StatusConflict, "conflict", "group operation conflicts with existing data", nil)
	case errors.Is(err, domain.ErrUpstreamTimeout):
		WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "group request timed out", err)
	default:
		log.ErrorContext(r.Context(), operation+" failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
	}
}

//nolint:dupl // Error mapping intentionally mirrors other gateway handler boundaries.
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
