package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"buf.build/go/protovalidate"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/authz"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/categories"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/config"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/contactpoints"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/groups"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/grpcapi"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/organizations"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/users"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func RunGRPC(ctx context.Context, cfg config.Config, log *slog.Logger, db *db.PostgresDB) error {
	validator, err := protovalidate.New()
	if err != nil {
		return fmt.Errorf("create protobuf validator: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(validateUnaryRPC(validator)),
	)

	authorizer := authz.New(log, db.Authorization)
	usersSvc := users.New(log, db.Users, authorizer)
	groupsSvc := groups.New(log, db.Groups, authorizer)
	categoriesSvc := categories.New(log, db.Categories, authorizer)
	contactPointsSvc := contactpoints.New(log, db.ContactPoints, authorizer)

	metadatalifecyclev1.RegisterIdentityServiceServer(grpcServer, grpcapi.NewIdentity(usersSvc))
	metadatalifecyclev1.RegisterUserServiceServer(grpcServer, grpcapi.NewUser(usersSvc))
	metadatalifecyclev1.RegisterGroupServiceServer(grpcServer, grpcapi.NewGroup(groupsSvc))
	metadatalifecyclev1.RegisterCategoryServiceServer(grpcServer, grpcapi.NewCategory(categoriesSvc))
	metadatalifecyclev1.RegisterContactPointServiceServer(grpcServer, grpcapi.NewContactPoint(contactPointsSvc))
	organizationsSvc := organizations.New(log, db.Organizations, authorizer)
	metadatalifecyclev1.RegisterOrganizationServiceServer(grpcServer, grpcapi.NewOrganization(organizationsSvc))

	listener, err := net.Listen("tcp", ":"+cfg.GRPCServerPort)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srvErr := make(chan error, 1)
	go func() {
		log.Info("starting grpc server", "listen", cfg.GRPCServerPort)
		srvErr <- grpcServer.Serve(listener)
	}()

	select {
	case err := <-srvErr:
		log.Error("grpc server failed", "error", err)
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
		log.Info("grpc server terminated: stopping")
	}

	grpcServer.GracefulStop()

	log.Info("server shutdown complete")

	return nil
}

func validateUnaryRPC(validator protovalidate.Validator) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		message, ok := req.(proto.Message)
		if !ok {
			return nil, status.Error(codes.Internal, "request is not a protobuf message")
		}

		if err := validator.Validate(message); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return handler(ctx, req)
	}
}
