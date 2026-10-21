package handlers

import (
	"context"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) AccessDistribution(ctx context.Context, request *gatewayapi.DistributionAccessRequest, params gatewayapi.AccessDistributionParams) (*gatewayapi.AccessDistribution2XXStatusCode, error) {
	input, err := fromAPIDistributionAccess(request, params.DistributionId, params.OutputFormat, params.ConversionID)
	if err != nil {
		return nil, err
	}

	actorID, _ := authcontext.ActorID(ctx)

	result, err := h.distributionAccess.Access(ctx, actorID, input)
	if err != nil {
		return nil, err
	}

	result, err = h.prepareRetrievalResponse(ctx, result)
	if err != nil {
		return nil, err
	}

	response := &gatewayapi.AccessDistribution2XXStatusCode{StatusCode: result.StatusCode}
	if result.Body != nil {
		response.Response.Data = result.Body
	}

	return response, nil
}

func (h *ogenHandler) PrepareDistribution(ctx context.Context, request *gatewayapi.DistributionAccessRequest, params gatewayapi.PrepareDistributionParams) (*gatewayapi.PrepareDistributionOKHeaders, error) {
	input, err := fromAPIDistributionAccess(request, params.DistributionId, params.OutputFormat, params.ConversionID)
	if err != nil {
		return nil, err
	}

	actorID, _ := authcontext.ActorID(ctx)

	prepared, err := h.distributionAccess.Prepare(ctx, actorID, input)
	if err != nil {
		return nil, err
	}

	response, err := toAPIPreparedDistributionRequest(prepared)
	if err != nil {
		return nil, err
	}

	return &gatewayapi.PrepareDistributionOKHeaders{
		CacheControl: gatewayapi.PrepareDistributionOKCacheControlNoStore,
		Response:     response,
	}, nil
}
