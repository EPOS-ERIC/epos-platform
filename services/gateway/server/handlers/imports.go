package handlers

import (
	"context"
	"fmt"
	"io"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) ImportTurtleMetadata(ctx context.Context, body gatewayapi.ImportTurtleMetadataReq) (*gatewayapi.ImportResult, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read import body: %w", err)
	}
	v, err := h.metadataLifecycleClient.ImportTurtle(ctx, actor, importTurtleRequest(data))
	if err != nil {
		return nil, fmt.Errorf("import turtle metadata: %w", err)
	}
	return toGatewayImportResult(v)
}

func importTurtleRequest(data []byte) *metadatalifecyclev1.ImportTurtleRequest {
	return &metadatalifecyclev1.ImportTurtleRequest{Turtle: data}
}
