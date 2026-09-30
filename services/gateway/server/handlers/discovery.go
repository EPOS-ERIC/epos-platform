package handlers

import (
	"context"
	"fmt"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) DiscoverResources(ctx context.Context, q gatewayapi.DiscoverResourcesParams) ([]gatewayapi.DiscoveryItem, error) {
	actor, _ := authcontext.ActorID(ctx)
	request, err := toProtoSearchResourcesRequest(q)
	if err != nil {
		return nil, err
	}
	response, err := h.metadataLifecycle.SearchResources(ctx, actor, request)
	if err != nil {
		return nil, fmt.Errorf("search resources: %w", err)
	}
	if response == nil {
		return []gatewayapi.DiscoveryItem{}, nil
	}

	out := make([]gatewayapi.DiscoveryItem, 0, len(response.Items))
	for _, item := range response.Items {
		mapped, err := toAPIDiscovery(item)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}
