// Package handlers contains HTTP transport handlers for gateway endpoints.
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/resources"
	"github.com/go-chi/chi/v5"
)

type Resources struct {
	service *resources.Service
}

func NewResources(service *resources.Service) *Resources {
	return &Resources{service: service}
}

func (h *Resources) GetConceptByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	concept, err := h.service.GetConceptByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err)
		return
	}

	writeJSON(w, http.StatusOK, concept)
}

func (h *Resources) GetConceptSchemeByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conceptScheme, err := h.service.GetConceptSchemeByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err)
		return
	}

	writeJSON(w, http.StatusOK, conceptScheme)
}

func (h *Resources) GetDatasetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dataset, err := h.service.GetDatasetByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err)
		return
	}

	writeJSON(w, http.StatusOK, dataset)
}

func (h *Resources) GetDistributionByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	distribution, err := h.service.GetDistributionByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err)
		return
	}

	writeJSON(w, http.StatusOK, distribution)
}

func (h *Resources) GetSoftwareApplicationByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	softwareApplication, err := h.service.GetSoftwareApplicationByID(id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err)
		return
	}

	writeJSON(w, http.StatusOK, softwareApplication)
}

func (h *Resources) GetSoftwareSourceCodeByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	softwareSourceCode, err := h.service.GetSoftwareSourceCodeByID(id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err)
		return
	}

	writeJSON(w, http.StatusOK, softwareSourceCode)
}

func (h *Resources) GetWebserviceByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	webservice, err := h.service.GetWebserviceByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err)
		return
	}

	writeJSON(w, http.StatusOK, webservice)
}

func (h *Resources) GetOrganizationByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	organization, err := h.service.GetOrganizationByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err)
		return
	}

	writeJSON(w, http.StatusOK, organization)
}

func (h *Resources) writeResourceError(ctx context.Context, w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		writeError(ctx, w, http.StatusNotFound, "not_found", "resource not found", nil)
		return
	}

	if errors.Is(err, domain.ErrUpstreamTimeout) {
		writeError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "resource request timed out", err)
		return
	}

	// TODO: remove this once we have implemented everything
	if errors.Is(err, domain.ErrNotImplemented) {
		writeNotImplemented(ctx, w)
		return
	}

	log.ErrorContext(ctx, "resource request failed", "error", err)
	writeError(ctx, w, http.StatusInternalServerError, "internal_server_error", "resource request failed", err)
}
