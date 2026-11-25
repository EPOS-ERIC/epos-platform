package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) RequestResourceContact(ctx context.Context, request *gatewayapi.ContactRequestMessage, params gatewayapi.RequestResourceContactParams) (gatewayapi.RequestResourceContactRes, error) {
	actorID, _ := authcontext.ActorID(ctx)
	if err := h.contactRequest.RequestResource(ctx, actorID, domain.ResourceType(params.ResourceType), params.ID, request.Message); err != nil {
		return nil, err
	}

	return &gatewayapi.RequestResourceContactNoContent{}, nil
}

func (h *ogenHandler) RequestDistributionContact(ctx context.Context, request *gatewayapi.ContactRequestMessage, params gatewayapi.RequestDistributionContactParams) (gatewayapi.RequestDistributionContactRes, error) {
	actorID, _ := authcontext.ActorID(ctx)
	if err := h.contactRequest.RequestDistribution(ctx, actorID, params.DistributionId, request.Message); err != nil {
		return nil, err
	}

	return &gatewayapi.RequestDistributionContactNoContent{}, nil
}
