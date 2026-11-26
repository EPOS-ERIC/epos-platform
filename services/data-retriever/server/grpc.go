package server

import (
	"context"
	"log/slog"

	sharedgrpc "github.com/EPOS-ERIC/epos-platform/packages/go/grpcserver"
	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/config"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/grpcapi"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retrieval"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/upstream"
	"google.golang.org/grpc"
)

func Run(ctx context.Context, cfg config.Config, log *slog.Logger, repo retrieval.Repository, client *upstream.Client, healthCheck func(context.Context) error) error {
	service := retrieval.New(repo, client, log, uint64(cfg.SchemaMaxBytes), uint64(cfg.MaxResponseBytes))
	api := grpcapi.New(service)
	return sharedgrpc.Run(ctx, sharedgrpc.Config{
		HealthCheck: healthCheck,
		Address:     ":" + cfg.ServerPort,
		Logger:      log,
		ValidationError: func(_ *grpc.UnaryServerInfo, err error) error {
			return grpcapi.ValidationAppError(err)
		},
		StreamValidationError: func(_ *grpc.StreamServerInfo, err error) error {
			return grpcapi.ValidationAppError(err)
		},
		Register: func(server *grpc.Server) {
			dataretrieverv1.RegisterDataRetrieverServiceServer(server, api)
		},
	})
}
