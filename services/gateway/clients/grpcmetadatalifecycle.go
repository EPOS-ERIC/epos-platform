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
	"google.golang.org/grpc/status"
)

type GRPCMetadataLifecycle struct {
	identityGRPC metadatalifecyclev1.IdentityServiceClient
	grpcConn     *grpc.ClientConn

	log *slog.Logger
}

func NewGRPCMetadataLifecycle(log *slog.Logger, baseURL string) (*GRPCMetadataLifecycle, error) {
	conn, err := grpc.NewClient(
		baseURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("create metadata-lifecycle gRPC client: %w", err)
	}

	return &GRPCMetadataLifecycle{
		identityGRPC: metadatalifecyclev1.NewIdentityServiceClient(conn),
		grpcConn:     conn,
		log:          log.With("component", "grpc-server"),
	}, nil
}

func (c *GRPCMetadataLifecycle) ProvisionUser(ctx context.Context, identity ActorIdentity) (domain.User, error) {
	user, err := c.identityGRPC.ProvisionUser(ctx, &metadatalifecyclev1.ProvisionUserRequest{
		Id:         identity.AuthIdentifier,
		Email:      identity.Email,
		GivenName:  identity.GivenName,
		FamilyName: identity.FamilyName,
	})
	if err != nil {
		return domain.User{}, wrapGRPCError("provision metadata-lifecycle user", err)
	}

	return domain.User{
		ID:         user.Id,
		Email:      user.Email,
		GivenName:  user.GivenName,
		FamilyName: user.FamilyName,
	}, nil
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
