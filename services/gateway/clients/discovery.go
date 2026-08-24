package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	backofficeclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/backoffice"
)

func (c *Backoffice) SearchResources(ctx context.Context, actorID string, filter domain.DiscoveryFilter) ([]domain.DiscoveryItem, error) {
	if c == nil || c.resources == nil {
		return nil, fmt.Errorf("backoffice resource client is not configured")
	}

	request := c.resources.SearchResources(ctx)
	if actorID != "" {
		request = request.XEPOSUserID(actorID)
	}
	if filter.Query != "" {
		request = request.Q(filter.Query)
	}
	if len(filter.Types) != 0 {
		types := make([]backofficeclient.ResourceType, 0, len(filter.Types))
		for _, resourceType := range filter.Types {
			types = append(types, backofficeclient.ResourceType(resourceType))
		}
		request = request.Type_(types)
	}
	if filter.SpatialFilter != "" {
		request = request.SpatialFilter(filter.SpatialFilter)
	}
	if filter.DateFrom != nil {
		request = request.DateFrom(*filter.DateFrom)
	}
	if filter.DateTo != nil {
		request = request.DateTo(*filter.DateTo)
	}
	if filter.Provider != "" {
		request = request.Provider(filter.Provider)
	}
	if filter.Keywords != "" {
		request = request.Keywords(filter.Keywords)
	}
	if len(filter.VersioningStatuses) != 0 {
		request = request.VersioningStatus(filter.VersioningStatuses)
	}

	items, response, err := request.Execute()
	if err != nil {
		return nil, wrapDiscoveryError("search backoffice resources", response, err)
	}

	result := make([]domain.DiscoveryItem, 0, len(items))
	for _, item := range items {
		result = append(result, discoveryItemFromClient(item))
	}
	return result, nil
}

func discoveryItemFromClient(item backofficeclient.DiscoveryItem) domain.DiscoveryItem {
	return domain.DiscoveryItem{
		Type:          item.Type,
		ResourceID:    item.ResourceId,
		VersionID:     item.VersionId,
		SourceIRI:     item.SourceIri,
		Name:          item.Name,
		Status:        string(item.Status),
		Group:         discoveryReferenceFromClient(item.Group),
		Providers:     discoveryReferencesFromClient(item.Providers),
		Creators:      discoveryReferencesFromClient(item.Creators),
		Categories:    discoveryReferencesFromClient(item.Categories),
		Distributions: discoveryReferencesFromClient(item.Distributions),
	}
}

func discoveryReferencesFromClient(values []backofficeclient.DiscoveryReference) []domain.DiscoveryReference {
	if values == nil {
		return nil
	}

	result := make([]domain.DiscoveryReference, 0, len(values))
	for _, value := range values {
		result = append(result, discoveryReferenceFromClient(value))
	}
	return result
}

func discoveryReferenceFromClient(value backofficeclient.DiscoveryReference) domain.DiscoveryReference {
	return domain.DiscoveryReference{ID: value.Id, Name: value.Name}
}

func wrapDiscoveryError(operation string, response *http.Response, err error) error {
	if response != nil {
		switch response.StatusCode {
		case http.StatusBadRequest:
			return fmt.Errorf("%s: %w", operation, domain.ErrBadRequest)
		case http.StatusUnauthorized:
			return fmt.Errorf("%s: %w", operation, domain.ErrUnauthorized)
		case http.StatusForbidden:
			return fmt.Errorf("%s: %w", operation, domain.ErrForbidden)
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
