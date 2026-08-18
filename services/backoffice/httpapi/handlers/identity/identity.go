// Package identity exposes internal user resource operations.
package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/transport"
	identitydomain "github.com/EPOS-ERIC/epos-platform/services/backoffice/identity"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type service interface {
	GetByAuthIdentifier(context.Context, string) (identitydomain.User, error)
	Create(context.Context, identitydomain.Claims) (identitydomain.User, error)
	Delete(context.Context, uuid.UUID) error
}

type Handler struct {
	service service
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetByAuthIdentifier(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.GetByAuthIdentifier(r.Context(), chi.URLParam(r, "auth_identifier"))
	if err != nil {
		if errors.Is(err, identitydomain.ErrInvalid) {
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "auth identifier is required")
			return
		}
		if errors.Is(err, identitydomain.ErrNotFound) {
			transport.WriteProblem(w, http.StatusNotFound, "user_not_found", "user not found")
			return
		}
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "get user failed")
		return
	}
	transport.WriteJSON(w, http.StatusOK, response(user))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AuthIdentifier string `json:"auth_identifier"`
		Email          string `json:"email"`
		GivenName      string `json:"given_name"`
		FamilyName     string `json:"family_name"`
	}
	if !transport.Decode(w, r, &request) {
		return
	}

	user, err := h.service.Create(r.Context(), identitydomain.Claims{
		AuthIdentifier: request.AuthIdentifier,
		Email:          request.Email,
		GivenName:      request.GivenName,
		FamilyName:     request.FamilyName,
	})
	if err != nil {
		if errors.Is(err, identitydomain.ErrInvalid) {
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "auth_identifier, email, given_name, and family_name are required")
			return
		}
		if strings.Contains(err.Error(), "duplicate") {
			transport.WriteProblem(w, http.StatusConflict, "conflict", "user already exists")
			return
		}
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "create user failed")
		return
	}
	transport.WriteJSON(w, http.StatusCreated, response(user))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || userID == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid user id")
		return
	}
	if err := h.service.Delete(r.Context(), userID); err != nil {
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "delete user failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type userResponse struct {
	ID                uuid.UUID            `json:"id"`
	AuthIdentifier    string               `json:"auth_identifier"`
	Email             string               `json:"email"`
	GivenName         string               `json:"given_name"`
	FamilyName        string               `json:"family_name"`
	Status            string               `json:"status"`
	GlobalRoles       []string             `json:"global_roles"`
	GlobalPermissions []string             `json:"global_permissions"`
	CreatedAt         string               `json:"created_at"`
	UpdatedAt         string               `json:"updated_at"`
	Memberships       []membershipResponse `json:"memberships"`
}

type membershipResponse struct {
	Group       groupResponse `json:"group"`
	Roles       []string      `json:"roles"`
	Status      string        `json:"status"`
	Permissions []string      `json:"permissions"`
	CreatedAt   string        `json:"created_at"`
	UpdatedAt   string        `json:"updated_at"`
}

type groupResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func response(user identitydomain.User) userResponse {
	memberships := make([]membershipResponse, 0, len(user.Memberships))
	for _, membership := range user.Memberships {
		memberships = append(memberships, membershipResponse{
			Group: groupResponse{
				ID:   membership.GroupID,
				Name: membership.GroupName,
			},
			Roles:       append([]string{}, membership.Roles...),
			Status:      membership.Status,
			Permissions: append([]string{}, membership.Permissions...),
			CreatedAt:   membership.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt:   membership.UpdatedAt.Format(time.RFC3339Nano),
		})
	}

	return userResponse{
		ID:                user.ID,
		AuthIdentifier:    user.AuthIdentifier,
		Email:             user.Email,
		GivenName:         user.GivenName,
		FamilyName:        user.FamilyName,
		Status:            user.Status,
		GlobalRoles:       append([]string{}, user.GlobalRoles...),
		GlobalPermissions: append([]string{}, user.GlobalPermissions...),
		CreatedAt:         user.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:         user.UpdatedAt.Format(time.RFC3339Nano),
		Memberships:       memberships,
	}
}
