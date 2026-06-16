// Package handlers contains HTTP transport handlers for gateway endpoints.
package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
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

func (h *Resources) GetResourceByID(w http.ResponseWriter, r *http.Request) {
	resourceTypeValue := chi.URLParam(r, "resourceType")
	resourceType, err := api.NewResourceTypeFromValue(resourceTypeValue)
	if err != nil {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("resource type '%s' not supported", resourceTypeValue), nil)
		return
	}

	switch resourceType {
	case api.DatasetType:
		h.GetDatasetByID(w, r)
	case api.DistributionType:
		h.GetDistributionByID(w, r)
	case api.SoftwareApplicationType:
		h.GetSoftwareApplicationByID(w, r)
	case api.SoftwareSourceCodeType:
		h.GetSoftwareSourceCodeByID(w, r)
	case api.WebserviceType:
		h.GetWebserviceByID(w, r)
	case api.OrganizationType:
		h.GetOrganizationByID(w, r)
	case api.ConceptType:
		h.GetConceptByID(w, r)
	case api.ConceptSchemeType:
		h.GetConceptSchemeByID(w, r)
	default:
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("resource type '%s' not supported", resourceTypeValue), nil)
	}
}

func (h *Resources) GetConceptByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	concept, err := h.service.GetConceptByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err, "concept")
		return
	}

	writeJSON(w, http.StatusOK, concept)
}

func (h *Resources) GetConceptSchemeByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conceptScheme, err := h.service.GetConceptSchemeByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err, "conceptscheme")
		return
	}

	writeJSON(w, http.StatusOK, conceptScheme)
}

func (h *Resources) GetDatasetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dataset, err := h.service.GetDatasetByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err, "dataset")
		return
	}

	writeJSON(w, http.StatusOK, dataset)
}

func (h *Resources) GetDistributionByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	distribution, err := h.service.GetDistributionByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err, "distribution")
		return
	}

	writeJSON(w, http.StatusOK, distribution)
}

func (h *Resources) GetSoftwareApplicationByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	softwareApplication, err := h.service.GetSoftwareApplicationByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err, "software application")
		return
	}

	writeJSON(w, http.StatusOK, softwareApplication)
}

func (h *Resources) GetSoftwareSourceCodeByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	softwareSourceCode, err := h.service.GetSoftwareSourceCodeByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err, "software source code")
		return
	}

	writeJSON(w, http.StatusOK, softwareSourceCode)
}

func (h *Resources) GetWebserviceByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	webservice, err := h.service.GetWebserviceByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err, "webservice")
		return
	}

	writeJSON(w, http.StatusOK, webservice)
}

func (h *Resources) GetOrganizationByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	organization, err := h.service.GetOrganizationByID(r.Context(), id)
	if err != nil {
		h.writeResourceError(r.Context(), w, err, "organization")
		return
	}

	writeJSON(w, http.StatusOK, organization)
}

func (h *Resources) writeResourceError(ctx context.Context, w http.ResponseWriter, err error, resource string) {
	if errors.Is(err, domain.ErrNotFound) {
		WriteError(ctx, w, http.StatusNotFound, "resource_not_found", resource+" not found", nil)
		return
	}

	if errors.Is(err, domain.ErrBadRequest) {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", "request rejected by metadata service", nil)
		return
	}

	if errors.Is(err, domain.ErrUpstreamTimeout) {
		WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", "resource request timed out", err)
		return
	}

	// TODO: remove this once we have implemented everything
	if errors.Is(err, domain.ErrNotImplemented) {
		writeNotImplemented(ctx, w)
		return
	}

	log.ErrorContext(ctx, "resource request failed", "error", err)
	WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", "resource request failed", err)
}
