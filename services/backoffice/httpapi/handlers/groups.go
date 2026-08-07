package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	service "github.com/EPOS-ERIC/epos-platform/services/backoffice/identity"
)

type Groups struct {
	service *service.Groups
	log     *slog.Logger
}

func NewGroups(svc *service.Groups) *Groups {
	return &Groups{
		service: svc,
		log:     logging.Get("groups-handler"),
	}
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
	items, err := h.service.List(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "list groups failed", "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Groups) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.log.DebugContext(r.Context(), "group not found", "group_id", id)
		}
		h.log.ErrorContext(r.Context(), "get group failed", "group_id", id, "error", err)
		writeServiceError(w, err, "group")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Groups) Create(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WarnContext(r.Context(), "invalid create group request body", "error", err)
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Create(r.Context(), service.CreateGroupParams{
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    req.IsSystem,
	})
	if err != nil {
		h.log.ErrorContext(r.Context(), "create group failed", "error", err)
		writeServiceError(w, err, "group")
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *Groups) Replace(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	var req createGroupRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WarnContext(r.Context(), "invalid replace group request body", "error", err)
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Replace(r.Context(), service.UpdateGroupParams{
		ID:          id,
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    req.IsSystem,
	})
	if err != nil {
		h.log.ErrorContext(r.Context(), "replace group failed", "group_id", id, "error", err)
		writeServiceError(w, err, "group")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Groups) Patch(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	var req patchGroupRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WarnContext(r.Context(), "invalid patch group request body", "error", err)
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.Patch(r.Context(), id, service.PatchGroupParams{
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    req.IsSystem,
	})
	if err != nil {
		h.log.ErrorContext(r.Context(), "patch group failed", "group_id", id, "error", err)
		writeServiceError(w, err, "group")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Groups) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "groupId")
	if err != nil {
		writeMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		h.log.ErrorContext(r.Context(), "delete group failed", "group_id", id, "error", err)
		writeServiceError(w, err, "group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeServiceError(w http.ResponseWriter, err error, resource string) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeMessage(w, http.StatusNotFound, resource+" not found")
	case errors.Is(err, service.ErrConflict):
		writeMessage(w, http.StatusConflict, resource+" already exists")
	case errors.Is(err, service.ErrInvalid):
		writeMessage(w, http.StatusBadRequest, "invalid request")
	default:
		writeMessage(w, http.StatusInternalServerError, "internal error")
	}
}
