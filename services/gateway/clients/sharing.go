package clients

import (
	"context"
	"fmt"
	"log/slog"

	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Metadata keys carrying the caller's identity to the sharing service. The
// actor is attached after gateway bearer validation; anonymous callers send
// no identity and may only access public configurations.
const (
	sharingUserIDMetadataKey = "x-epos-user-id"
	sharingAdminMetadataKey  = "x-epos-admin"
)

// Sharing is a Gateway-owned adapter for the upstream sharing service. Pure
// proxy calls return generated protobuf responses unchanged; handlers map
// them to Gateway API types.
type Sharing struct {
	log            *slog.Logger
	configurations configurationv1.ConfigurationServiceClient
}

func NewSharing(log *slog.Logger, baseURL string) (*Sharing, error) {
	conn, err := grpc.NewClient(
		baseURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("create sharing gRPC client: %w", err)
	}
	return &Sharing{
		log:            log,
		configurations: configurationv1.NewConfigurationServiceClient(conn),
	}, nil
}

// withCaller attaches the caller's identity. An empty actor means anonymous:
// no identity metadata is sent and the sharing service applies public-only
// visibility.
func withCaller(ctx context.Context, actor string, admin bool) context.Context {
	if actor == "" {
		return ctx
	}
	ctx = metadata.AppendToOutgoingContext(ctx, sharingUserIDMetadataKey, actor)
	if admin {
		ctx = metadata.AppendToOutgoingContext(ctx, sharingAdminMetadataKey, "true")
	}
	return ctx
}

func (c *Sharing) CreateConfiguration(ctx context.Context, actor string, admin bool, request *configurationv1.CreateConfigurationRequest) (*configurationv1.CreateConfigurationResponse, error) {
	response, err := c.configurations.CreateConfiguration(withCaller(ctx, actor, admin), request)
	if err != nil {
		return nil, wrapRPCError(c.log, ctx, "sharing", "create configuration", err)
	}
	return response, nil
}

func (c *Sharing) GetConfiguration(ctx context.Context, actor string, admin bool, request *configurationv1.GetConfigurationRequest) (*configurationv1.GetConfigurationResponse, error) {
	response, err := c.configurations.GetConfiguration(withCaller(ctx, actor, admin), request)
	if err != nil {
		return nil, wrapRPCError(c.log, ctx, "sharing", "get configuration", err)
	}
	return response, nil
}

func (c *Sharing) ListConfigurations(ctx context.Context, actor string, admin bool, request *configurationv1.ListConfigurationsRequest) (*configurationv1.ListConfigurationsResponse, error) {
	response, err := c.configurations.ListConfigurations(withCaller(ctx, actor, admin), request)
	if err != nil {
		return nil, wrapRPCError(c.log, ctx, "sharing", "list configurations", err)
	}
	return response, nil
}

func (c *Sharing) UpdateConfiguration(ctx context.Context, actor string, admin bool, request *configurationv1.UpdateConfigurationRequest) (*configurationv1.UpdateConfigurationResponse, error) {
	response, err := c.configurations.UpdateConfiguration(withCaller(ctx, actor, admin), request)
	if err != nil {
		return nil, wrapRPCError(c.log, ctx, "sharing", "replace configuration", err)
	}
	return response, nil
}

func (c *Sharing) DeleteConfiguration(ctx context.Context, actor string, admin bool, request *configurationv1.DeleteConfigurationRequest) (*configurationv1.DeleteConfigurationResponse, error) {
	response, err := c.configurations.DeleteConfiguration(withCaller(ctx, actor, admin), request)
	if err != nil {
		return nil, wrapRPCError(c.log, ctx, "sharing", "delete configuration", err)
	}
	return response, nil
}
