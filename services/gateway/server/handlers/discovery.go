package handlers

import (
	"errors"
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
	h.log.DebugContext(r.Context(), "search metadata")

	params, err := parseSearchParams(r)
	if err != nil {
		h.log.WarnContext(r.Context(), "invalid search metadata params", "error", err.Error())
		writeBadRequest(r.Context(), w, err)
		return
	}

	results, err := h.service.Search(r.Context(), params)
	if err != nil {
		h.log.ErrorContext(r.Context(), "search metadata failed", "error", err.Error())
		switch {
		case errors.Is(err, discovery.ErrUpstreamTimeout):
			writeError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "metadata search timed out", err)
		case errors.Is(err, discovery.ErrUpstreamRequestFailed):
			writeError(r.Context(), w, http.StatusInternalServerError, "internal_server_error", "metadata search request failed", err)
		default:
			writeError(r.Context(), w, http.StatusInternalServerError, "internal_server_error", "search failed", err)
		}
		return
	}

	h.log.DebugContext(r.Context(), "search metadata success", "results_count", len(results.Items), "total", results.Total)
	writeJSON(w, http.StatusOK, results)
}

func (h *Discovery) GetConceptTree(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(r.Context(), w)
}

func parseSearchParams(r *http.Request) (discovery.SearchParams, error) {
	log := logging.Get("discovery-handler")
	query := r.URL.Query()

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
	if err := discovery.ValidateSearchTypes(types); err != nil {
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
		Providers:        parseCSVStrings(query, "provider"),
		VersioningStatus: versioningStatus,
		Keywords:         parseCSVStrings(query, "keywords"),
	}

	log.Debug("parsed search params", "page_size", params.PageSize, "page_number", params.PageNumber, "types_count", len(params.Types), "keywords_count", len(params.Keywords), "providers_count", len(params.Providers), "versioning_status_count", len(params.VersioningStatus))

	return params, nil
}
