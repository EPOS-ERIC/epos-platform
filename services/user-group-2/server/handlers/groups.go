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

type Groups struct {
	db *db.PostgresDB
}

func NewGroups(database *db.PostgresDB) *Groups {
	return &Groups{db: database}
}

type groupResponse struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsSystem    bool      `json:"isSystem"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type createGroupRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsSystem    bool   `json:"isSystem"`
}

type patchGroupRequest struct {
	Slug        *string `json:"slug"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsSystem    *bool   `json:"isSystem"`
}

func (h *Groups) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.ListGroups(r.Context())
	if err != nil {
		log.Error("list groups failed", "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	groups := make([]groupResponse, len(rows))
	for i, row := range rows {
		groups[i] = groupResponse{
			ID:          row.ID.String(),
			Slug:        row.Slug,
			Name:        row.Name,
			Description: row.Description,
			IsSystem:    row.IsSystem,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
	}

	writeJSONGroups(w, http.StatusOK, groups)
}

func (h *Groups) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	row, err := h.db.GetGroupByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "group not found")
			return
		}
		log.Error("get group by id failed", "group_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONGroup(w, http.StatusOK, groupResponse{
		ID:          row.ID.String(),
		Slug:        row.Slug,
		Name:        row.Name,
		Description: row.Description,
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	})
}

func (h *Groups) Create(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Slug = strings.TrimSpace(req.Slug)
	req.Name = strings.TrimSpace(req.Name)
	if req.Slug == "" || req.Name == "" {
		writeMessage(w, http.StatusBadRequest, "slug and name are required")
		return
	}

	row, err := h.db.CreateGroup(r.Context(), sqlc.CreateGroupParams{
		Slug:        req.Slug,
		Name:        req.Name,
		Description: nullString(req.Description),
		IsSystem:    req.IsSystem,
	})
	if err != nil {
		if isUniqueViolation(err, "groups_slug_key") {
			writeMessage(w, http.StatusConflict, "group slug already exists")
			return
		}
		log.Error("create group failed", "slug", req.Slug, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONGroup(w, http.StatusCreated, groupResponse{
		ID:          row.ID.String(),
		Slug:        row.Slug,
		Name:        row.Name,
		Description: row.Description,
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	})
}

func (h *Groups) Replace(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Slug = strings.TrimSpace(req.Slug)
	req.Name = strings.TrimSpace(req.Name)
	if req.Slug == "" || req.Name == "" {
		writeMessage(w, http.StatusBadRequest, "slug and name are required")
		return
	}

	row, err := h.db.UpdateGroup(r.Context(), sqlc.UpdateGroupParams{
		ID:          id,
		Slug:        req.Slug,
		Name:        req.Name,
		Description: nullString(req.Description),
		IsSystem:    req.IsSystem,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "group not found")
			return
		}
		if isUniqueViolation(err, "groups_slug_key") {
			writeMessage(w, http.StatusConflict, "group slug already exists")
			return
		}
		log.Error("replace group failed", "group_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONGroup(w, http.StatusOK, groupResponse{
		ID:          row.ID.String(),
		Slug:        row.Slug,
		Name:        row.Name,
		Description: row.Description,
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	})
}

func (h *Groups) Patch(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	var req patchGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	current, err := h.db.GetGroupByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "group not found")
			return
		}
		log.Error("patch group lookup failed", "group_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	updated := sqlc.UpdateGroupParams{
		ID:          id,
		Slug:        current.Slug,
		Name:        current.Name,
		Description: nullString(current.Description),
		IsSystem:    current.IsSystem,
	}
	if req.Slug != nil {
		trimmed := strings.TrimSpace(*req.Slug)
		if trimmed == "" {
			writeMessage(w, http.StatusBadRequest, "slug cannot be empty")
			return
		}
		updated.Slug = trimmed
	}
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			writeMessage(w, http.StatusBadRequest, "name cannot be empty")
			return
		}
		updated.Name = trimmed
	}
	if req.Description != nil {
		updated.Description = nullString(*req.Description)
	}
	if req.IsSystem != nil {
		updated.IsSystem = *req.IsSystem
	}

	row, err := h.db.UpdateGroup(r.Context(), updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "group not found")
			return
		}
		if isUniqueViolation(err, "groups_slug_key") {
			writeMessage(w, http.StatusConflict, "group slug already exists")
			return
		}
		log.Error("patch group failed", "group_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSONGroup(w, http.StatusOK, groupResponse{
		ID:          row.ID.String(),
		Slug:        row.Slug,
		Name:        row.Name,
		Description: row.Description,
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	})
}

func (h *Groups) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.db.GetGroupByID(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeMessage(w, http.StatusNotFound, "group not found")
			return
		}
		log.Error("delete group lookup failed", "group_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.db.DeleteGroup(r.Context(), id); err != nil {
		log.Error("delete group failed", "group_id", id, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSONGroup(w http.ResponseWriter, status int, body groupResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("failed to encode group response", "status", status, "error", err)
	}
}

func writeJSONGroups(w http.ResponseWriter, status int, body []groupResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error("failed to encode groups response", "status", status, "error", err)
	}
}
