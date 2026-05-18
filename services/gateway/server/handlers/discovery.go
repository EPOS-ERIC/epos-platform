package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
)

type Discovery struct {
	service *discovery.Service
	log     *slog.Logger
}

func NewDiscovery(service *discovery.Service) *Discovery {
	return &Discovery{
		service: service,
		log:     logging.Get("discovery-handler"),
	}
}

func (h *Discovery) SearchMetadata(w http.ResponseWriter, r *http.Request) {
	h.log.Info("search metadata request", "path", r.URL.Path, "method", r.Method, "raw_query", r.URL.RawQuery)

	params, err := parseSearchParams(r)
	if err != nil {
		h.log.Warn("invalid search metadata params", "error", err.Error(), "raw_query", r.URL.RawQuery)
		writeBadRequest(w, err)
		return
	}

	results, err := h.service.Search(r.Context(), params)
	if err != nil {
		h.log.Error("search metadata failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "TODO", "TODO")
		return
	}

	h.log.Info("search metadata success", "results_count", len(results.Items))
	writeJSON(w, http.StatusOK, results)
}

func (h *Discovery) GetConceptTree(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w)
}

func parseSearchParams(r *http.Request) (discovery.SearchParams, error) {
	log := logging.Get("discovery-handler")
	query := r.URL.Query()
	log.Debug("parse search params", "raw_query", r.URL.RawQuery)

	pageSize, err := parsePageSize(query)
	if err != nil {
		return discovery.SearchParams{}, fmt.Errorf("parse page size: %w", err)
	}

	pageNumber, err := parsePageNumber(query)
	if err != nil {
		return discovery.SearchParams{}, fmt.Errorf("parse page number: %w", err)
	}

	types, err := parseCSVResourceTypes(query, "type")
	if err != nil {
		return discovery.SearchParams{}, fmt.Errorf("parse type filter: %w", err)
	}

	dateFrom, err := parseDateTime(query, "dateFrom")
	if err != nil {
		return discovery.SearchParams{}, fmt.Errorf("parse dateFrom: %w", err)
	}

	dateTo, err := parseDateTime(query, "dateTo")
	if err != nil {
		return discovery.SearchParams{}, fmt.Errorf("parse dateTo: %w", err)
	}

	if !dateFrom.IsZero() && !dateTo.IsZero() && dateTo.Before(dateFrom) {
		log.Debug("invalid search params date range", "date_from", dateFrom, "date_to", dateTo)
		return discovery.SearchParams{}, fmt.Errorf("validate date range: dateTo must be >= dateFrom")
	}

	versioningStatus, err := parseCSVVersioningStatuses(query, "versioningStatus", []api.VersioningStatus{api.PublishedStatus})
	if err != nil {
		return discovery.SearchParams{}, fmt.Errorf("parse versioning status filter: %w", err)
	}

	params := discovery.SearchParams{
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
	}

	log.Debug("parsed search params", "page_size", params.PageSize, "page_number", params.PageNumber, "types_count", len(params.Types), "keywords_count", len(params.Keywords))

	return params, nil
}
