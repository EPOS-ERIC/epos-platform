package clients

import (
	"context"
	"errors"
	"fmt"
	"io"
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

type RetrievalStream interface {
	Recv(context.Context) (*dataretrieverv1.RetrieveStreamResponse, error)
}

func NewDataRetriever(log *slog.Logger, address string) (*DataRetriever, error) {
	if log == nil {
		log = slog.Default()
	}
	conn, err := grpc.NewClient(
		address,
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

func (c *DataRetriever) RetrieveStream(ctx context.Context, request *dataretrieverv1.RetrieveStreamRequest) (RetrievalStream, error) {
	stream, err := c.retrieval.RetrieveStream(ctx, request)
	if err != nil {
		return nil, wrapRPCError(c.log, ctx, "data-retriever", "retrieve resource version", err)
	}
	return &dataRetrieverStream{stream: stream, log: c.log}, nil
}

type dataRetrieverStream struct {
	stream dataretrieverv1.DataRetrieverService_RetrieveStreamClient
	log    *slog.Logger
}

func (s *dataRetrieverStream) Recv(ctx context.Context) (*dataretrieverv1.RetrieveStreamResponse, error) {
	response, err := s.stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, wrapRPCError(s.log, ctx, "data-retriever", "retrieve resource version", err)
	}
	return response, nil
}
