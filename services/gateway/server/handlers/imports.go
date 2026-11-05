package handlers

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) ImportTurtleMetadata(ctx context.Context, request *gatewayapi.ImportResourcesRequest) (*gatewayapi.ImportResourcesResult, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	protoRequest, err := toProtoImportResourcesRequest(request)
	if err != nil {
		return nil, fmt.Errorf("map resource import request: %w", err)
	}
	response, err := h.semanticAggregator.ImportResources(ctx, actor, protoRequest)
	if err != nil {
		return nil, fmt.Errorf("import resources: %w", err)
	}
	return toAPIImportResourcesResult(response)
}
