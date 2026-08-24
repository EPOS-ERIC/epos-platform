package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Discovery struct {
	service *discovery.Service
}

func NewDiscovery(service *discovery.Service) *Discovery {
	return &Discovery{service: service}
}

func (h *Discovery) Search(w http.ResponseWriter, r *http.Request) {
	filter, err := discoveryFilterFromRequest(r.URL.Query())
	if err != nil {
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", err.Error(), err)
		return
	}

	actorID := ""
	if actor, ok := AuthenticatedUserFromContext(r.Context()); ok {
		actorID = actor.ID
	}
	items, err := h.service.Search(r.Context(), actorID, filter)
	if err != nil {
		writeDiscoveryError(r, w, err)
		return
	}

	result := make([]api.DiscoveryItem, 0, len(items))
	for _, item := range items {
		result = append(result, discoveryItemFromDomain(item))
	}
	writeJSON(w, http.StatusOK, result)
}

func discoveryFilterFromRequest(values url.Values) (domain.DiscoveryFilter, error) {
	dateFrom, err := parseDiscoveryTime(values.Get("dateFrom"), "dateFrom")
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	dateTo, err := parseDiscoveryTime(values.Get("dateTo"), "dateTo")
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	if dateFrom != nil && dateTo != nil && dateFrom.After(*dateTo) {
		return domain.DiscoveryFilter{}, fmt.Errorf("dateFrom must be less than or equal to dateTo")
	}

	types, err := discoveryCSV(values["type"], discoveryTypeNames, "type")
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}
	statuses, err := discoveryStatuses(values["versioningStatus"])
	if err != nil {
		return domain.DiscoveryFilter{}, err
	}

	return domain.DiscoveryFilter{
		Query:              strings.TrimSpace(values.Get("q")),
		Types:              types,
		SpatialFilter:      strings.TrimSpace(values.Get("spatialFilter")),
		DateFrom:           dateFrom,
		DateTo:             dateTo,
		Provider:           strings.Join(splitDiscoveryCSV(values["provider"]), ","),
		Keywords:           strings.Join(splitDiscoveryCSV(values["keywords"]), ","),
		VersioningStatuses: statuses,
	}, nil
}

var discoveryTypeNames = map[string]struct{}{
	"dataset":              {},
	"software-application": {},
	"software-source-code": {},
}

// TODO: use a domain level enum
var discoveryStatusNames = map[string]struct{}{
	"PUBLISHED": {},
	"DRAFT":     {},
	"SUBMITTED": {},
	"ARCHIVED":  {},
	"REJECTED":  {},
}

func parseDiscoveryTime(value, name string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339", name)
	}
	return &parsed, nil
}

func discoveryCSV(values []string, allowed map[string]struct{}, name string) ([]string, error) {
	result := splitDiscoveryCSV(values)
	for _, value := range result {
		if _, ok := allowed[value]; !ok {
			return nil, fmt.Errorf("unsupported %s %q", name, value)
		}
	}
	return result, nil
}

func discoveryStatuses(values []string) ([]string, error) {
	rawStatuses := splitDiscoveryCSV(values)
	if len(rawStatuses) == 0 {
		// TODO: use an enum
		return []string{"PUBLISHED"}, nil
	}
	statuses := make([]string, 0, len(rawStatuses))
	for _, rawStatus := range rawStatuses {
		status := strings.ToUpper(rawStatus)
		if _, ok := discoveryStatusNames[status]; !ok {
			return nil, fmt.Errorf("unsupported versioningStatus %q", rawStatus)
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func splitDiscoveryCSV(values []string) []string {
	var result []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}

func discoveryItemFromDomain(item domain.DiscoveryItem) api.DiscoveryItem {
	return api.DiscoveryItem{
		Type:          item.Type,
		ResourceID:    item.ResourceID,
		VersionID:     item.VersionID,
		SourceIRI:     item.SourceIRI,
		Name:          item.Name,
		Status:        item.Status,
		Group:         discoveryReferenceFromDomain(item.Group),
		Providers:     discoveryReferencesFromDomain(item.Providers),
		Creators:      discoveryReferencesFromDomain(item.Creators),
		Categories:    discoveryReferencesFromDomain(item.Categories),
		Distributions: discoveryReferencesFromDomain(item.Distributions),
	}
}

func discoveryReferencesFromDomain(values []domain.DiscoveryReference) []api.Reference {
	if values == nil {
		return nil
	}
	result := make([]api.Reference, 0, len(values))
	for _, value := range values {
		result = append(result, discoveryReferenceFromDomain(value))
	}
	return result
}

func discoveryReferenceFromDomain(value domain.DiscoveryReference) api.Reference {
	return api.Reference{ID: value.ID, Name: value.Name}
}

func writeDiscoveryError(r *http.Request, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrBadRequest):
		WriteError(r.Context(), w, http.StatusBadRequest, "invalid_request", "invalid discovery request", err)
	case errors.Is(err, domain.ErrUnauthorized):
		WriteError(r.Context(), w, http.StatusUnauthorized, "unauthorized", "authentication is required for non-published discovery results", nil)
	case errors.Is(err, domain.ErrForbidden):
		WriteError(r.Context(), w, http.StatusForbidden, "forbidden", "discovery access denied", nil)
	case errors.Is(err, domain.ErrUpstreamTimeout):
		WriteError(r.Context(), w, http.StatusGatewayTimeout, "upstream_timeout", "discovery request timed out", err)
	default:
		log.ErrorContext(r.Context(), "discovery request failed", "error", err)
		WriteError(r.Context(), w, http.StatusServiceUnavailable, "backoffice_unavailable", "backoffice service unavailable", err)
	}
}
