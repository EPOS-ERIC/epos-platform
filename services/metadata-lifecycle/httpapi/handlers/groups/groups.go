// Package groups exposes public group metadata endpoints.
package groups

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	groupdomain "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/groups"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/httpapi/transport"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var log = logging.Get("groups-handler")

type service interface {
	Create(context.Context, string, groupdomain.CreateRequest) (groupdomain.Group, error)
	Replace(context.Context, string, uuid.UUID, groupdomain.ReplaceRequest) (groupdomain.Group, error)
	Delete(context.Context, string, uuid.UUID) error
	List(context.Context) ([]groupdomain.Group, error)
	Get(context.Context, uuid.UUID) (groupdomain.Group, error)
}

type Handler struct {
	service service
}

func New(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	var body createGroupRequest
	if !transport.Decode(w, r, &body) {
		return
	}

	group, err := h.service.Create(r.Context(), actorID, groupdomain.CreateRequest{
		Name:        body.Name,
		Description: body.Description,
		Label:       body.Label,
		Logo:        body.Logo,
		Page:        body.Page,
		Color:       body.Color,
		Position:    body.Position,
		Concept:     body.Concept,
	})
	if err != nil {
		switch {
		case errors.Is(err, groupdomain.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group")
		case errors.Is(err, groupdomain.ErrForbidden):
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "global admin required")
		case errors.Is(err, groupdomain.ErrConflict):
			transport.WriteProblem(w, http.StatusConflict, "conflict", "group already exists")
		default:
			log.ErrorContext(r.Context(), "create group failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "create group failed")
		}
		return
	}

	transport.WriteJSON(w, http.StatusCreated, response(group))
}

func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group id")
		return
	}

	var body replaceGroupRequest
	if !transport.Decode(w, r, &body) {
		return
	}

	request, err := body.domain()
	if err != nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "all mutable group fields are required")
		return
	}

	group, err := h.service.Replace(r.Context(), actorID, id, request)
	// TODO: maybe we should look into generalizing error handling in handlers
	if err != nil {
		switch {
		case errors.Is(err, groupdomain.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group")
		case errors.Is(err, groupdomain.ErrForbidden):
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "category scheme management permission required")
		case errors.Is(err, groupdomain.ErrNotFound):
			transport.WriteProblem(w, http.StatusNotFound, "group_not_found", "group not found")
		case errors.Is(err, groupdomain.ErrConflict):
			transport.WriteProblem(w, http.StatusConflict, "conflict", "group already exists")
		default:
			log.ErrorContext(r.Context(), "replace group failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "replace group failed")
		}
		return
	}

	transport.WriteJSON(w, http.StatusOK, response(group))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := transport.Actor(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group id")
		return
	}

	if err := h.service.Delete(r.Context(), actorID, id); err != nil {
		switch {
		case errors.Is(err, groupdomain.ErrInvalid):
			transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group")
		case errors.Is(err, groupdomain.ErrForbidden):
			transport.WriteProblem(w, http.StatusForbidden, "forbidden", "global admin required")
		case errors.Is(err, groupdomain.ErrNotFound):
			transport.WriteProblem(w, http.StatusNotFound, "group_not_found", "group not found")
		case errors.Is(err, groupdomain.ErrConflict):
			transport.WriteProblem(w, http.StatusConflict, "conflict", "group owns logical resources")
		default:
			log.ErrorContext(r.Context(), "delete group failed", "error", err)
			transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "delete group failed")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.List(r.Context())
	if err != nil {
		log.ErrorContext(r.Context(), "list groups failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "list groups failed")
		return
	}

	result := make([]groupResponse, 0, len(groups))
	for _, group := range groups {
		result = append(result, response(group))
	}

	transport.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "group_id"))
	if err != nil || id == uuid.Nil {
		transport.WriteProblem(w, http.StatusBadRequest, "invalid_request", "invalid group id")
		return
	}

	group, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, groupdomain.ErrNotFound) {
			transport.WriteProblem(w, http.StatusNotFound, "group_not_found", "group not found")
			return
		}

		log.ErrorContext(r.Context(), "get group failed", "error", err)
		transport.WriteProblem(w, http.StatusInternalServerError, "internal_server_error", "get group failed")
		return
	}

	transport.WriteJSON(w, http.StatusOK, response(group))
}

type groupResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Label       *string   `json:"label"`
	Logo        *string   `json:"logo"`
	Page        *string   `json:"page"`
	Color       *string   `json:"color"`
	Position    int32     `json:"position"`
	Concept     string    `json:"concept"`
}

type createGroupRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Label       *string `json:"label"`
	Logo        *string `json:"logo"`
	Page        *string `json:"page"`
	Color       *string `json:"color"`
	Position    int32   `json:"position"`
	Concept     string  `json:"concept"`
}

type replaceGroupRequest struct {
	Name        json.RawMessage `json:"name"`
	Description json.RawMessage `json:"description"`
	Label       json.RawMessage `json:"label"`
	Logo        json.RawMessage `json:"logo"`
	Page        json.RawMessage `json:"page"`
	Color       json.RawMessage `json:"color"`
	Position    json.RawMessage `json:"position"`
	Concept     json.RawMessage `json:"concept"`
}

func (r replaceGroupRequest) domain() (groupdomain.ReplaceRequest, error) {
	name, err := requiredString(r.Name)
	if err != nil {
		return groupdomain.ReplaceRequest{}, err
	}
	description, err := nullableString(r.Description)
	if err != nil {
		return groupdomain.ReplaceRequest{}, err
	}
	label, err := nullableString(r.Label)
	if err != nil {
		return groupdomain.ReplaceRequest{}, err
	}
	logo, err := nullableString(r.Logo)
	if err != nil {
		return groupdomain.ReplaceRequest{}, err
	}
	page, err := nullableString(r.Page)
	if err != nil {
		return groupdomain.ReplaceRequest{}, err
	}
	color, err := nullableString(r.Color)
	if err != nil {
		return groupdomain.ReplaceRequest{}, err
	}
	var position int32
	if len(r.Position) == 0 || json.Unmarshal(r.Position, &position) != nil {
		return groupdomain.ReplaceRequest{}, errors.New("invalid position")
	}
	concept, err := requiredString(r.Concept)
	if err != nil {
		return groupdomain.ReplaceRequest{}, err
	}

	return groupdomain.ReplaceRequest{
		Name:        name,
		Description: description,
		Label:       label,
		Logo:        logo,
		Page:        page,
		Color:       color,
		Position:    position,
		Concept:     concept,
	}, nil
}

func requiredString(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", errors.New("required string is missing")
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("decode string: %w", err)
	}
	return value, nil
}

func nullableString(raw json.RawMessage) (*string, error) {
	if len(raw) == 0 {
		return nil, errors.New("nullable string is missing")
	}
	if string(raw) == "null" {
		return nil, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("decode nullable string: %w", err)
	}
	return &value, nil
}

func response(group groupdomain.Group) groupResponse {
	return groupResponse{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		Label:       group.Label,
		Logo:        group.Logo,
		Page:        group.Page,
		Color:       group.Color,
		Position:    group.Position,
		Concept:     group.Concept,
	}
}
