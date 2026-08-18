// Package handlers contains Gateway HTTP handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/lifecycle"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Lifecycle struct {
	service *lifecycle.Service
}

func NewLifecycle(service *lifecycle.Service) *Lifecycle {
	return &Lifecycle{service: service}
}

func (h *Lifecycle) Create(w http.ResponseWriter, r *http.Request) {
	h.forward(w, r, http.MethodPost, "/api/v1/groups/%s/resources/%s", true, true, "groupId", "resourceType")
}

func (h *Lifecycle) Get(w http.ResponseWriter, r *http.Request) {
	h.forward(w, r, http.MethodGet, "/api/v1/resources/%s/%s/versions/%s", false, false, "resourceType", "id", "versionId")
}

func (h *Lifecycle) ReplaceDraft(w http.ResponseWriter, r *http.Request) {
	h.forward(w, r, http.MethodPut, "/api/v1/resources/%s/%s/versions/%s/draft", true, true, "resourceType", "id", "versionId")
}

func (h *Lifecycle) Discard(w http.ResponseWriter, r *http.Request) {
	h.forward(w, r, http.MethodDelete, "/api/v1/resources/%s/%s/versions/%s/draft", false, true, "resourceType", "id", "versionId")
}

func (h *Lifecycle) Action(w http.ResponseWriter, r *http.Request) {
	action := chi.URLParam(r, "action")
	switch action {
	case "submit", "withdraw", "reject", "reopen", "publish", "unpublish":
	default:
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "unsupported lifecycle action", nil)
		return
	}
	h.forward(w, r, http.MethodPost, "/api/v1/resources/%s/%s/versions/%s/"+action, action == "reject", true, "resourceType", "id", "versionId")
}

func (h *Lifecycle) DraftFromPublication(w http.ResponseWriter, r *http.Request) {
	h.forward(w, r, http.MethodPost, "/api/v1/resources/%s/%s/draft-from-publication", false, true, "resourceType", "id")
}

func (h *Lifecycle) DraftFromArchive(w http.ResponseWriter, r *http.Request) {
	h.forward(w, r, http.MethodPost, "/api/v1/resources/%s/%s/draft-from-archive", false, true, "resourceType", "id")
}

func (h *Lifecycle) Copy(w http.ResponseWriter, r *http.Request) {
	h.forward(w, r, http.MethodPost, "/api/v1/groups/%s/resources/%s/%s/copy", true, true, "groupId", "resourceType", "id")
}

func (h *Lifecycle) forward(w http.ResponseWriter, r *http.Request, method, pattern string, requiresBody, requiresActor bool, params ...string) {
	path, err := lifecyclePath(pattern, params, r)
	if err != nil {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	body, err := lifecycleBody(w, r, requiresBody)
	if err != nil {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err)
		return
	}

	actor, ok := AuthenticatedUserFromContext(r.Context())
	if requiresActor && !ok {
		WriteError(r.Context(), w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}

	response, err := h.service.Request(r.Context(), method, path, actor.ID, body)
	if err != nil {
		h.writeError(r.Context(), w, err)
		return
	}

	if response.ContentType != "" {
		w.Header().Set("Content-Type", response.ContentType)
	}
	w.WriteHeader(response.StatusCode)
	if _, err := w.Write(response.Body); err != nil {
		log.ErrorContext(r.Context(), "write lifecycle response failed", "error", err)
	}
}

func lifecyclePath(pattern string, params []string, r *http.Request) (string, error) {
	path := pattern
	for _, param := range params {
		value := chi.URLParam(r, param)
		switch param {
		case "resourceType":
			if !validLifecycleResourceType(value) {
				return "", fmt.Errorf("resource type '%s' not supported", value)
			}
		case "id", "versionId", "groupId":
			if _, err := uuid.Parse(value); err != nil {
				return "", fmt.Errorf("invalid %s", param)
			}
		}
		path = strings.Replace(path, "%s", value, 1)
	}
	return path, nil
}

func validLifecycleResourceType(value string) bool {
	switch value {
	case "dataset", "software-application", "software-source-code":
		return true
	default:
		return false
	}
}

func lifecycleBody(w http.ResponseWriter, r *http.Request, required bool) ([]byte, error) {
	if !required {
		return nil, nil
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read JSON request: %w", err)
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("invalid JSON")
	}
	return body, nil
}

func (h *Lifecycle) writeError(ctx context.Context, w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrUpstreamTimeout) {
		WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "lifecycle request timed out", err)
		return
	}
	log.ErrorContext(ctx, "lifecycle request failed", "error", err)
	WriteError(ctx, w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
}
