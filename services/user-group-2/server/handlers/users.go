package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/service"
	"github.com/go-chi/chi/v5"
)

type Users struct {
	service *service.Users
}

func NewUsers(svc *service.Users) *Users {
	return &Users{
		service: svc,
	}
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

func (h *Users) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
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
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
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
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Users) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Create(r.Context(), service.CreateUserParams{
		AuthIdentifier: req.AuthIdentifier,
		Email:          req.Email,
		GivenName:      req.GivenName,
		FamilyName:     req.FamilyName,
		Status:         req.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalid):
			writeMessage(w, http.StatusBadRequest, "invalid request")
		case errors.Is(err, service.ErrConflict):
			writeMessage(w, http.StatusConflict, "user already exists")
		default:
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
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Replace(r.Context(), service.UpdateUserParams{
		ID:             id,
		AuthIdentifier: req.AuthIdentifier,
		Email:          req.Email,
		GivenName:      req.GivenName,
		FamilyName:     req.FamilyName,
		Status:         req.Status,
	})
	if err != nil {
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
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Patch(r.Context(), id, service.PatchUserParams{
		AuthIdentifier: req.AuthIdentifier,
		Email:          req.Email,
		GivenName:      req.GivenName,
		FamilyName:     req.FamilyName,
		Status:         req.Status,
	})
	if err != nil {
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
