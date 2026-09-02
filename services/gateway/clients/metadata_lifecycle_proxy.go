package clients

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"google.golang.org/grpc/metadata"
)

type MetadataLifecycleProxy struct {
	userGRPC metadatalifecyclev1.UserServiceClient
}

func (c *MetadataLifecycleProxy) GetUser(ctx context.Context, actorID, id string) (*metadatalifecyclev1.GetUserResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	user, err := c.userGRPC.GetUser(ctx, &metadatalifecyclev1.GetUserRequest{
		UserId: id,
	})
	if err != nil {
		return nil, wrapGRPCError("get metadata-lifecycle user", err)
	}

	return user, nil
}
