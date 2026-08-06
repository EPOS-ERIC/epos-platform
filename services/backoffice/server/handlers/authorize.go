package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/service"
)

type Authorize struct {
	service *service.Authorizer
	log     *slog.Logger
}

func NewAuthorize(svc *service.Authorizer) *Authorize {
	return &Authorize{
		service: svc,
		log:     logging.Get("authorize-handler"),
	}
}

type AuthorizeRequest struct {
	AuthIdentifier          string `json:"authIdentifier"`
	Group                   string `json:"group"`
	Service                 string `json:"service"`
	Action                  string `json:"action"`
	ResourceType            string `json:"resourceType,omitempty"`
	ResourceID              string `json:"resourceId,omitempty"`
	ResourceOwnerIdentifier string `json:"resourceOwnerIdentifier,omitempty"`
}

type AuthorizeResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

func (h *Authorize) Authorize(w http.ResponseWriter, r *http.Request) {
	var req AuthorizeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WarnContext(r.Context(), "invalid authorize request body", "error", err)
		writeMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.AuthIdentifier = strings.TrimSpace(req.AuthIdentifier)
	req.Group = strings.TrimSpace(req.Group)
	req.Service = strings.TrimSpace(req.Service)
	req.Action = strings.TrimSpace(req.Action)

	if req.AuthIdentifier == "" || req.Group == "" || req.Service == "" || req.Action == "" {
		h.log.WarnContext(r.Context(), "authorize request missing required fields")
		writeMessage(w, http.StatusBadRequest, "authIdentifier, group, service and action are required")
		return
	}

	result, err := h.service.Decide(r.Context(), service.AuthorizeRequest{
		AuthIdentifier:          req.AuthIdentifier,
		Group:                   req.Group,
		Service:                 req.Service,
		Action:                  req.Action,
		ResourceType:            req.ResourceType,
		ResourceID:              req.ResourceID,
		ResourceOwnerIdentifier: req.ResourceOwnerIdentifier,
	})
	if err != nil {
		h.log.ErrorContext(r.Context(), "authorize decision failed", "auth_identifier", req.AuthIdentifier, "group", req.Group, "service", req.Service, "action", req.Action, "error", err)
		writeMessage(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, AuthorizeResponse{Allowed: result.Allowed, Reason: result.Reason})
}
