package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
)

type FriendshipsHandler struct {
	usersRepo       repository.UserRepository
	friendshipsRepo repository.FriendshipRepository
}

func NewFriendshipsHandler(usersRepo repository.UserRepository, friendshipsRepo repository.FriendshipRepository) *FriendshipsHandler {
	return &FriendshipsHandler{usersRepo: usersRepo, friendshipsRepo: friendshipsRepo}
}

func (h *FriendshipsHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.PathValue("userId"))
	friendID := strings.TrimSpace(r.PathValue("friendUserId"))
	if userID == "" || friendID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "userId and friendUserId are required"})
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
	if _, err := h.usersRepo.GetByID(friendID); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "friend user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	f, err := h.friendshipsRepo.Create(userID, friendID)
	if err != nil {
		if errors.Is(err, repository.ErrFriendshipInvalid) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid friendship"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, f)
}

func (h *FriendshipsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.PathValue("userId"))
	friendID := strings.TrimSpace(r.PathValue("friendUserId"))
	if userID == "" || friendID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "userId and friendUserId are required"})
		return
	}

	err := h.friendshipsRepo.Delete(userID, friendID)
	if err != nil {
		if errors.Is(err, repository.ErrFriendshipNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "friendship not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
