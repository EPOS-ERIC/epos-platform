package clients

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"google.golang.org/grpc/metadata"
)

func (c *MetadataLifecycle) SearchResources(ctx context.Context, actorID string, request *metadatalifecyclev1.SearchResourcesRequest) (*metadatalifecyclev1.SearchResourcesResponse, error) {
	if actorID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	}

	response, err := c.discovery.SearchResources(ctx, request)
	if err != nil {
		return nil, c.wrapError(ctx, "search metadata-lifecycle resources", err)
	}
	return response, nil
}
