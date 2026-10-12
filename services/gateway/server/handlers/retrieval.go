package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) RetrieveResourceVersion(ctx context.Context, request *gatewayapi.RetrieveResourceVersionRequest) (*gatewayapi.RetrieveResourceVersionResponse, error) {
	protoRequest, err := toProtoRetrieveResourceVersionRequest(request)
	if err != nil {
		return nil, err
	}
	stream, err := h.dataRetriever.RetrieveStream(ctx, protoRequest)
	if err != nil {
		return nil, err
	}
	event, err := stream.Recv(ctx)
	if err != nil {
		return nil, err
	}
	if event.GetHeaders() == nil || event.GetHeaders().GetCurlCommand() == "" {
		return nil, apperror.Default(apperror.CodeInternal)
	}
	return toAPIDryResponse(event.GetHeaders())
}
