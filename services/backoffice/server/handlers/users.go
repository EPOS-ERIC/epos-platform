package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/service"
	"github.com/go-chi/chi/v5"
)

type Users struct {
	service *service.Users
	log     *slog.Logger
}

func NewUsers(svc *service.Users) *Users {
	return &Users{
		service: svc,
		log:     logging.Get("users-handler"),
	}
}

type createUserRequest struct {
	AuthIdentifier string             `json:"authIdentifier"`
	Email          string             `json:"email"`
	GivenName      string             `json:"givenName"`
	FamilyName     string             `json:"familyName"`
	IsSuperAdmin   bool               `json:"isSuperAdmin"`
	Status         service.UserStatus `json:"status"`
}

type patchUserRequest struct {
	AuthIdentifier *string             `json:"authIdentifier"`
	Email          *string             `json:"email"`
	GivenName      *string             `json:"givenName"`
	FamilyName     *string             `json:"familyName"`
	IsSuperAdmin   *bool               `json:"isSuperAdmin"`
	Status         *service.UserStatus `json:"status"`
}

func (h *Users) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "list users failed", "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *Users) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "userId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.log.DebugContext(r.Context(), "user not found", "user_id", id)
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
		h.log.ErrorContext(r.Context(), "get user by id failed", "user_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Users) GetByAuthIdentifier(w http.ResponseWriter, r *http.Request) {
	authIdentifier := chi.URLParam(r, "authIdentifier")
	user, err := h.service.GetByAuthIdentifier(r.Context(), authIdentifier)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.log.DebugContext(r.Context(), "user not found", "auth_identifier", authIdentifier)
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
		h.log.ErrorContext(r.Context(), "get user by auth identifier failed", "auth_identifier", authIdentifier, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Users) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WarnContext(r.Context(), "invalid create user request body", "error", err)
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Create(r.Context(), service.CreateUserParams{
		AuthIdentifier: req.AuthIdentifier,
		Email:          req.Email,
		GivenName:      req.GivenName,
		FamilyName:     req.FamilyName,
		IsSuperAdmin:   req.IsSuperAdmin,
		Status:         req.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalid):
			h.log.WarnContext(r.Context(), "invalid create user request")
			writeMessage(w, http.StatusBadRequest, "invalid request")
		case errors.Is(err, service.ErrConflict):
			h.log.WarnContext(r.Context(), "create user conflict", "auth_identifier", req.AuthIdentifier)
			writeMessage(w, http.StatusConflict, "user already exists")
		default:
			h.log.ErrorContext(r.Context(), "create user failed", "error", err)
			writeMessage(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *Users) Replace(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "userId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WarnContext(r.Context(), "invalid replace user request body", "error", err)
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Replace(r.Context(), service.UpdateUserParams{
		ID:             id,
		AuthIdentifier: req.AuthIdentifier,
		Email:          req.Email,
		GivenName:      req.GivenName,
		FamilyName:     req.FamilyName,
		IsSuperAdmin:   req.IsSuperAdmin,
		Status:         req.Status,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.log.DebugContext(r.Context(), "user not found", "user_id", id)
		}
		h.log.ErrorContext(r.Context(), "replace user failed", "user_id", id, "error", err)
		writeUserError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Users) Patch(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "userId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	var req patchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WarnContext(r.Context(), "invalid patch user request body", "error", err)
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Patch(r.Context(), id, service.PatchUserParams{
		AuthIdentifier: req.AuthIdentifier,
		Email:          req.Email,
		GivenName:      req.GivenName,
		FamilyName:     req.FamilyName,
		IsSuperAdmin:   req.IsSuperAdmin,
		Status:         req.Status,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.log.DebugContext(r.Context(), "user not found", "user_id", id)
		}
		h.log.ErrorContext(r.Context(), "patch user failed", "user_id", id, "error", err)
		writeUserError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Users) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "userId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.log.DebugContext(r.Context(), "user not found", "user_id", id)
		}
		h.log.ErrorContext(r.Context(), "delete user failed", "user_id", id, "error", err)
		writeUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeMessage(w, http.StatusNotFound, "user not found")
	case errors.Is(err, service.ErrConflict):
		writeMessage(w, http.StatusConflict, "user already exists")
	case errors.Is(err, service.ErrInvalid):
		writeMessage(w, http.StatusBadRequest, "invalid request")
	default:
		writeMessage(w, http.StatusInternalServerError, "internal error")
	}
}
