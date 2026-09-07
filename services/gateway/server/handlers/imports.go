package handlers

import (
	"context"
	"fmt"
	"io"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) ImportTurtleMetadata(ctx context.Context, body gatewayapi.ImportTurtleMetadataReq) (*gatewayapi.ImportResult, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read import body: %w", err)
	}
	v, err := h.metadataLifecycle.ImportTurtle(ctx, actor, importTurtleRequest(data))
	if err != nil {
		return nil, fmt.Errorf("import turtle metadata: %w", err)
	}
	return toGatewayImportResult(v)
}

func importTurtleRequest(data []byte) *metadatalifecyclev1.ImportTurtleRequest {
	return &metadatalifecyclev1.ImportTurtleRequest{Turtle: data}
}
