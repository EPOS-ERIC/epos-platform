package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/db"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/db/sqlc"
	"github.com/google/uuid"
)

type Authorize struct {
	db *db.PostgresDB
}

func NewAuthorize(database *db.PostgresDB) *Authorize {
	return &Authorize{db: database}
}

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

func (h *Authorize) Authorize(w http.ResponseWriter, r *http.Request) {
	var req AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.AuthIdentifier = strings.TrimSpace(req.AuthIdentifier)
	req.Group = strings.TrimSpace(req.Group)
	req.Service = strings.TrimSpace(req.Service)
	req.Action = strings.TrimSpace(req.Action)
	if req.AuthIdentifier == "" || req.Group == "" || req.Service == "" || req.Action == "" {
		writeMessage(w, http.StatusBadRequest, "authIdentifier, group, service and action are required")
		return
	}

	allowed, reason, err := h.decide(r, req)
	if err != nil {
		log.Error("failed to evaluate authorization", "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONAuthorize(w, http.StatusOK, AuthorizeResponse{Allowed: allowed, Reason: reason})
}

func (h *Authorize) decide(r *http.Request, req AuthorizeRequest) (bool, string, error) {
	groupID, err := uuid.Parse(req.Group)
	if err != nil {
		return false, "group is invalid", nil
	}

	requester, err := h.db.GetUserByAuthIdentifier(r.Context(), req.AuthIdentifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, "requesting user not found", nil
		}
		return false, "", fmt.Errorf("get requester by auth identifier: %w", err)
	}

	isRequesterMember, err := h.db.IsActiveMember(r.Context(), sqlc.IsActiveMemberParams{UserID: requester.ID, GroupID: groupID})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, "", fmt.Errorf("check requester membership: %w", err)
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

	owner, err := h.db.GetUserByAuthIdentifier(r.Context(), ownerIdentifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, "draft owner not found", nil
		}
		return false, "", fmt.Errorf("get owner by auth identifier: %w", err)
	}

	isOwnerMember, err := h.db.IsActiveMember(r.Context(), sqlc.IsActiveMemberParams{UserID: owner.ID, GroupID: groupID})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, "", fmt.Errorf("check owner membership: %w", err)
	}
	if !isOwnerMember {
		return false, "draft owner is not an active group member", nil
	}

	areFriends, err := h.db.AreFriends(r.Context(), sqlc.AreFriendsParams{UserID: requester.ID, FriendID: owner.ID})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, "", fmt.Errorf("check friendship: %w", err)
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

func writeJSONAuthorize(w http.ResponseWriter, status int, body AuthorizeResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("failed to encode authorize response", "status", status, "error", err)
	}
}
