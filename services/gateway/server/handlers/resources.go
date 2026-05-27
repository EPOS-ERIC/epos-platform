// Package handlers contains HTTP transport handlers for gateway endpoints.
package handlers

import (
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
	concept, err := h.service.GetConceptByID(id)
	if err != nil {
		h.writeResourceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, concept)
}

func (h *Resources) GetDatasetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dataset, err := h.service.GetDatasetByID(id)
	if err != nil {
		h.writeResourceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dataset)
}

func (h *Resources) GetDistributionByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	distribution, err := h.service.GetDistributionByID(id)
	if err != nil {
		h.writeResourceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, distribution)
}

func (h *Resources) GetSoftwareApplicationByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	softwareApplication, err := h.service.GetSoftwareApplicationByID(id)
	if err != nil {
		h.writeResourceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, softwareApplication)
}

func (h *Resources) GetSoftwareSourceCodeByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	softwareSourceCode, err := h.service.GetSoftwareSourceCodeByID(id)
	if err != nil {
		h.writeResourceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, softwareSourceCode)
}

func (h *Resources) GetWebserviceByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	webservice, err := h.service.GetWebserviceByID(id)
	if err != nil {
		h.writeResourceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, webservice)
}

func (h *Resources) GetOrganizationByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	organization, err := h.service.GetOrganizationByID(id)
	if err != nil {
		h.writeResourceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, organization)
}

func (h *Resources) writeResourceError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	if errors.Is(err, domain.ErrNotImplemented) {
		writeNotImplemented(w)
		return
	}

	writeError(w, http.StatusInternalServerError, "internal_server_error", "resource request failed")
}
