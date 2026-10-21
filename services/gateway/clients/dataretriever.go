package clients

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DataRetriever struct {
	log       *slog.Logger
	retrieval dataretrieverv1.DataRetrieverServiceClient
}

func NewDataRetriever(log *slog.Logger, address string) (*DataRetriever, error) {
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

func (c *DataRetriever) RetrieveDownload(ctx context.Context, target domain.DownloadTarget) (domain.RetrievalResponse, error) {
	return c.retrieve(ctx, &dataretrieverv1.RetrieveStreamRequest{
		Method:     dataretrieverv1.HTTPMethod_HTTP_METHOD_GET,
		Url:        target.URL,
		ResourceId: target.ResourceID.String(),
		VersionId:  target.VersionID.String(),
		// Skip database persistence for downloadable files
		SkipPersistence: true,
	})
}

func (c *DataRetriever) RetrieveAccess(ctx context.Context, request domain.PreparedRequest) (domain.RetrievalResponse, error) {
	var method dataretrieverv1.HTTPMethod
	switch request.Method {
	case domain.HTTPMethodGET:
		method = dataretrieverv1.HTTPMethod_HTTP_METHOD_GET
	case domain.HTTPMethodPOST:
		method = dataretrieverv1.HTTPMethod_HTTP_METHOD_POST
	case domain.HTTPMethodPUT:
		method = dataretrieverv1.HTTPMethod_HTTP_METHOD_PUT
	case domain.HTTPMethodDELETE:
		method = dataretrieverv1.HTTPMethod_HTTP_METHOD_DELETE
	default:
		return domain.RetrievalResponse{}, apperror.Default(apperror.CodeInternal)
	}

	return c.retrieve(ctx, &dataretrieverv1.RetrieveStreamRequest{
		Method:     method,
		Url:        request.URL,
		Headers:    request.Headers,
		Body:       request.Body,
		ResourceId: request.ResourceID.String(),
		VersionId:  request.VersionID.String(),
	})
}

func (c *DataRetriever) retrieve(ctx context.Context, request *dataretrieverv1.RetrieveStreamRequest) (domain.RetrievalResponse, error) {
	if request.ResourceId == uuid.Nil.String() || request.VersionId == uuid.Nil.String() || request.Url == "" {
		return domain.RetrievalResponse{}, apperror.Default(apperror.CodeInternal)
	}

	streamCtx, cancel := context.WithCancel(ctx)

	stream, err := c.retrieval.RetrieveStream(streamCtx, request)
	if err != nil {
		rpcErr := retrievalRPCError(c.log, streamCtx, err)
		cancel()
		return domain.RetrievalResponse{}, rpcErr
	}

	first, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			cancel()
			return domain.RetrievalResponse{}, apperror.Default(apperror.CodeInternal)
		}

		rpcErr := retrievalRPCError(c.log, streamCtx, err)
		cancel()

		return domain.RetrievalResponse{}, rpcErr
	}

	status, err := fromProtoRetrievalStreamHeaders(first)
	if err != nil {
		cancel()
		return domain.RetrievalResponse{}, err
	}

	return domain.RetrievalResponse{
		StatusCode: status,
		Body: &retrievalBody{
			recv: stream.Recv,
			mapError: func(err error) error {
				return retrievalRPCError(c.log, streamCtx, err)
			},
			cancel: cancel,
		},
	}, nil
}

type retrievalBody struct {
	recv     func() (*dataretrieverv1.RetrieveStreamResponse, error)
	mapError func(error) error
	cancel   context.CancelFunc
	pending  []byte
}

func (r *retrievalBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(r.pending) == 0 {
		response, err := r.recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0, io.EOF
			}
			return 0, r.mapError(err)
		}
		if response == nil || response.GetHeaders() != nil {
			return 0, apperror.Default(apperror.CodeInternal)
		}
		r.pending = response.GetChunk()
		if len(r.pending) == 0 {
			return 0, apperror.Default(apperror.CodeInternal)
		}
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

func (r *retrievalBody) Close() error {
	r.cancel()
	return nil
}

func retrievalRPCError(log *slog.Logger, ctx context.Context, err error) error {
	wrapped := wrapRPCError(log, ctx, "data-retriever", "retrieve distribution response", err)
	if ctx.Err() != nil ||
		apperror.IsCode(wrapped, apperror.CodeInternal) ||
		apperror.IsCode(wrapped, apperror.CodeUpstreamUnavailable) ||
		apperror.IsCode(wrapped, apperror.CodeUpstreamTimeout) {
		return wrapped
	}

	return apperror.Wrap(apperror.CodeInternal, "", wrapped)
}
