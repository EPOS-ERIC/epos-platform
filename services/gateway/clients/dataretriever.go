package clients

import (
	"context"
	"fmt"
	"log/slog"

	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DataRetriever struct {
	log       *slog.Logger
	retrieval dataretrieverv1.DataRetrieverServiceClient
}

func NewDataRetriever(log *slog.Logger, baseURL string) (*DataRetriever, error) {
	conn, err := grpc.NewClient(
		baseURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("create data-retriever gRPC client: %w", err)
	}
	return &DataRetriever{
		log:       log,
		retrieval: dataretrieverv1.NewDataRetrieverServiceClient(conn),
	}, nil
}

func (c *DataRetriever) Retrieve(ctx context.Context, request *dataretrieverv1.RetrieveRequest) (*dataretrieverv1.RetrieveResponse, error) {
	response, err := c.retrieval.Retrieve(ctx, request)
	if err != nil {
		return nil, wrapRPCError(c.log, ctx, "data-retriever", "retrieve resource version", err)
	}
	return response, nil
}
