package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/service"
)

type Memberships struct {
	service *service.Memberships
	log     *slog.Logger
}

func NewMemberships(svc *service.Memberships) *Memberships {
	return &Memberships{
		service: svc,
		log:     logging.Get("memberships-handler"),
	}
}

type upsertMembershipRequest struct {
	Role   service.MembershipRole   `json:"role"`
	Status service.MembershipStatus `json:"status"`
}

func (h *Memberships) ListByGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	items, err := h.service.ListByGroup(r.Context(), groupID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "list memberships failed", "group_id", groupID, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Memberships) Upsert(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := parseUUIDParam(r, "userId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	var req upsertMembershipRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WarnContext(r.Context(), "invalid upsert membership request body", "error", err)
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Upsert(r.Context(), service.UpsertMembershipParams{
		UserID:  userID,
		GroupID: groupID,
		Role:    req.Role,
		Status:  req.Status,
	})
	if err != nil {
		h.log.ErrorContext(r.Context(), "upsert membership failed", "group_id", groupID, "user_id", userID, "error", err)
		writeServiceError(w, err, "membership")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Memberships) Delete(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := parseUUIDParam(r, "userId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.Delete(r.Context(), userID, groupID); err != nil {
		h.log.ErrorContext(r.Context(), "delete membership failed", "group_id", groupID, "user_id", userID, "error", err)
		writeServiceError(w, err, "membership")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
