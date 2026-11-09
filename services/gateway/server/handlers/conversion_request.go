package handlers

import (
	"context"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) RequestDistributionConversion(ctx context.Context, request *gatewayapi.RequestDistributionConversionReq, params gatewayapi.RequestDistributionConversionParams) error {
	actorID, _ := authcontext.ActorID(ctx)
	return h.metadataLifecycleClient.RequestConversion(ctx, actorID, toProtoConversionRequest(request, params.DistributionId))
}
