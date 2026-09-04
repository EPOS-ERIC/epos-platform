package clients

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const actorMetadataKey = "x-epos-user-id"

func NewMetadataLifecycleGRPC(log *slog.Logger, baseURL string) (*MetadataLifecycleGRPC, *MetadataLifecycleIdentity, error) {
	conn, err := grpc.NewClient(
		baseURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create metadata-lifecycle gRPC client: %w", err)
	}

	proxy := &MetadataLifecycleGRPC{
		userGRPC:         metadatalifecyclev1.NewUserServiceClient(conn),
		groupGRPC:        metadatalifecyclev1.NewGroupServiceClient(conn),
		categoryGRPC:     metadatalifecyclev1.NewCategoryServiceClient(conn),
		contactPointGRPC: metadatalifecyclev1.NewContactPointServiceClient(conn),
		organizationGRPC: metadatalifecyclev1.NewOrganizationServiceClient(conn),
	}
	identity := &MetadataLifecycleIdentity{
		identityGRPC: metadatalifecyclev1.NewIdentityServiceClient(conn),
		proxy:        proxy,
	}

	return proxy, identity, nil
}

func wrapGRPCError(operation string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	}

	switch status.Code(err) {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return fmt.Errorf("%s: %w", operation, domain.ErrBadRequest)
	case codes.NotFound:
		return fmt.Errorf("%s: %w", operation, domain.ErrNotFound)
	case codes.AlreadyExists, codes.Aborted:
		return fmt.Errorf("%s: %w", operation, domain.ErrConflict)
	case codes.PermissionDenied:
		return fmt.Errorf("%s: %w", operation, domain.ErrForbidden)
	case codes.Unauthenticated:
		return fmt.Errorf("%s: %w", operation, domain.ErrUnauthorized)
	case codes.DeadlineExceeded, codes.Canceled:
		return fmt.Errorf("%s: %w", operation, domain.ErrUpstreamTimeout)
	default:
		// Internal, Unavailable, Unknown, ResourceExhausted, Unimplemented.
		// Gateway handlers already turn unknown upstream failures into 503.
		return fmt.Errorf("%s: %w", operation, err)
	}
}

func (c *MetadataLifecycleGRPC) GrantGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	_, err := c.userGRPC.GrantGlobalPermission(ctx, &metadatalifecyclev1.GrantGlobalPermissionRequest{
		UserId:     userID,
		Permission: permission,
	})
	if err != nil {
		return wrapGRPCError("grant metadata-lifecycle global permission", err)
	}

	return nil
}

func (c *MetadataLifecycleGRPC) RevokeGlobalPermission(ctx context.Context, actorID, userID, permission string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	_, err := c.userGRPC.RevokeGlobalPermission(ctx, &metadatalifecyclev1.RevokeGlobalPermissionRequest{
		UserId:     userID,
		Permission: permission,
	})
	if err != nil {
		return wrapGRPCError("revoke metadata-lifecycle global permission", err)
	}

	return nil
}

func (c *MetadataLifecycleGRPC) AssignGlobalRole(ctx context.Context, actorID, userID, role string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	_, err := c.userGRPC.AssignGlobalRole(ctx, &metadatalifecyclev1.AssignGlobalRoleRequest{
		UserId: userID,
		Role:   role,
	})
	if err != nil {
		return wrapGRPCError("assign metadata-lifecycle global role", err)
	}

	return nil
}

func (c *MetadataLifecycleGRPC) RemoveGlobalRole(ctx context.Context, actorID, userID, role string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	_, err := c.userGRPC.RemoveGlobalRole(ctx, &metadatalifecyclev1.RemoveGlobalRoleRequest{
		UserId: userID,
		Role:   role,
	})
	if err != nil {
		return wrapGRPCError("remove metadata-lifecycle global role", err)
	}

	return nil
}

func (c *MetadataLifecycleGRPC) BlockUser(ctx context.Context, actorID, userID, reason string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	_, err := c.userGRPC.BlockUser(ctx, &metadatalifecyclev1.BlockUserRequest{
		UserId: userID,
		Reason: reason,
	})
	if err != nil {
		return wrapGRPCError("block metadata-lifecycle user", err)
	}

	return nil
}

func (c *MetadataLifecycleGRPC) UnblockUser(ctx context.Context, actorID, userID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)

	_, err := c.userGRPC.UnblockUser(ctx, &metadatalifecyclev1.UnblockUserRequest{
		UserId: userID,
	})
	if err != nil {
		return wrapGRPCError("unblock metadata-lifecycle user", err)
	}

	return nil
}
