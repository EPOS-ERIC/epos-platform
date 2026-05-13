package handlers

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/resources"
)

type Resources struct {
	service *resources.Service
}

func NewResources(service *resources.Service) *Resources {
	return &Resources{service: service}
}

func (h *Resources) GetConceptByID(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w)
}

func (h *Resources) GetDatasetByID(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w)
}

func (h *Resources) GetDistributionByID(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w)
}

func (h *Resources) GetSoftwareApplicationByID(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w)
}

func (h *Resources) GetSoftwareSourceCodeByID(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w)
}
