package clients

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"google.golang.org/grpc/metadata"
)

func (c *MetadataLifecycle) ListActivity(ctx context.Context, actorID string, request *metadatalifecyclev1.ListActivityRequest) (*metadatalifecyclev1.ListActivityResponse, error) {
	if actorID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	}

	response, err := c.activity.ListActivity(ctx, request)
	if err != nil {
		return nil, c.wrapError(ctx, "list metadata-lifecycle activity", err)
	}

	return response, nil
}
