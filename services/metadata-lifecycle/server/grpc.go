package server

import (
	"context"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	sharedgrpc "github.com/EPOS-ERIC/epos-platform/packages/go/grpcserver"
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
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/notifications"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/organizations"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwareapplications"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwaresourcecodes"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/turtle"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/users"
	"google.golang.org/grpc"
)

const maxImportTurtleMessageSize = 10<<20 + 5

func Run(ctx context.Context, cfg config.Config, log *slog.Logger, db *db.PostgresDB) error {
	authorizer := authz.New(log, db.Authorization)
	notificationsSvc := notifications.New(log, db.Notifications, authorizer, cfg.GatewayPublicURL)
	usersSvc := users.New(log, db.Users, authorizer, notificationsSvc)
	importSvc := metadataimport.New(log, db.Imports, authorizer)
	turtler := turtle.New(log)
	groupsSvc := groups.New(log, db.Groups, authorizer)
	categoriesSvc := categories.New(log, db.Categories, authorizer)
	contactPointsSvc := contactpoints.New(log, db.ContactPoints, authorizer)
	accessSvc := access.New(log, db.Access, authorizer, notificationsSvc)
	organizationsSvc := organizations.New(log, db.Organizations, authorizer)
	datasetSvc := datasets.New(log, db.Datasets, authorizer, notificationsSvc)
	applicationSvc := softwareapplications.New(log, db.SoftwareApplications, authorizer, notificationsSvc)
	sourceCodeSvc := softwaresourcecodes.New(log, db.SoftwareSourceCodes, authorizer, notificationsSvc)
	discoverySvc := discovery.New(log, db.Discovery, authorizer)

	return sharedgrpc.Run(ctx, sharedgrpc.Config{
		Address:      ":" + cfg.ServerPort,
		Logger:       log,
		Options:      []grpc.ServerOption{grpc.MaxRecvMsgSize(maxImportTurtleMessageSize)},
		Interceptors: []grpc.UnaryServerInterceptor{grpcapi.ActorTracingInterceptor()},
		ValidationError: func(info *grpc.UnaryServerInfo, err error) error {
			if appErr := grpcapi.ValidationAppError(info.FullMethod, err); appErr != nil {
				return appErr
			}
			return apperror.Wrap(apperror.CodeValidationFailed, "The request is invalid.", err)
		},
		Register: func(server *grpc.Server) {
			metadatalifecyclev1.RegisterIdentityServiceServer(server, grpcapi.NewIdentity(usersSvc))
			metadatalifecyclev1.RegisterUserServiceServer(server, grpcapi.NewUser(usersSvc))
			metadatalifecyclev1.RegisterAuthorizationServiceServer(server, grpcapi.NewAuthorization(authorizer))
			metadatalifecyclev1.RegisterImportServiceServer(server, grpcapi.NewImport(importSvc, turtler))
			metadatalifecyclev1.RegisterGroupServiceServer(server, grpcapi.NewGroup(groupsSvc))
			metadatalifecyclev1.RegisterCategoryServiceServer(server, grpcapi.NewCategory(categoriesSvc))
			metadatalifecyclev1.RegisterContactPointServiceServer(server, grpcapi.NewContactPoint(contactPointsSvc))
			metadatalifecyclev1.RegisterGroupAccessServiceServer(server, grpcapi.NewAccess(accessSvc))
			metadatalifecyclev1.RegisterOrganizationServiceServer(server, grpcapi.NewOrganization(organizationsSvc))
			metadatalifecyclev1.RegisterResourceServiceServer(server, grpcapi.NewResource(datasetSvc, applicationSvc, sourceCodeSvc))
			metadatalifecyclev1.RegisterDiscoveryServiceServer(server, grpcapi.NewDiscovery(discoverySvc))
			metadatalifecyclev1.RegisterNotificationServiceServer(server, grpcapi.NewNotification(notificationsSvc))
		},
	})
}
