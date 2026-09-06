package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"buf.build/go/protovalidate"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/access"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/authz"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/categories"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/config"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/contactpoints"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/datasets"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/groups"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/grpcapi"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadataimport"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/organizations"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwareapplications"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwaresourcecodes"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/turtle"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/users"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const maxImportTurtleMessageSize = 10<<20 + 5

func Run(ctx context.Context, cfg config.Config, log *slog.Logger, db *db.PostgresDB) error {
	validator, err := protovalidate.New()
	if err != nil {
		return fmt.Errorf("create protobuf validator: %w", err)
	}

	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(validateUnaryRPC(validator)),
		grpc.MaxRecvMsgSize(maxImportTurtleMessageSize),
	)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	authorizer := authz.New(log, db.Authorization)
	usersSvc := users.New(log, db.Users, authorizer)
	importSvc := metadataimport.New(log, db.Imports, authorizer)
	turtler := turtle.New(log)
	groupsSvc := groups.New(log, db.Groups, authorizer)
	categoriesSvc := categories.New(log, db.Categories, authorizer)
	contactPointsSvc := contactpoints.New(log, db.ContactPoints, authorizer)

	metadatalifecyclev1.RegisterIdentityServiceServer(server, grpcapi.NewIdentity(usersSvc))
	metadatalifecyclev1.RegisterUserServiceServer(server, grpcapi.NewUser(usersSvc))
	metadatalifecyclev1.RegisterImportServiceServer(server, grpcapi.NewImport(importSvc, turtler))
	metadatalifecyclev1.RegisterGroupServiceServer(server, grpcapi.NewGroup(groupsSvc))
	metadatalifecyclev1.RegisterCategoryServiceServer(server, grpcapi.NewCategory(categoriesSvc))
	metadatalifecyclev1.RegisterContactPointServiceServer(server, grpcapi.NewContactPoint(contactPointsSvc))
	accessSvc := access.New(log, db.Access, authorizer)
	metadatalifecyclev1.RegisterGroupAccessServiceServer(server, grpcapi.NewAccess(accessSvc))
	organizationsSvc := organizations.New(log, db.Organizations, authorizer)
	metadatalifecyclev1.RegisterOrganizationServiceServer(server, grpcapi.NewOrganization(organizationsSvc))
	datasetSvc := datasets.New(log, db.Datasets, authorizer)
	applicationSvc := softwareapplications.New(log, db.SoftwareApplications, authorizer)
	sourceCodeSvc := softwaresourcecodes.New(log, db.SoftwareSourceCodes, authorizer)
	metadatalifecyclev1.RegisterResourceServiceServer(server, grpcapi.NewResource(datasetSvc, applicationSvc, sourceCodeSvc))
	discoverySvc := discovery.New(log, db.Discovery, authorizer, datasetSvc, applicationSvc, sourceCodeSvc, groupsSvc)
	metadatalifecyclev1.RegisterDiscoveryServiceServer(server, grpcapi.NewDiscovery(discoverySvc))

	listener, err := net.Listen("tcp", ":"+cfg.ServerPort)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srvErr := make(chan error, 1)
	go func() {
		log.Info("starting server", "listen", cfg.ServerPort)
		srvErr <- server.Serve(listener)
	}()

	select {
	case err := <-srvErr:
		log.Error("server failed", "error", err)
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
		log.Info("server terminated: stopping")
	}

	server.GracefulStop()

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
