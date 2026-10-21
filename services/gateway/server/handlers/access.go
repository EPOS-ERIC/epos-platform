package handlers

import (
	"context"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) AccessDistribution(ctx context.Context, request *gatewayapi.DistributionAccessRequest, params gatewayapi.AccessDistributionParams) (*gatewayapi.AccessDistribution2XXStatusCode, error) {
	input, err := fromAPIDistributionAccess(request, params)
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
