package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	client "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

type SearchResourcesInput struct {
	Query         string
	Types         []string
	Concepts      []string
	GroupID       string
	SpatialFilter string
	DateFrom      *time.Time
	DateTo        *time.Time
	Provider      string
	// TODO: this should become an array of strings
	Keywords           string
	EditorID           string
	VersioningStatuses []string
}

func (c *MetadataLifecycle) SearchResources(ctx context.Context, actorID string, filter SearchResourcesInput) ([]client.DiscoveryItem, error) {
	if c == nil || c.resources == nil {
		return nil, fmt.Errorf("metadata-lifecycle resource client is not configured")
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
	if len(filter.Concepts) != 0 {
		request = request.Concept(filter.Concepts)
	}
	if filter.GroupID != "" {
		request = request.GroupId(filter.GroupID)
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
	if filter.EditorID != "" {
		request = request.EditorId(filter.EditorID)
	}
	if len(filter.VersioningStatuses) != 0 {
		request = request.VersioningStatus(filter.VersioningStatuses)
	}

	items, response, err := request.Execute()
	if err != nil {
		c.log.ErrorContext(ctx, "search metadata-lifecycle resources", "error", err)
		return nil, wrapDiscoveryError("search metadata-lifecycle resources", response, err)
	}

	return items, nil
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
