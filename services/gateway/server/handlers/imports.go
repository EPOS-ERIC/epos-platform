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
		if errors.Is(err, domain.ErrForbidden) {
			WriteError(ctx, w, http.StatusForbidden, "forbidden", "user access denied", err)
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			WriteError(ctx, w, http.StatusConflict, "resource_conflict", "import conflicts with existing metadata", err)
			return
		}
		if errors.Is(err, domain.ErrUpstreamTimeout) {
			WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "metadata import request timed out", err)
			return
		}
		log.ErrorContext(ctx, "metadata import request failed", "error", err)
		WriteError(ctx, w, http.StatusServiceUnavailable, "metadata_lifecycle_unavailable", "metadata-lifecycle service unavailable", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	response := toAPIImportResult(*result)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.ErrorContext(ctx, "encode metadata import response failed", "error", err)
	}
}
