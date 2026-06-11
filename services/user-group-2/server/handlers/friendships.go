package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/db"
	"github.com/EPOS-ERIC/epos-platform/services/user-group/db/sqlc"
)

type Friendships struct {
	db *db.PostgresDB
}

func NewFriendships(database *db.PostgresDB) *Friendships {
	return &Friendships{db: database}
}

type friendshipResponse struct {
	UserID    string    `json:"userId"`
	FriendID  string    `json:"friendId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *Friendships) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUUIDParam(r, "userId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	friendID, err := parseUUIDParam(r, "friendUserId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	if userID == friendID {
		writeMessage(w, http.StatusBadRequest, "invalid friendship")
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
	if _, err := h.db.GetUserByID(r.Context(), friendID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "friend user not found")
			return
		}
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.db.CreateFriendship(r.Context(), sqlc.CreateFriendshipParams{
		UserID:     userID,
		FriendID:   friendID,
		UserID_2:   friendID,
		FriendID_2: userID,
	}); err != nil {
		log.Error("create friendship failed", "user_id", userID, "friend_id", friendID, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	row, err := h.db.GetFriendship(r.Context(), sqlc.GetFriendshipParams{UserID: userID, FriendID: friendID})
	if err != nil {
		log.Error("get friendship after create failed", "user_id", userID, "friend_id", friendID, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONFriendship(w, http.StatusCreated, friendshipResponse{
		UserID:    row.UserID.String(),
		FriendID:  row.FriendID.String(),
		CreatedAt: row.CreatedAt,
	})
}

func (h *Friendships) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUUIDParam(r, "userId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	friendID, err := parseUUIDParam(r, "friendUserId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.db.GetFriendship(r.Context(), sqlc.GetFriendshipParams{UserID: userID, FriendID: friendID}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "friendship not found")
			return
		}
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.db.DeleteFriendship(r.Context(), sqlc.DeleteFriendshipParams{UserID: userID, FriendID: friendID, UserID_2: friendID, FriendID_2: userID}); err != nil {
		log.Error("delete friendship failed", "user_id", userID, "friend_id", friendID, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSONFriendship(w http.ResponseWriter, status int, body friendshipResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("failed to encode friendship response", "status", status, "error", err)
	}
}
