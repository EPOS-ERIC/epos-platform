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
			log.Warn("group not found", "group_id", id)
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "group not found"})
			return
		}
		log.Error("failed to get group by id", "group_id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, group)
}

func (h *GroupsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid create group request body", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	req.Slug = strings.TrimSpace(req.Slug)
	req.Name = strings.TrimSpace(req.Name)
	if req.Slug == "" || req.Name == "" {
		log.Warn("missing required fields in create group request")
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "slug and name are required"})
		return
	}

	group, err := h.repo.Create(req.Slug, req.Name, req.Description, req.IsSystem)
	if err != nil {
		if errors.Is(err, repository.ErrGroupSlugExists) {
			log.Warn("create group conflict", "slug", req.Slug)
			writeJSON(w, http.StatusConflict, map[string]string{"message": "group slug already exists"})
			return
		}
		log.Error("failed to create group", "slug", req.Slug, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, group)
}

func (h *GroupsHandler) Replace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("groupId")

	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid replace group request body", "group_id", id, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	req.Slug = strings.TrimSpace(req.Slug)
	req.Name = strings.TrimSpace(req.Name)
	if req.Slug == "" || req.Name == "" {
		log.Warn("missing required fields in replace group request", "group_id", id)
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "slug and name are required"})
		return
	}

	group, err := h.repo.Replace(id, req.Slug, req.Name, req.Description, req.IsSystem)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrGroupNotFound):
			log.Warn("replace group not found", "group_id", id)
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "group not found"})
		case errors.Is(err, repository.ErrGroupSlugExists):
			log.Warn("replace group conflict", "group_id", id, "slug", req.Slug)
			writeJSON(w, http.StatusConflict, map[string]string{"message": "group slug already exists"})
		default:
			log.Error("failed to replace group", "group_id", id, "error", err)
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
		log.Warn("invalid patch group request body", "group_id", id, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}

	if req.Slug != nil {
		trimmed := strings.TrimSpace(*req.Slug)
		if trimmed == "" {
			log.Warn("empty slug in patch group request", "group_id", id)
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "slug cannot be empty"})
			return
		}
		req.Slug = &trimmed
	}
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			log.Warn("empty name in patch group request", "group_id", id)
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "name cannot be empty"})
			return
		}
		req.Name = &trimmed
	}

	group, err := h.repo.Patch(id, req.Slug, req.Name, req.Description, req.IsSystem)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrGroupNotFound):
			log.Warn("patch group not found", "group_id", id)
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "group not found"})
		case errors.Is(err, repository.ErrGroupSlugExists):
			log.Warn("patch group conflict", "group_id", id)
			writeJSON(w, http.StatusConflict, map[string]string{"message": "group slug already exists"})
		default:
			log.Error("failed to patch group", "group_id", id, "error", err)
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
			log.Warn("delete group not found", "group_id", id)
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "group not found"})
			return
		}
		log.Error("failed to delete group", "group_id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "internal error"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
