package grpcapi

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/configuration"
	"github.com/EPOS-ERIC/epos-platform/services/sharing/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

const (
	userIDMetadataKey = "x-epos-user-id"
	manageMetadataKey = "x-epos-configuration-manage"
)

func callerFromContext(ctx context.Context) (domain.Caller, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get(userIDMetadataKey)
	if len(values) == 0 {
		return domain.Caller{}, nil
	}
	if len(values) != 1 {
		return domain.Caller{}, apperror.Field(userIDMetadataKey, "A single non-blank actor identifier is required.", nil)
	}
	actor, err := domain.NewActor(values[0])
	if err != nil {
		return domain.Caller{}, apperror.Field(userIDMetadataKey, "A single non-blank actor identifier is required.", err)
	}
	manage := md.Get(manageMetadataKey)
	if len(manage) > 1 || (len(manage) == 1 && manage[0] != "true" && manage[0] != "false") {
		return domain.Caller{}, apperror.Field(manageMetadataKey, "Management capability must be a single boolean.", nil)
	}
	return domain.Caller{Actor: actor, CanManage: len(manage) == 1 && manage[0] == "true"}, nil
}

type Server struct {
	configurationv1.UnimplementedConfigurationServiceServer
	service *configuration.Service
}

func New(service *configuration.Service) *Server { return &Server{service: service} }

func (s *Server) CreateConfiguration(ctx context.Context, request *configurationv1.CreateConfigurationRequest) (*configurationv1.CreateConfigurationResponse, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	caller, err := callerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	input, err := toDomainConfigurationInput(request.GetConfiguration(), request.GetTitle(), request.Description, request.GetListOfServices(), new(request.GetIsPublic()))
	if err != nil {
		return nil, err
	}
	record, err := s.service.Create(ctx, caller, input)
	if err != nil {
		return nil, err
	}
	return &configurationv1.CreateConfigurationResponse{Configuration: toProtoConfiguration(record)}, nil
}

func (s *Server) GetConfiguration(ctx context.Context, request *configurationv1.GetConfigurationRequest) (*configurationv1.GetConfigurationResponse, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	caller, id, err := readSelector(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	record, err := s.service.Get(ctx, caller, id)
	if err != nil {
		return nil, err
	}
	return &configurationv1.GetConfigurationResponse{Configuration: toProtoConfiguration(record)}, nil
}

func (s *Server) ListConfigurations(ctx context.Context, request *configurationv1.ListConfigurationsRequest) (*configurationv1.ListConfigurationsResponse, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	scope, err := toDomainListScope(request.GetScope())
	if err != nil {
		return nil, err
	}
	caller, err := callerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var ownerID domain.Actor
	if request.OwnerId != nil {
		ownerID, err = domain.NewActor(request.GetOwnerId())
		if err != nil {
			return nil, apperror.Field("owner_id", "Owner ID must not be blank.", err)
		}
	}
	records, err := s.service.List(ctx, caller, configuration.ListFilter{Scope: scope, OwnerID: ownerID})
	if err != nil {
		return nil, err
	}
	includeOwner := caller.CanManage && scope != domain.ListScopeOfficial
	return &configurationv1.ListConfigurationsResponse{Configurations: toProtoConfigurationSummaries(records, includeOwner)}, nil
}

func (s *Server) UpdateConfiguration(ctx context.Context, request *configurationv1.UpdateConfigurationRequest) (*configurationv1.UpdateConfigurationResponse, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	caller, id, err := readSelector(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	input, err := toDomainConfigurationInput(request.GetConfiguration(), request.GetTitle(), request.Description, request.GetListOfServices(), request.IsPublic)
	if err != nil {
		return nil, err
	}
	record, err := s.service.Update(ctx, caller, id, input)
	if err != nil {
		return nil, err
	}
	return &configurationv1.UpdateConfigurationResponse{Configuration: toProtoConfiguration(record)}, nil
}

func (s *Server) DeleteConfiguration(ctx context.Context, request *configurationv1.DeleteConfigurationRequest) (*configurationv1.DeleteConfigurationResponse, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	caller, id, err := readSelector(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.service.Delete(ctx, caller, id); err != nil {
		return nil, err
	}
	return &configurationv1.DeleteConfigurationResponse{}, nil
}

func (s *Server) SetConfigurationOfficialStatus(ctx context.Context, request *configurationv1.SetConfigurationOfficialStatusRequest) (*configurationv1.SetConfigurationOfficialStatusResponse, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	if request.IsOfficial == nil {
		return nil, apperror.Field("is_official", "Official status is required.", nil)
	}
	caller, id, err := readSelector(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	record, err := s.service.SetOfficialStatus(ctx, caller, id, request.GetIsOfficial())
	if err != nil {
		return nil, err
	}
	return &configurationv1.SetConfigurationOfficialStatusResponse{Configuration: toProtoConfiguration(record)}, nil
}

func readSelector(ctx context.Context, rawID string) (domain.Caller, uuid.UUID, error) {
	id, err := uuid.Parse(rawID)
	if err != nil || id == uuid.Nil {
		return domain.Caller{}, uuid.Nil, apperror.Field("id", "ID must be a non-zero UUID.", err)
	}
	caller, err := callerFromContext(ctx)
	return caller, id, err
}
