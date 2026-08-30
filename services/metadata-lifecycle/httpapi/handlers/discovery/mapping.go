package discovery

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func filterFromRequest(r *http.Request) (domain.DiscoveryFilter, error) {
	values := r.URL.Query()
	statuses, err := parseStatuses(split(values["versioningStatus"]))
	if err != nil {
		return domain.DiscoveryFilter{}, fmt.Errorf("parse versioningStatus: %w", err)
	}
	types, err := parseTypes(split(values["type"]))
	if err != nil {
		return domain.DiscoveryFilter{}, fmt.Errorf("parse type: %w", err)
	}
	var groupID *uuid.UUID
	if value := values.Get("groupId"); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return domain.DiscoveryFilter{}, fmt.Errorf("parse groupId: %w", err)
		}
		groupID = &parsed
	}
	providers, err := parseUUIDs(split(values["provider"]))
	if err != nil {
		return domain.DiscoveryFilter{}, fmt.Errorf("parse provider: %w", err)
	}
	from, to, err := parseTimes(values)
	if err != nil {
		return domain.DiscoveryFilter{}, fmt.Errorf("parse date range: %w", err)
	}
	var spatialFilter *domain.SpatialFilter
	if value := values.Get("spatialFilter"); value != "" {
		spatialFilter, err = domain.NewSpatialFilter(value)
		if err != nil {
			return domain.DiscoveryFilter{}, fmt.Errorf("parse spatialFilter: %w", err)
		}
	}

	filter, err := domain.NewDiscoveryFilter(domain.DiscoveryFilter{
		Query:              new(values.Get("q")),
		Types:              types,
		Concepts:           parseConcepts(split(values["concept"])),
		GroupID:            groupID,
		SpatialFilter:      spatialFilter,
		DateFrom:           from,
		DateTo:             to,
		Providers:          providers,
		Keywords:           split(values["keywords"]),
		EditorID:           new(values.Get("editorId")),
		VersioningStatuses: statuses,
	})
	if err != nil {
		return domain.DiscoveryFilter{}, fmt.Errorf("validate discovery filter: %w", err)
	}
	return filter, nil
}

func parseConcepts(values []string) []domain.TopLevelConcept {
	result := make([]domain.TopLevelConcept, 0, len(values))
	for _, value := range values {
		result = append(result, domain.TopLevelConcept(value))
	}
	return result
}

func parseStatuses(values []string) ([]domain.Status, error) {
	result := make([]domain.Status, 0, len(values))
	seen := map[domain.Status]bool{}
	for _, value := range values {
		status := domain.Status(strings.ToUpper(strings.TrimSpace(value)))
		switch status {
		case domain.Published, domain.Draft, domain.Submitted, domain.Archived, domain.Rejected:
		default:
			return nil, fmt.Errorf("unsupported versioningStatus %q: %w", value, domain.ErrInvalidDiscovery)
		}
		if !seen[status] {
			result = append(result, status)
			seen[status] = true
		}
	}
	return result, nil
}

func parseTypes(values []string) ([]domain.ResourceType, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]domain.ResourceType, 0, len(values))
	seen := map[domain.ResourceType]bool{}
	for _, value := range values {
		resourceType := domain.ResourceType(strings.TrimSpace(value))
		if resourceType == "" {
			continue
		}
		switch resourceType {
		case domain.Dataset, domain.SoftwareApplication, domain.SoftwareSourceCode:
		default:
			return nil, fmt.Errorf("unsupported resource type %q: %w", value, domain.ErrInvalidDiscovery)
		}
		if !seen[resourceType] {
			result = append(result, resourceType)
			seen[resourceType] = true
		}
	}
	return result, nil
}

func parseUUIDs(values []string) ([]uuid.UUID, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("invalid UUID %q: %w", value, domain.ErrInvalidDiscovery)
		}
		result = append(result, id)
	}
	return result, nil
}

func parseTimes(values url.Values) (*time.Time, *time.Time, error) {
	parse := func(key string) (*time.Time, error) {
		value := strings.TrimSpace(values.Get(key))
		if value == "" {
			return nil, nil
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", key, domain.ErrInvalidDiscovery)
		}
		return &parsed, nil
	}
	from, err := parse("dateFrom")
	if err != nil {
		return nil, nil, err
	}
	to, err := parse("dateTo")
	if err != nil {
		return nil, nil, err
	}
	return from, to, nil
}

func split(values []string) []string {
	var result []string
	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			if value := strings.TrimSpace(part); value != "" {
				result = append(result, value)
			}
		}
	}
	return result
}

func toResponse(values []domain.DiscoveryItem) []discoveryItemResponse {
	result := make([]discoveryItemResponse, 0, len(values))
	for _, value := range values {
		result = append(result, discoveryItemResponse{
			Type:          value.Type,
			ResourceID:    value.ResourceID,
			VersionID:     value.VersionID,
			Name:          value.Name,
			Status:        value.Status,
			Group:         discoveryReferenceResponse(value.Group),
			Providers:     toReferenceResponses(value.Providers),
			Creators:      toReferenceResponses(value.Creators),
			Categories:    toReferenceResponses(value.Categories),
			Distributions: toReferenceResponses(value.Distributions),
		})
	}
	return result
}

func toReferenceResponses(values []domain.DiscoveryReference) []discoveryReferenceResponse {
	if values == nil {
		return nil
	}
	result := make([]discoveryReferenceResponse, 0, len(values))
	for _, value := range values {
		result = append(result, discoveryReferenceResponse(value))
	}
	return result
}
