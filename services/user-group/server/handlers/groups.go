package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/user-group/repository"
)

type GroupsHandler struct {
	repo repository.GroupRepository
}

func NewGroupsHandler(repo repository.GroupRepository) *GroupsHandler {
	return &GroupsHandler{repo: repo}
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

func (h *GroupsHandler) List(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.repo.List())
}

func (h *GroupsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("groupId")
	group, err := h.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrGroupNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "group not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, group)
}

func (h *GroupsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	req.Slug = strings.TrimSpace(req.Slug)
	req.Name = strings.TrimSpace(req.Name)
	if req.Slug == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "slug and name are required"})
		return
	}

	group, err := h.repo.Create(req.Slug, req.Name, req.Description, req.IsSystem)
	if err != nil {
		if errors.Is(err, repository.ErrGroupSlugExists) {
			writeJSON(w, http.StatusConflict, map[string]string{"message": "group slug already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, group)
}

func (h *GroupsHandler) Replace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("groupId")

	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	req.Slug = strings.TrimSpace(req.Slug)
	req.Name = strings.TrimSpace(req.Name)
	if req.Slug == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "slug and name are required"})
		return
	}

	group, err := h.repo.Replace(id, req.Slug, req.Name, req.Description, req.IsSystem)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrGroupNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "group not found"})
		case errors.Is(err, repository.ErrGroupSlugExists):
			writeJSON(w, http.StatusConflict, map[string]string{"message": "group slug already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, group)
}

func (h *GroupsHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("groupId")

	var req patchGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	if req.Slug != nil {
		trimmed := strings.TrimSpace(*req.Slug)
		if trimmed == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "slug cannot be empty"})
			return
		}
		req.Slug = &trimmed
	}
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "name cannot be empty"})
			return
		}
		req.Name = &trimmed
	}

	group, err := h.repo.Patch(id, req.Slug, req.Name, req.Description, req.IsSystem)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrGroupNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "group not found"})
		case errors.Is(err, repository.ErrGroupSlugExists):
			writeJSON(w, http.StatusConflict, map[string]string{"message": "group slug already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, group)
}

func (h *GroupsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("groupId")
	if err := h.repo.Delete(id); err != nil {
		if errors.Is(err, repository.ErrGroupNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "group not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
