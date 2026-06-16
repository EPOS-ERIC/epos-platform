package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
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
		h.writeDiscoveryError(r.Context(), w, err, "metadata search")
		return
	}

	h.log.DebugContext(r.Context(), "search metadata success", "results_count", len(results.Items), "total", results.Total)
	writeJSON(w, http.StatusOK, results)
}

func (h *Discovery) GetConceptTree(w http.ResponseWriter, r *http.Request) {
	results, err := h.service.GetConceptTree(r.Context())
	if err != nil {
		h.log.Error("get concept tree failed", "method", r.Method, "path", r.URL.Path, "error", err.Error())
		h.writeDiscoveryError(r.Context(), w, err, "concept tree")
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (h *Discovery) writeDiscoveryError(ctx context.Context, w http.ResponseWriter, err error, operation string) {
	if errors.Is(err, discovery.ErrUpstreamTimeout) {
		WriteError(ctx, w, http.StatusGatewayTimeout, "upstream_timeout", operation+" timed out", err)
		return
	}

	if errors.Is(err, domain.ErrBadRequest) {
		WriteError(ctx, w, http.StatusBadRequest, "invalid_request", operation+" request rejected by metadata service", nil)
		return
	}

	if errors.Is(err, discovery.ErrUpstreamRequestFailed) {
		WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", operation+" request failed", err)
		return
	}

	WriteError(ctx, w, http.StatusInternalServerError, "internal_server_error", operation+" failed", err)
}

func parseSearchParams(r *http.Request) (discovery.SearchParams, error) {
	log := logging.Get("discovery-handler")
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
	if err := discovery.ValidateSearchTypes(types); err != nil {
		// keep raw validation message for api contract
		return discovery.SearchParams{}, err //nolint:wrapcheck
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
		log.Debug("invalid search params date range", "date_from", dateFrom, "date_to", dateTo)
		return discovery.SearchParams{}, fmt.Errorf("date to must be greater than or equal to date from")
	}

	versioningStatus, err := parseCSVVersioningStatuses(query, "versioningStatus", []api.VersioningStatus{api.PublishedStatus})
	if err != nil {
		return discovery.SearchParams{}, err
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
