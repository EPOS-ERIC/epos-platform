package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"buf.build/go/protovalidate"
	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/grpcapi"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func Run(ctx context.Context, addr string, log *slog.Logger, repo repository.RetrievalRepository, client *retriever.Client, schemaBudget uint64) error {
	validator, err := protovalidate.New()
	if err != nil {
		return fmt.Errorf("create protobuf validator: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(logUnaryRPC(log), validateUnaryRPC(validator)),
	)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	dataretrieverv1.RegisterDataRetrieverServiceServer(grpcServer, grpcapi.New(repo, client, log, schemaBudget))

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- grpcServer.Serve(listener) }()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve grpc: %w", err)
	case <-ctx.Done():
		grpcServer.GracefulStop()
		return nil
	}
}

func validateUnaryRPC(validator protovalidate.Validator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		message, ok := request.(proto.Message)
		if !ok {
			return nil, status.Error(codes.Internal, "request is not a protobuf message")
		}
		if err := validator.Validate(message); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return handler(ctx, request)
	}
}

func logUnaryRPC(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		response, err := handler(ctx, request)
		if err != nil {
			code := status.Code(err)
			level := slog.LevelDebug
			switch code {
			case codes.Unknown, codes.ResourceExhausted, codes.Unimplemented, codes.Internal, codes.Unavailable, codes.DataLoss, codes.DeadlineExceeded:
				level = slog.LevelError
			}
			log.Log(ctx, level, "grpc request failed", "method", info.FullMethod, "code", code.String(), "error", err)
		}
		return response, err
	}
}
