//nolint:dupl,exhaustruct_v5,wrapcheck
package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) DiscoverResources(ctx context.Context, q gatewayapi.DiscoverResourcesParams) ([]gatewayapi.DiscoveryItem, error) {
	actor, _ := actorID(ctx)
	filter := domain.DiscoveryFilter{}
	if q.Q.Set {
		filter.Query = q.Q.Value
	}
	if q.GroupId.Set {
		filter.GroupID = q.GroupId.Value.String()
	}
	if q.EditorId.Set {
		filter.EditorID = q.EditorId.Value
	}
	if q.SpatialFilter.Set {
		filter.SpatialFilter = q.SpatialFilter.Value
	}
	for _, x := range q.Type {
		filter.Types = append(filter.Types, string(x))
	}
	for _, x := range q.Concept {
		filter.Concepts = append(filter.Concepts, string(x))
	}
	for _, x := range q.Provider {
		if filter.Provider != "" {
			filter.Provider += ","
		}
		filter.Provider += x.String()
	}
	for _, x := range q.Keywords {
		if filter.Keywords != "" {
			filter.Keywords += ","
		}
		filter.Keywords += x
	}
	for _, x := range q.VersioningStatus {
		filter.VersioningStatuses = append(filter.VersioningStatuses, string(x))
	}
	if len(filter.VersioningStatuses) == 0 {
		filter.VersioningStatuses = []string{"PUBLISHED"}
	}
	if q.DateFrom.Set {
		filter.DateFrom = &q.DateFrom.Value
	}
	if q.DateTo.Set {
		filter.DateTo = &q.DateTo.Value
	}
	v, err := h.discovery.Search(ctx, actor, filter)
	if err != nil {
		return nil, err
	}
	out := []gatewayapi.DiscoveryItem{}
	for _, x := range v {
		mapped, err := mapDiscovery(x)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}
