package clients

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"google.golang.org/grpc/metadata"
)

func (c *MetadataLifecycle) IssueUserAPIToken(ctx context.Context, actorID string, rotate bool) (string, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	if rotate {
		response, err := c.users.RotateUserAPIToken(ctx, &metadatalifecyclev1.RotateUserAPITokenRequest{})
		if err != nil {
			return "", c.wrapError(ctx, "issue or rotate user API token", err)
		}

		return response.GetToken(), nil
	}

	response, err := c.users.IssueUserAPIToken(ctx, &metadatalifecyclev1.IssueUserAPITokenRequest{})
	if err != nil {
		return "", c.wrapError(ctx, "issue or rotate user API token", err)
	}

	return response.GetToken(), nil
}

func (c *MetadataLifecycle) RevokeUserAPIToken(ctx context.Context, actorID, userID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	_, err := c.users.RevokeUserAPIToken(ctx, &metadatalifecyclev1.RevokeUserAPITokenRequest{UserId: userID})
	if err != nil {
		return c.wrapError(ctx, "revoke user API token", err)
	}

	return nil
}

func (c *MetadataLifecycle) ValidateUserAPIToken(ctx context.Context, digest []byte) (string, error) {
	response, err := c.users.ValidateUserAPIToken(ctx, &metadatalifecyclev1.ValidateUserAPITokenRequest{Digest: digest})
	if err != nil {
		return "", c.wrapError(ctx, "validate user API token", err)
	}

	return response.GetUserId(), nil
}
