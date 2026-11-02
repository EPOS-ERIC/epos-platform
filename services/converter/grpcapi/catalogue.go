// Package grpcapi exposes converter operations over gRPC.
package grpcapi

import (
	"context"
	"log/slog"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	converterv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/converter/v1"
	"github.com/EPOS-ERIC/epos-platform/services/converter/catalogue"
	"github.com/EPOS-ERIC/epos-platform/services/converter/plugin"
	"google.golang.org/grpc/metadata"
)

type Server struct {
	converterv1.UnimplementedConverterServiceServer
	service *catalogue.Service
	manager *plugin.Manager
	max     int64
	log     *slog.Logger
}

func New(service *catalogue.Service, manager *plugin.Manager, maxBytes int64, log *slog.Logger) *Server {
	return &Server{service: service, manager: manager, max: maxBytes, log: log}
}

func actor(ctx context.Context) (string, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("x-epos-user-id")
	if len(values) == 0 {
		return "", apperror.Default(apperror.CodeUnauthenticated)
	}
	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", apperror.Field("x-epos-user-id", "A single non-blank actor identifier is required.", nil)
	}
	return strings.TrimSpace(values[0]), nil
}

func (s *Server) CreatePlugin(ctx context.Context, request *converterv1.CreatePluginRequest) (*converterv1.CreatePluginResponse, error) {
	who, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	input, err := toDomainPluginInput(request.GetInput())
	if err != nil {
		return nil, err
	}
	record, failed, err := s.service.CreatePlugin(ctx, input, who)
	if err != nil {
		return nil, err
	}
	status, err := s.service.Status(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	p, err := toProtoPlugin(record, status)
	return &converterv1.CreatePluginResponse{Plugin: p, SyncFailed: failed}, err
}

func (s *Server) GetPlugin(ctx context.Context, request *converterv1.GetPluginRequest) (*converterv1.GetPluginResponse, error) {
	id, err := parseID("id", request.GetId())
	if err != nil {
		return nil, err
	}
	record, err := s.service.Plugin(ctx, id)
	if err != nil {
		return nil, err
	}
	status, err := s.service.Status(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := toProtoPlugin(record, status)
	return &converterv1.GetPluginResponse{Plugin: p}, err
}

func (s *Server) ListPlugins(ctx context.Context, _ *converterv1.ListPluginsRequest) (*converterv1.ListPluginsResponse, error) {
	records, err := s.service.Plugins(ctx)
	if err != nil {
		return nil, err
	}
	response := &converterv1.ListPluginsResponse{}
	for _, record := range records {
		status, err := s.service.Status(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		p, err := toProtoPlugin(record, status)
		if err != nil {
			return nil, err
		}
		response.Plugins = append(response.Plugins, p)
	}
	return response, nil
}

func (s *Server) UpdatePlugin(ctx context.Context, request *converterv1.UpdatePluginRequest) (*converterv1.UpdatePluginResponse, error) {
	who, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", request.GetId())
	if err != nil {
		return nil, err
	}
	input, err := toDomainPluginInput(request.GetInput())
	if err != nil {
		return nil, err
	}
	record, failed, err := s.service.UpdatePlugin(ctx, id, input, request.GetEnabled(), who)
	if err != nil {
		return nil, err
	}
	status, err := s.service.Status(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := toProtoPlugin(record, status)
	return &converterv1.UpdatePluginResponse{Plugin: p, SyncFailed: failed}, err
}

func (s *Server) DeletePlugin(ctx context.Context, request *converterv1.DeletePluginRequest) (*converterv1.DeletePluginResponse, error) {
	if _, err := actor(ctx); err != nil {
		return nil, err
	}
	id, err := parseID("id", request.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.service.DeletePlugin(ctx, id); err != nil {
		return nil, err
	}
	return &converterv1.DeletePluginResponse{}, nil
}

func (s *Server) SyncPlugin(ctx context.Context, request *converterv1.SyncPluginRequest) (*converterv1.SyncPluginResponse, error) {
	if _, err := actor(ctx); err != nil {
		return nil, err
	}
	id, err := parseID("id", request.GetId())
	if err != nil {
		return nil, err
	}
	record, err := s.service.SyncPlugin(ctx, id)
	if err != nil {
		s.log.ErrorContext(ctx, "manual plugin sync failed", "plugin_id", id, "error", err)
		return nil, err
	}
	status, err := s.service.Status(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := toProtoPlugin(record, status)
	return &converterv1.SyncPluginResponse{Plugin: p}, err
}

func (s *Server) CreateRelation(ctx context.Context, request *converterv1.CreateRelationRequest) (*converterv1.CreateRelationResponse, error) {
	who, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	input, err := toDomainRelationInput(request.GetInput())
	if err != nil {
		return nil, err
	}
	record, err := s.service.CreateRelation(ctx, input, who)
	if err != nil {
		return nil, err
	}
	return &converterv1.CreateRelationResponse{Relation: toProtoRelation(record)}, nil
}

func (s *Server) GetRelation(ctx context.Context, request *converterv1.GetRelationRequest) (*converterv1.GetRelationResponse, error) {
	id, err := parseID("id", request.GetId())
	if err != nil {
		return nil, err
	}
	record, err := s.service.Relation(ctx, id)
	if err != nil {
		return nil, err
	}
	return &converterv1.GetRelationResponse{Relation: toProtoRelation(record)}, nil
}

func (s *Server) ListRelations(ctx context.Context, _ *converterv1.ListRelationsRequest) (*converterv1.ListRelationsResponse, error) {
	records, err := s.service.Relations(ctx)
	if err != nil {
		return nil, err
	}
	response := &converterv1.ListRelationsResponse{}
	for _, record := range records {
		response.Relations = append(response.Relations, toProtoRelation(record))
	}
	return response, nil
}

func (s *Server) UpdateRelation(ctx context.Context, request *converterv1.UpdateRelationRequest) (*converterv1.UpdateRelationResponse, error) {
	who, err := actor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseID("id", request.GetId())
	if err != nil {
		return nil, err
	}
	input, err := toDomainRelationInput(request.GetInput())
	if err != nil {
		return nil, err
	}
	record, err := s.service.UpdateRelation(ctx, id, input, who)
	if err != nil {
		return nil, err
	}
	return &converterv1.UpdateRelationResponse{Relation: toProtoRelation(record)}, nil
}

func (s *Server) DeleteRelation(ctx context.Context, request *converterv1.DeleteRelationRequest) (*converterv1.DeleteRelationResponse, error) {
	if _, err := actor(ctx); err != nil {
		return nil, err
	}
	id, err := parseID("id", request.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.service.DeleteRelation(ctx, id); err != nil {
		return nil, err
	}
	return &converterv1.DeleteRelationResponse{}, nil
}
