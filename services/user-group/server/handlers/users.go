package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
)

type UsersHandler struct {
	repo repository.UserRepository
}

func NewUsersHandler(repo repository.UserRepository) *UsersHandler {
	return &UsersHandler{repo: repo}
}

type createUserRequest struct {
	AuthIdentifier string `json:"authIdentifier"`
	Email          string `json:"email"`
	GivenName      string `json:"givenName"`
	FamilyName     string `json:"familyName"`
	Status         string `json:"status"`
}

type patchUserRequest struct {
	AuthIdentifier *string `json:"authIdentifier"`
	Email          *string `json:"email"`
	GivenName      *string `json:"givenName"`
	FamilyName     *string `json:"familyName"`
	Status         *string `json:"status"`
}

func (h *UsersHandler) List(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.repo.List())
}

func (h *UsersHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("userId")
	user, err := h.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			log.Warn("user not found", "user_id", id)
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "user not found"})
			return
		}
		log.Error("failed to get user by id", "user_id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *UsersHandler) GetByAuthIdentifier(w http.ResponseWriter, r *http.Request) {
	authIdentifier := strings.TrimSpace(r.PathValue("authIdentifier"))
	if authIdentifier == "" {
		log.Warn("missing auth identifier")
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "authIdentifier is required"})
		return
	}

	user, err := h.repo.GetByAuthIdentifier(authIdentifier)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			log.Warn("user not found by auth identifier", "auth_identifier", authIdentifier)
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "user not found"})
			return
		}
		log.Error("failed to get user by auth identifier", "auth_identifier", authIdentifier, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid create user request body", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	normalizeCreateUserRequest(&req)
	if req.AuthIdentifier == "" {
		log.Warn("missing auth identifier in create user request")
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "authIdentifier is required"})
		return
	}
	if req.Status == "" {
		req.Status = "ACTIVE"
	}

	user, err := h.repo.Create(req.AuthIdentifier, req.Email, req.GivenName, req.FamilyName, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserAuthIDExists):
			log.Warn("create user conflict", "auth_identifier", req.AuthIdentifier)
			writeJSON(w, http.StatusConflict, map[string]string{"message": "user auth identifier already exists"})
		case errors.Is(err, repository.ErrUserInvalidStatus):
			log.Warn("create user invalid status", "status", req.Status)
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid status"})
		default:
			log.Error("failed to create user", "auth_identifier", req.AuthIdentifier, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *UsersHandler) Replace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("userId")

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid replace user request body", "user_id", id, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	normalizeCreateUserRequest(&req)
	if req.AuthIdentifier == "" || req.Status == "" {
		log.Warn("missing required fields in replace user request", "user_id", id)
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "authIdentifier and status are required"})
		return
	}

	user, err := h.repo.Replace(id, req.AuthIdentifier, req.Email, req.GivenName, req.FamilyName, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			log.Warn("replace user not found", "user_id", id)
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "user not found"})
		case errors.Is(err, repository.ErrUserAuthIDExists):
			log.Warn("replace user conflict", "user_id", id, "auth_identifier", req.AuthIdentifier)
			writeJSON(w, http.StatusConflict, map[string]string{"message": "user auth identifier already exists"})
		case errors.Is(err, repository.ErrUserInvalidStatus):
			log.Warn("replace user invalid status", "user_id", id, "status", req.Status)
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid status"})
		default:
			log.Error("failed to replace user", "user_id", id, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *UsersHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("userId")

	var req patchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid patch user request body", "user_id", id, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	normalizePatchUserRequest(&req)

	user, err := h.repo.Patch(id, req.AuthIdentifier, req.Email, req.GivenName, req.FamilyName, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			log.Warn("patch user not found", "user_id", id)
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "user not found"})
		case errors.Is(err, repository.ErrUserAuthIDExists):
			log.Warn("patch user conflict", "user_id", id)
			writeJSON(w, http.StatusConflict, map[string]string{"message": "user auth identifier already exists"})
		case errors.Is(err, repository.ErrUserInvalidStatus):
			log.Warn("patch user invalid status", "user_id", id)
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid status"})
		default:
			log.Error("failed to patch user", "user_id", id, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *UsersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("userId")
	if err := h.repo.Delete(id); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			log.Warn("delete user not found", "user_id", id)
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "user not found"})
			return
		}
		log.Error("failed to delete user", "user_id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func normalizeCreateUserRequest(req *createUserRequest) {
	req.AuthIdentifier = strings.TrimSpace(req.AuthIdentifier)
	req.Email = strings.TrimSpace(req.Email)
	req.GivenName = strings.TrimSpace(req.GivenName)
	req.FamilyName = strings.TrimSpace(req.FamilyName)
	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
}

func normalizePatchUserRequest(req *patchUserRequest) {
	if req.AuthIdentifier != nil {
		value := strings.TrimSpace(*req.AuthIdentifier)
		req.AuthIdentifier = &value
	}
	if req.Email != nil {
		value := strings.TrimSpace(*req.Email)
		req.Email = &value
	}
	if req.GivenName != nil {
		value := strings.TrimSpace(*req.GivenName)
		req.GivenName = &value
	}
	if req.FamilyName != nil {
		value := strings.TrimSpace(*req.FamilyName)
		req.FamilyName = &value
	}
	if req.Status != nil {
		value := strings.ToUpper(strings.TrimSpace(*req.Status))
		req.Status = &value
	}
}
