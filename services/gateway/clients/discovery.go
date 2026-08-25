package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
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
		request = request.Type_(toClientResourceTypes(filter.Types))
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

	return mapSlice(items, toDomainDiscoveryItem), nil
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
