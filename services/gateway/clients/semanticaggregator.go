package clients

import (
	"context"
	"fmt"
	"log/slog"

	semanticaggregatorv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/semantic_aggregator/v1"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// SemanticAggregator adapts Semantic Aggregator's resource import RPCs for Gateway.
type SemanticAggregator struct {
	log     *slog.Logger
	imports semanticaggregatorv1.ResourceImportServiceClient
}

func NewSemanticAggregator(log *slog.Logger, address string) (*SemanticAggregator, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("create semantic-aggregator gRPC client: %w", err)
	}

	return &SemanticAggregator{
		log:     log,
		imports: semanticaggregatorv1.NewResourceImportServiceClient(conn),
	}, nil
}

func (c *SemanticAggregator) ImportResources(ctx context.Context, actorID string, request *semanticaggregatorv1.ImportResourcesRequest) (*semanticaggregatorv1.ImportResourcesResponse, error) {
	ctx = metadata.AppendToOutgoingContext(ctx, actorMetadataKey, actorID)
	response, err := c.imports.ImportResources(ctx, request)
	if err != nil {
		return nil, wrapRPCError(c.log, ctx, "semantic-aggregator", "import resources", err)
	}
	return response, nil
}
