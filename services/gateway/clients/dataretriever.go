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

func (c *DataRetriever) RetrieveDownload(ctx context.Context, target domain.DownloadTarget) (domain.DownloadResponse, error) {
	if target.ResourceID == uuid.Nil || target.VersionID == uuid.Nil || target.URL == "" {
		return domain.DownloadResponse{}, apperror.Default(apperror.CodeInternal)
	}

	streamCtx, cancel := context.WithCancel(ctx)

	stream, err := c.retrieval.RetrieveStream(streamCtx, &dataretrieverv1.RetrieveStreamRequest{
		Method:      dataretrieverv1.HTTPMethod_HTTP_METHOD_GET,
		UrlTemplate: target.URL,
		ResourceId:  target.ResourceID.String(),
		VersionId:   target.VersionID.String(),
		Mode:        dataretrieverv1.RetrieveMode_RETRIEVE_MODE_EXECUTE,
	})
	if err != nil {
		rpcErr := downloadRPCError(c.log, streamCtx, err)
		cancel()
		return domain.DownloadResponse{}, rpcErr
	}

	first, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			cancel()
			return domain.DownloadResponse{}, apperror.Default(apperror.CodeInternal)
		}

		rpcErr := downloadRPCError(c.log, streamCtx, err)
		cancel()

		return domain.DownloadResponse{}, rpcErr
	}

	status, err := fromProtoDownloadStreamHeaders(first)
	if err != nil {
		cancel()
		return domain.DownloadResponse{}, err
	}

	return domain.DownloadResponse{
		StatusCode: status,
		Body: &downloadBody{
			recv: stream.Recv,
			mapError: func(err error) error {
				return downloadRPCError(c.log, streamCtx, err)
			},
			cancel: cancel,
		},
	}, nil
}

type downloadBody struct {
	recv     func() (*dataretrieverv1.RetrieveStreamResponse, error)
	mapError func(error) error
	cancel   context.CancelFunc
	pending  []byte
}

func (r *downloadBody) Read(p []byte) (int, error) {
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

func (r *downloadBody) Close() error {
	r.cancel()
	return nil
}

func downloadRPCError(log *slog.Logger, ctx context.Context, err error) error {
	wrapped := wrapRPCError(log, ctx, "data-retriever", "retrieve distribution download", err)
	if ctx.Err() != nil || apperror.IsCode(wrapped, apperror.CodeInternal) {
		return wrapped
	}

	return apperror.Wrap(apperror.CodeInternal, "", wrapped)
}
