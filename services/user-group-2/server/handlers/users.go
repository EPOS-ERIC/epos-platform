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
	"github.com/go-chi/chi/v5"
)

type Users struct {
	db *db.PostgresDB
}

func NewUsers(database *db.PostgresDB) *Users {
	return &Users{db: database}
}

type userResponse struct {
	ID             string    `json:"id"`
	AuthIdentifier string    `json:"authIdentifier"`
	Email          string    `json:"email,omitempty"`
	GivenName      string    `json:"givenName,omitempty"`
	FamilyName     string    `json:"familyName,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
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
	rows, err := h.db.ListUsers(r.Context())
	if err != nil {
		log.Error("list users failed", "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	users := make([]userResponse, len(rows))
	for i, row := range rows {
		users[i] = userResponse{
			ID:             row.ID.String(),
			AuthIdentifier: row.AuthIdentifier,
			Email:          row.Email,
			GivenName:      row.GivenName,
			FamilyName:     row.FamilyName,
			Status:         row.Status,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(users); err != nil {
		log.Error("failed to encode users response", "error", err)
	}
}

func (h *Users) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "userId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	row, err := h.db.GetUserByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
		log.Error("get user by id failed", "user_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONUser(w, http.StatusOK, userResponse{
		ID:             row.ID.String(),
		AuthIdentifier: row.AuthIdentifier,
		Email:          row.Email,
		GivenName:      row.GivenName,
		FamilyName:     row.FamilyName,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}

func (h *Users) GetByAuthIdentifier(w http.ResponseWriter, r *http.Request) {
	authIdentifier := strings.TrimSpace(chi.URLParam(r, "authIdentifier"))
	if authIdentifier == "" {
		writeMessage(w, http.StatusBadRequest, "authIdentifier is required")
		return
	}

	row, err := h.db.GetUserByAuthIdentifier(r.Context(), authIdentifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
		log.Error("get user by auth identifier failed", "auth_identifier", authIdentifier, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONUser(w, http.StatusOK, userResponse{
		ID:             row.ID.String(),
		AuthIdentifier: row.AuthIdentifier,
		Email:          row.Email,
		GivenName:      row.GivenName,
		FamilyName:     row.FamilyName,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}

func (h *Users) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	normalizeUserCreateRequest(&req)
	if req.AuthIdentifier == "" {
		writeMessage(w, http.StatusBadRequest, "authIdentifier is required")
		return
	}
	if req.Status == "" {
		req.Status = "ACTIVE"
	}

	row, err := h.db.CreateUser(r.Context(), sqlc.CreateUserParams{
		AuthIdentifier: req.AuthIdentifier,
		Email:          nullString(req.Email),
		GivenName:      nullString(req.GivenName),
		FamilyName:     nullString(req.FamilyName),
		Status:         req.Status,
	})
	if err != nil {
		if isUniqueViolation(err, "users_auth_identifier_key") {
			writeMessage(w, http.StatusConflict, "user auth identifier already exists")
			return
		}
		if isUniqueViolation(err, "users_email_key") {
			writeMessage(w, http.StatusConflict, "user email already exists")
			return
		}
		log.Error("create user failed", "auth_identifier", req.AuthIdentifier, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONUser(w, http.StatusCreated, userResponse{
		ID:             row.ID.String(),
		AuthIdentifier: row.AuthIdentifier,
		Email:          row.Email,
		GivenName:      row.GivenName,
		FamilyName:     row.FamilyName,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
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

	normalizeUserCreateRequest(&req)
	if req.AuthIdentifier == "" || req.Status == "" {
		writeMessage(w, http.StatusBadRequest, "authIdentifier and status are required")
		return
	}

	row, err := h.db.UpdateUser(r.Context(), sqlc.UpdateUserParams{
		ID:             id,
		AuthIdentifier: req.AuthIdentifier,
		Email:          nullString(req.Email),
		GivenName:      nullString(req.GivenName),
		FamilyName:     nullString(req.FamilyName),
		Status:         req.Status,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
		if isUniqueViolation(err, "users_auth_identifier_key") {
			writeMessage(w, http.StatusConflict, "user auth identifier already exists")
			return
		}
		if isUniqueViolation(err, "users_email_key") {
			writeMessage(w, http.StatusConflict, "user email already exists")
			return
		}
		log.Error("replace user failed", "user_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONUser(w, http.StatusOK, userResponse{
		ID:             row.ID.String(),
		AuthIdentifier: row.AuthIdentifier,
		Email:          row.Email,
		GivenName:      row.GivenName,
		FamilyName:     row.FamilyName,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
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

	normalizeUserPatchRequest(&req)
	current, err := h.db.GetUserByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
		log.Error("patch user lookup failed", "user_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	updated := sqlc.UpdateUserParams{
		ID:             id,
		AuthIdentifier: current.AuthIdentifier,
		Email:          nullString(current.Email),
		GivenName:      nullString(current.GivenName),
		FamilyName:     nullString(current.FamilyName),
		Status:         current.Status,
	}
	if req.AuthIdentifier != nil {
		updated.AuthIdentifier = *req.AuthIdentifier
	}
	if req.Email != nil {
		updated.Email = nullString(*req.Email)
	}
	if req.GivenName != nil {
		updated.GivenName = nullString(*req.GivenName)
	}
	if req.FamilyName != nil {
		updated.FamilyName = nullString(*req.FamilyName)
	}
	if req.Status != nil {
		updated.Status = *req.Status
	}

	row, err := h.db.UpdateUser(r.Context(), updated)
	if err != nil {
		if isUniqueViolation(err, "users_auth_identifier_key") {
			writeMessage(w, http.StatusConflict, "user auth identifier already exists")
			return
		}
		if isUniqueViolation(err, "users_email_key") {
			writeMessage(w, http.StatusConflict, "user email already exists")
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
		log.Error("patch user failed", "user_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONUser(w, http.StatusOK, userResponse{
		ID:             row.ID.String(),
		AuthIdentifier: row.AuthIdentifier,
		Email:          row.Email,
		GivenName:      row.GivenName,
		FamilyName:     row.FamilyName,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}

func (h *Users) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "userId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.db.GetUserByID(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "user not found")
			return
		}
		log.Error("delete user lookup failed", "user_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.db.DeleteUser(r.Context(), id); err != nil {
		log.Error("delete user failed", "user_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func normalizeUserCreateRequest(req *createUserRequest) {
	req.AuthIdentifier = strings.TrimSpace(req.AuthIdentifier)
	req.Email = strings.TrimSpace(req.Email)
	req.GivenName = strings.TrimSpace(req.GivenName)
	req.FamilyName = strings.TrimSpace(req.FamilyName)
	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
}

func normalizeUserPatchRequest(req *patchUserRequest) {
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

func writeJSONUser(w http.ResponseWriter, status int, body userResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("failed to encode user response", "status", status, "error", err)
	}
}

func isUniqueViolation(err error, constraint string) bool {
	return strings.Contains(err.Error(), constraint)
}
