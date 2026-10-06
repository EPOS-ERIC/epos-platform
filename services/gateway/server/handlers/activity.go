package handlers

import (
	"context"
	"fmt"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) ListActivity(ctx context.Context, params gatewayapi.ListActivityParams) ([]gatewayapi.ActivityEvent, error) {
	actor, _ := authcontext.ActorID(ctx)
	response, err := h.metadataLifecycle.ListActivity(ctx, actor, toProtoListActivityRequest(params))
	if err != nil {
		return nil, fmt.Errorf("list activity: %w", err)
	}

	if response == nil {
		return []gatewayapi.ActivityEvent{}, nil
	}

	items := make([]gatewayapi.ActivityEvent, 0, len(response.Items))
	for _, item := range response.Items {
		mapped, err := toAPIActivityEvent(item)
		if err != nil {
			return nil, err
		}
		items = append(items, mapped)
	}

	return items, nil
}
