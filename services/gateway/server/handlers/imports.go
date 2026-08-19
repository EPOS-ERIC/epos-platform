package handlers

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	metadataimports "github.com/EPOS-ERIC/epos-platform/services/gateway/imports"
)

type Imports struct {
	service *metadataimports.Service
}

func NewImports(service *metadataimports.Service) *Imports {
	return &Imports{service: service}
}

func (h *Imports) Turtle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := AuthenticatedUserFromContext(ctx)
	if !ok {
		WriteError(ctx, w, http.StatusUnauthorized, "missing_token", "bearer token required", nil)
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "text/turtle" {
		WriteError(ctx, w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be text/turtle", nil)
		return
	}

	result, err := h.service.Turtle(ctx, actor.ID, http.MaxBytesReader(w, r.Body, 50<<20))
	if err != nil {
		if errors.Is(err, domain.ErrBadRequest) {
			WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "invalid Turtle request body", err)
			return
		}
		if responseErr, ok := errors.AsType[*metadataimports.ResponseError](err); ok {
			if responseErr.ContentType != "" {
				w.Header().Set("Content-Type", responseErr.ContentType)
			}
			w.WriteHeader(responseErr.StatusCode)
			_, _ = w.Write(responseErr.Body)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "metadata import request timed out", err)
			return
		}
		log.ErrorContext(ctx, "metadata import request failed", "error", err)
		WriteError(ctx, w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	response := importResultResponse{
		Groups:               result.Groups,
		Categories:           result.Categories,
		Organizations:        result.Organizations,
		ContactPoints:        result.ContactPoints,
		Datasets:             result.Datasets,
		SoftwareApplications: result.SoftwareApplications,
		SoftwareSourceCodes:  result.SoftwareSourceCodes,
		Warnings:             make([]importDiagnosticResponse, 0, len(result.Warnings)),
	}
	for _, warning := range result.Warnings {
		response.Warnings = append(response.Warnings, importDiagnosticResponse{
			Severity:  warning.Severity,
			Subject:   warning.Subject,
			Predicate: warning.Predicate,
			Message:   warning.Message,
		})
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.ErrorContext(ctx, "encode metadata import response failed", "error", err)
	}
}

type importResultResponse struct {
	Groups               int32                      `json:"groups"`
	Categories           int32                      `json:"categories"`
	Organizations        int32                      `json:"organizations"`
	ContactPoints        int32                      `json:"contact_points"`
	Datasets             int32                      `json:"datasets"`
	SoftwareApplications int32                      `json:"software_applications"`
	SoftwareSourceCodes  int32                      `json:"software_source_codes"`
	Warnings             []importDiagnosticResponse `json:"warnings"`
}

type importDiagnosticResponse struct {
	Severity  string `json:"severity"`
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Message   string `json:"message"`
}
