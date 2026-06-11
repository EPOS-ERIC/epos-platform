package handlers

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/service"
)

type Friendships struct {
	service *service.Friendships
}

func NewFriendships(svc *service.Friendships) *Friendships {
	return &Friendships{
		service: svc,
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
		writeServiceError(w, err, "friendship")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
