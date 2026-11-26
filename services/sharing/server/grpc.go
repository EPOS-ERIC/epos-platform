package server

import (
	"context"
	"log/slog"

	sharedgrpc "github.com/EPOS-ERIC/epos-platform/packages/go/grpcserver"
	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/config"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/configuration"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/db"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/grpcapi"
	"google.golang.org/grpc"
)

func Run(ctx context.Context, cfg config.Config, log *slog.Logger, database *db.PostgresDB) error {
	service := configuration.New(database.Configurations)
	return sharedgrpc.Run(ctx, sharedgrpc.Config{
		HealthCheck: database.Ping,
		Address:     ":" + cfg.ServerPort,
		Logger:      log,
		ValidationError: func(_ *grpc.UnaryServerInfo, err error) error {
			return grpcapi.ValidationAppError(err)
		},
		Register: func(server *grpc.Server) {
			configurationv1.RegisterConfigurationServiceServer(server, grpcapi.New(service))
		},
	})
}
