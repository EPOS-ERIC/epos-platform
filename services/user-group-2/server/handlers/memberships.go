package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/db"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/db/sqlc"
)

type Memberships struct {
	db *db.PostgresDB
}

func NewMemberships(database *db.PostgresDB) *Memberships {
	return &Memberships{db: database}
}

type membershipResponse struct {
	UserID    string    `json:"userId"`
	GroupID   string    `json:"groupId"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type upsertMembershipRequest struct {
	Role   string `json:"role"`
	Status string `json:"status"`
}

func (h *Memberships) ListByGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	rows, err := h.db.ListMembershipsByGroupID(r.Context(), groupID)
	if err != nil {
		log.Error("list memberships failed", "group_id", groupID, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	memberships := make([]membershipResponse, len(rows))
	for i, row := range rows {
		memberships[i] = membershipResponse{
			UserID:    row.UserID.String(),
			GroupID:   row.GroupID.String(),
			Role:      row.Role,
			Status:    row.Status,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}

	writeJSONMemberships(w, http.StatusOK, memberships)
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

	if _, err := h.db.GetGroupByID(r.Context(), groupID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "group not found")
			return
		}
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}
	if _, err := h.db.GetUserByID(r.Context(), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	var req upsertMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Role = strings.ToUpper(strings.TrimSpace(req.Role))
	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
	if req.Role == "" || req.Status == "" {
		writeMessage(w, http.StatusBadRequest, "role and status are required")
		return
	}

	row, err := h.db.UpsertMembership(r.Context(), sqlc.UpsertMembershipParams{
		UserID:  userID,
		GroupID: groupID,
		Role:    req.Role,
		Status:  req.Status,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "membership not found")
			return
		}
		if isMembershipRoleError(err) {
			writeMessage(w, http.StatusBadRequest, "invalid role")
			return
		}
		if isMembershipStatusError(err) {
			writeMessage(w, http.StatusBadRequest, "invalid status")
			return
		}
		log.Error("upsert membership failed", "group_id", groupID, "user_id", userID, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONMembership(w, http.StatusOK, membershipResponse{
		UserID:    row.UserID.String(),
		GroupID:   row.GroupID.String(),
		Role:      row.Role,
		Status:    row.Status,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	})
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

	if _, err := h.db.GetMembershipByIDs(r.Context(), sqlc.GetMembershipByIDsParams{UserID: userID, GroupID: groupID}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "membership not found")
			return
		}
		log.Error("delete membership lookup failed", "group_id", groupID, "user_id", userID, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.db.DeleteMembership(r.Context(), sqlc.DeleteMembershipParams{UserID: userID, GroupID: groupID}); err != nil {
		log.Error("delete membership failed", "group_id", groupID, "user_id", userID, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSONMembership(w http.ResponseWriter, status int, body membershipResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("failed to encode membership response", "status", status, "error", err)
	}
}

func writeJSONMemberships(w http.ResponseWriter, status int, body []membershipResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("failed to encode memberships response", "status", status, "error", err)
	}
}

func isMembershipRoleError(err error) bool {
	return strings.Contains(err.Error(), "group_memberships_role_check")
}

func isMembershipStatusError(err error) bool {
	return strings.Contains(err.Error(), "group_memberships_status_check")
}
