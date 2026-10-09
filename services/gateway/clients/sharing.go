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

const (
	sharingUserIDMetadataKey = "x-epos-user-id"
	sharingManageMetadataKey = "x-epos-configuration-manage"
)

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

func withCaller(ctx context.Context, actor string, manage bool) context.Context {
	if actor == "" {
		return ctx
	}
	ctx = metadata.AppendToOutgoingContext(ctx, sharingUserIDMetadataKey, actor)
	if manage {
		ctx = metadata.AppendToOutgoingContext(ctx, sharingManageMetadataKey, "true")
	}
	return ctx
}

func (c *Sharing) wrapError(ctx context.Context, operation string, err error) error {
	return wrapRPCError(c.log, ctx, "sharing", operation, err)
}

func (c *Sharing) CreateConfiguration(ctx context.Context, actor string, manage bool, request *configurationv1.CreateConfigurationRequest) (*configurationv1.CreateConfigurationResponse, error) {
	response, err := c.configurations.CreateConfiguration(withCaller(ctx, actor, manage), request)
	if err != nil {
		return nil, c.wrapError(ctx, "create configuration", err)
	}
	return response, nil
}

func (c *Sharing) GetConfiguration(ctx context.Context, actor string, manage bool, request *configurationv1.GetConfigurationRequest) (*configurationv1.GetConfigurationResponse, error) {
	response, err := c.configurations.GetConfiguration(withCaller(ctx, actor, manage), request)
	if err != nil {
		return nil, c.wrapError(ctx, "get configuration", err)
	}
	return response, nil
}

func (c *Sharing) ListConfigurations(ctx context.Context, actor string, manage bool, request *configurationv1.ListConfigurationsRequest) (*configurationv1.ListConfigurationsResponse, error) {
	response, err := c.configurations.ListConfigurations(withCaller(ctx, actor, manage), request)
	if err != nil {
		return nil, c.wrapError(ctx, "list configurations", err)
	}
	return response, nil
}

func (c *Sharing) UpdateConfiguration(ctx context.Context, actor string, manage bool, request *configurationv1.UpdateConfigurationRequest) (*configurationv1.UpdateConfigurationResponse, error) {
	response, err := c.configurations.UpdateConfiguration(withCaller(ctx, actor, manage), request)
	if err != nil {
		return nil, c.wrapError(ctx, "replace configuration", err)
	}
	return response, nil
}

func (c *Sharing) DeleteConfiguration(ctx context.Context, actor string, manage bool, request *configurationv1.DeleteConfigurationRequest) (*configurationv1.DeleteConfigurationResponse, error) {
	response, err := c.configurations.DeleteConfiguration(withCaller(ctx, actor, manage), request)
	if err != nil {
		return nil, c.wrapError(ctx, "delete configuration", err)
	}
	return response, nil
}

func (c *Sharing) SetConfigurationOfficialStatus(ctx context.Context, actor string, manage bool, request *configurationv1.SetConfigurationOfficialStatusRequest) (*configurationv1.SetConfigurationOfficialStatusResponse, error) {
	response, err := c.configurations.SetConfigurationOfficialStatus(withCaller(ctx, actor, manage), request)
	if err != nil {
		return nil, c.wrapError(ctx, "set configuration official status", err)
	}
	return response, nil
}
