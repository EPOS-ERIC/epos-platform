package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
)

type AuthorizeRequest struct {
	AuthIdentifier          string `json:"authIdentifier"`
	Group                   string `json:"group"`
	Service                 string `json:"service"`
	Action                  string `json:"action"`
	ResourceType            string `json:"resourceType,omitempty"`
	ResourceID              string `json:"resourceId,omitempty"`
	ResourceOwnerIdentifier string `json:"resourceOwnerIdentifier,omitempty"`
}

type AuthorizeResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

type AuthorizeHandler struct {
	usersRepo       repository.UserRepository
	membershipsRepo repository.MembershipRepository
	friendshipsRepo repository.FriendshipRepository
}

func NewAuthorizeHandler(usersRepo repository.UserRepository, membershipsRepo repository.MembershipRepository, friendshipsRepo repository.FriendshipRepository) *AuthorizeHandler {
	return &AuthorizeHandler{usersRepo: usersRepo, membershipsRepo: membershipsRepo, friendshipsRepo: friendshipsRepo}
}

func (h *AuthorizeHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	var req AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid authorize request body", "error", err)
		writeJSON(w, http.StatusBadRequest, AuthorizeResponse{Allowed: false, Reason: "invalid request body"})
		return
	}

	if strings.TrimSpace(req.AuthIdentifier) == "" || strings.TrimSpace(req.Group) == "" || strings.TrimSpace(req.Service) == "" || strings.TrimSpace(req.Action) == "" {
		log.Warn("missing required authorize fields")
		writeJSON(w, http.StatusBadRequest, AuthorizeResponse{Allowed: false, Reason: "authIdentifier, group, service and action are required"})
		return
	}

	allowed, reason, err := h.decide(req)
	if err != nil {
		log.Error("failed to evaluate authorization", "error", err)
		writeJSON(w, http.StatusInternalServerError, AuthorizeResponse{Allowed: false, Reason: "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, AuthorizeResponse{Allowed: allowed, Reason: reason})
}

func (h *AuthorizeHandler) decide(req AuthorizeRequest) (bool, string, error) {
	requester, err := h.usersRepo.GetByAuthIdentifier(req.AuthIdentifier)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return false, "requesting user not found", nil
		}
		return false, "", err
	}

	isRequesterMember, err := h.membershipsRepo.IsActiveMember(requester.ID, req.Group)
	if err != nil {
		return false, "", err
	}
	if !isRequesterMember {
		return false, "requesting user is not an active group member", nil
	}

	if !isDraftReadRequest(req) {
		return false, "rule not applicable", nil
	}

	ownerIdentifier := strings.TrimSpace(req.ResourceOwnerIdentifier)
	if ownerIdentifier == "" {
		return false, "resourceOwnerIdentifier is required for draft access", nil
	}
	if ownerIdentifier == requester.AuthIdentifier {
		return true, "same owner", nil
	}

	owner, err := h.usersRepo.GetByAuthIdentifier(ownerIdentifier)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return false, "draft owner not found", nil
		}
		return false, "", err
	}

	isOwnerMember, err := h.membershipsRepo.IsActiveMember(owner.ID, req.Group)
	if err != nil {
		return false, "", err
	}
	if !isOwnerMember {
		return false, "draft owner is not an active group member", nil
	}

	areFriends, err := h.friendshipsRepo.AreFriends(requester.ID, owner.ID)
	if err != nil {
		return false, "", err
	}
	if !areFriends {
		return false, "users are not friends", nil
	}

	return true, "friends in same group can access draft", nil
}

func isDraftReadRequest(req AuthorizeRequest) bool {
	service := strings.ToLower(strings.TrimSpace(req.Service))
	action := strings.ToLower(strings.TrimSpace(req.Action))
	resourceType := strings.ToLower(strings.TrimSpace(req.ResourceType))

	if service != "metadata-lifecycle" {
		return false
	}
    if action != "read" && action != "view" && action != "get" {
		return false
	}

	return strings.Contains(resourceType, "draft") || resourceType == ""
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("failed to encode JSON response", "status", status, "error", err)
	}
}
