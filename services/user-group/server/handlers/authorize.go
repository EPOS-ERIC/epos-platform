package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

type AuthorizeRequest struct {
	AuthIdentifier string `json:"authIdentifier"`
	Group          string `json:"group"`
	Service        string `json:"service"`
	Action         string `json:"action"`
	ResourceType   string `json:"resourceType,omitempty"`
	ResourceID     string `json:"resourceId,omitempty"`
}

type AuthorizeResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

func Authorize(w http.ResponseWriter, r *http.Request) {
	var req AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, AuthorizeResponse{Allowed: false, Reason: "invalid request body"})
		return
	}

	if strings.TrimSpace(req.AuthIdentifier) == "" || strings.TrimSpace(req.Group) == "" || strings.TrimSpace(req.Service) == "" || strings.TrimSpace(req.Action) == "" {
		writeJSON(w, http.StatusBadRequest, AuthorizeResponse{Allowed: false, Reason: "authIdentifier, group, service and action are required"})
		return
	}

	writeJSON(w, http.StatusNotImplemented, AuthorizeResponse{
		Allowed: false,
		Reason:  "authorization backend not wired yet",
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
