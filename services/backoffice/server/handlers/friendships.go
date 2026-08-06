package handlers

import (
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/service"
)

type Friendships struct {
	service *service.Friendships
	log     *slog.Logger
}

func NewFriendships(svc *service.Friendships) *Friendships {
	return &Friendships{
		service: svc,
		log:     logging.Get("friendships-handler"),
	}
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

	item, err := h.service.Create(r.Context(), userID, friendID)
	if err != nil {
		h.log.ErrorContext(r.Context(), "create friendship failed", "user_id", userID, "friend_user_id", friendID, "error", err)
		writeServiceError(w, err, "friendship")
		return
	}

	writeJSON(w, http.StatusCreated, item)
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

	if err := h.service.Delete(r.Context(), userID, friendID); err != nil {
		h.log.ErrorContext(r.Context(), "delete friendship failed", "user_id", userID, "friend_user_id", friendID, "error", err)
		writeServiceError(w, err, "friendship")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
