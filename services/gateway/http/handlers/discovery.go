package handlers

import (
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
)

type Discovery struct {
	service *discovery.Service
}

func NewDiscovery(service *discovery.Service) *Discovery {
	return &Discovery{service: service}
}

func (h *Discovery) SearchMetadata(w http.ResponseWriter, r *http.Request) {
	params, err := parseSearchParams(r)
	if err != nil {
		writeBadRequest(w, err)
		return
	}

	_ = params

	results, err := h.service.Search(r.Context(), params)
	if err != nil {
		writeBadRequest(w, err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (h *Discovery) GetConceptTree(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w)
}

func parseSearchParams(r *http.Request) (discovery.SearchParams, error) {
	query := r.URL.Query()
	pageSize, err := parsePageSize(query)
	if err != nil {
		return discovery.SearchParams{}, err
	}

	pageNumber, err := parsePageNumber(query)
	if err != nil {
		return discovery.SearchParams{}, err
	}

	types, err := parseCSVResourceTypes(query, "type")
	if err != nil {
		return discovery.SearchParams{}, err
	}

	dateFrom, err := parseDateTime(query, "dateFrom")
	if err != nil {
		return discovery.SearchParams{}, err
	}

	dateTo, err := parseDateTime(query, "dateTo")
	if err != nil {
		return discovery.SearchParams{}, err
	}

	if !dateFrom.IsZero() && !dateTo.IsZero() && dateTo.Before(dateFrom) {
		return discovery.SearchParams{}, fmt.Errorf("dateTo must be >= dateFrom")
	}

	versioningStatus, err := parseCSVVersioningStatuses(query, "versioningStatus", []api.VersioningStatus{api.PublishedStatus})
	if err != nil {
		return discovery.SearchParams{}, err
	}

	return discovery.SearchParams{
		Query:            query.Get("q"),
		PageSize:         pageSize,
		PageNumber:       pageNumber,
		Types:            types,
		SpatialFilter:    query.Get("spatialFilter"),
		DateFrom:         dateFrom,
		DateTo:           dateTo,
		ServiceProviders: parseCSVStrings(query, "serviceProvider"),
		VersioningStatus: versioningStatus,
		Keywords:         parseCSVStrings(query, "keywords"),
	}, nil
}
