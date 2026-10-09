// Package grpcserver configures shared gRPC server infrastructure.
package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"

	"buf.build/go/protovalidate"
	apperrorgrpc "github.com/EPOS-ERIC/epos-platform/packages/go/apperror/grpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Config supplies service-specific registration and transport error policy.
type Config struct {
	Address         string
	Logger          *slog.Logger
	Options         []grpc.ServerOption
	Interceptors    []grpc.UnaryServerInterceptor
	ValidationError func(*grpc.UnaryServerInfo, error) error
	Register        func(*grpc.Server)
}

// Run starts the gRPC server, registers health and service handlers, and stops
// immediately when ctx is canceled.
func Run(ctx context.Context, cfg Config) error {
	if cfg.Address == "" {
		return fmt.Errorf("gRPC listen address is required")
	}
	if cfg.ValidationError == nil {
		return fmt.Errorf("gRPC validation error handler is required")
	}
	if cfg.Register == nil {
		return fmt.Errorf("gRPC service registration is required")
	}
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	validator, err := protovalidate.New()
	if err != nil {
		return fmt.Errorf("create protobuf validator: %w", err)
	}

	interceptors := []grpc.UnaryServerInterceptor{recoverUnaryRPC(log)}
	interceptors = append(interceptors, cfg.Interceptors...)
	interceptors = append(interceptors,
		logAndNormalizeUnaryRPC(log),
		validateUnaryRPC(validator, cfg.ValidationError),
	)
	options := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(interceptors...),
	}
	options = append(options, cfg.Options...)
	server := grpc.NewServer(options...)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	cfg.Register(server)

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	log.InfoContext(ctx, "starting grpc server", "listen", cfg.Address)
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	case <-ctx.Done():
		log.InfoContext(ctx, "grpc server terminated: stopping")
		server.Stop()
		log.Info("grpc server shutdown complete")
		return nil
	}
}

func recoverUnaryRPC(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (response any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.ErrorContext(ctx, "grpc handler panicked", "method", info.FullMethod, "panic", recovered, "stack", string(debug.Stack()))

				span := trace.SpanFromContext(ctx)
				if span.IsRecording() {
					span.RecordError(fmt.Errorf("grpc handler panic: %v", recovered))
					span.SetStatus(otelcodes.Error, "internal server error")
				}

				response = nil
				err = status.Error(grpcCodes.Internal, "internal server error")
			}
		}()

		return handler(ctx, request)
	}
}

func logAndNormalizeUnaryRPC(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		response, cause := handler(ctx, request)
		if cause == nil {
			return response, nil
		}

		err := apperrorgrpc.NormalizeError(cause)
		code := status.Code(err)
		level := slog.LevelDebug
		switch code {
		case grpcCodes.Unknown,
			grpcCodes.ResourceExhausted,
			grpcCodes.Unimplemented,
			grpcCodes.Internal,
			grpcCodes.Unavailable,
			grpcCodes.DataLoss,
			grpcCodes.DeadlineExceeded:
			level = slog.LevelError
		case grpcCodes.OK,
			grpcCodes.Canceled,
			grpcCodes.InvalidArgument,
			grpcCodes.NotFound,
			grpcCodes.AlreadyExists,
			grpcCodes.PermissionDenied,
			grpcCodes.FailedPrecondition,
			grpcCodes.Aborted,
			grpcCodes.OutOfRange,
			grpcCodes.Unauthenticated:
		default:
		}
		if level == slog.LevelError {
			span := trace.SpanFromContext(ctx)
			if span.IsRecording() {
				span.RecordError(status.Error(code, cause.Error()))
			}
		}
		log.Log(ctx, level, "grpc request failed", "method", info.FullMethod, "code", code.String(), "error", cause)
		return response, err
	}
}

func validateUnaryRPC(
	validator protovalidate.Validator,
	validationError func(*grpc.UnaryServerInfo, error) error,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		message, ok := request.(proto.Message)
		if !ok {
			return nil, fmt.Errorf("gRPC request %s is not a protobuf message", info.FullMethod)
		}
		if err := validator.Validate(message); err != nil {
			appErr := validationError(info, err)
			if appErr == nil {
				appErr = fmt.Errorf("convert validation error for %s: %w", info.FullMethod, err)
			}
			return nil, appErr
		}
		return handler(ctx, request)
	}
}
