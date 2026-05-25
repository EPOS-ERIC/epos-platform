package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
)

type MembershipsHandler struct {
	usersRepo       repository.UserRepository
	groupsRepo      repository.GroupRepository
	membershipsRepo repository.MembershipRepository
}

func NewMembershipsHandler(usersRepo repository.UserRepository, groupsRepo repository.GroupRepository, membershipsRepo repository.MembershipRepository) *MembershipsHandler {
	return &MembershipsHandler{usersRepo: usersRepo, groupsRepo: groupsRepo, membershipsRepo: membershipsRepo}
}

type upsertMembershipRequest struct {
	Role   string `json:"role"`
	Status string `json:"status"`
}

func (h *MembershipsHandler) ListByGroup(w http.ResponseWriter, r *http.Request) {
	groupID := strings.TrimSpace(r.PathValue("groupId"))
	if groupID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "groupId is required"})
		return
	}

	writeJSON(w, http.StatusOK, h.membershipsRepo.ListByGroupID(groupID))
}

func (h *MembershipsHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	groupID := strings.TrimSpace(r.PathValue("groupId"))
	userID := strings.TrimSpace(r.PathValue("userId"))
	if groupID == "" || userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "groupId and userId are required"})
		return
	}

	if _, err := h.groupsRepo.GetByID(groupID); err != nil {
		if errors.Is(err, repository.ErrGroupNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "group not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}
	if _, err := h.usersRepo.GetByID(userID); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	var req upsertMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}
	req.Role = strings.ToUpper(strings.TrimSpace(req.Role))
	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
	if req.Role == "" || req.Status == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "role and status are required"})
		return
	}

	m, err := h.membershipsRepo.Upsert(userID, groupID, req.Role, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrMembershipInvalidRole):
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid role"})
		case errors.Is(err, repository.ErrMembershipInvalidStatus):
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid status"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, m)
}

func (h *MembershipsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	groupID := strings.TrimSpace(r.PathValue("groupId"))
	userID := strings.TrimSpace(r.PathValue("userId"))
	if groupID == "" || userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "groupId and userId are required"})
		return
	}

	err := h.membershipsRepo.Delete(userID, groupID)
	if err != nil {
		if errors.Is(err, repository.ErrMembershipNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "membership not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
