package clients

import (
	"context"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
)

func (c *MetadataLifecycle) SearchResources(ctx context.Context, actorID string, request *metadatalifecyclev1.SearchResourcesRequest) (*metadatalifecyclev1.SearchResourcesResponse, error) {
	if c == nil || c.GRPC == nil {
		return nil, fmt.Errorf("metadata-lifecycle gRPC client is not configured")
	}

	return c.GRPC.SearchResources(ctx, actorID, request)
}
