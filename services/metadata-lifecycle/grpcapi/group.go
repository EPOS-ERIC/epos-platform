package grpcapi

import (
	"context"
	"errors"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/groups"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GroupServer struct {
	metadatalifecyclev1.UnimplementedGroupServiceServer
	groups *groups.Service
}

func NewGroup(groups *groups.Service) *GroupServer {
	return &GroupServer{groups: groups}
}

func (g *GroupServer) ListGroups(ctx context.Context, _ *metadatalifecyclev1.ListGroupsRequest) (*metadatalifecyclev1.ListGroupsResponse, error) {
	values, err := g.groups.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", status.Error(codes.Internal, "list groups"))
	}

	result := make([]*metadatalifecyclev1.Group, len(values))
	for i, value := range values {
		mapped, err := toProtoGroup(value)
		if err != nil {
			return nil, fmt.Errorf("map groups: %w", status.Error(codes.Internal, "map groups"))
		}
		result[i] = mapped
	}

	return &metadatalifecyclev1.ListGroupsResponse{Groups: result}, nil
}

func (g *GroupServer) GetGroup(ctx context.Context, request *metadatalifecyclev1.GetGroupRequest) (*metadatalifecyclev1.GetGroupResponse, error) {
	id, err := uuid.Parse(request.GetGroupId())
	if err != nil || id == uuid.Nil {
		return nil, fmt.Errorf("invalid group id: %w", status.Error(codes.InvalidArgument, "invalid group id"))
	}

	value, err := g.groups.Get(ctx, id)
	if err != nil {
		return nil, toGroupError(err)
	}

	group, err := toProtoGroup(value)
	if err != nil {
		return nil, fmt.Errorf("map group: %w", status.Error(codes.Internal, "map group"))
	}

	return &metadatalifecyclev1.GetGroupResponse{Group: group}, nil
}

func (g *GroupServer) CreateGroup(ctx context.Context, request *metadatalifecyclev1.CreateGroupRequest) (*metadatalifecyclev1.CreateGroupResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	group, err := g.groups.Create(ctx, actor, domain.GroupInput{
		Name:        request.GetName(),
		Description: request.Description,
		Label:       request.Label,
		Logo:        request.Logo,
		Page:        request.Page,
		Color:       request.Color,
		Position:    request.GetPosition(),
		Concept:     request.GetConcept(),
	})
	if err != nil {
		return nil, toGroupMutationError("create group", err)
	}

	result, err := toProtoGroup(group)
	if err != nil {
		return nil, fmt.Errorf("map group: %w", status.Error(codes.Internal, "map group"))
	}

	return &metadatalifecyclev1.CreateGroupResponse{Group: result}, nil
}

func (g *GroupServer) ReplaceGroup(ctx context.Context, request *metadatalifecyclev1.ReplaceGroupRequest) (*metadatalifecyclev1.ReplaceGroupResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetGroupId())
	if err != nil || id == uuid.Nil {
		return nil, fmt.Errorf("invalid group id: %w", status.Error(codes.InvalidArgument, "invalid group id"))
	}

	group, err := g.groups.Replace(ctx, actor, id, domain.GroupInput{
		Name:        request.GetName(),
		Description: request.Description,
		Label:       request.Label,
		Logo:        request.Logo,
		Page:        request.Page,
		Color:       request.Color,
		Position:    request.GetPosition(),
		Concept:     request.GetConcept(),
	})
	if err != nil {
		return nil, toGroupMutationError("replace group", err)
	}

	result, err := toProtoGroup(group)
	if err != nil {
		return nil, fmt.Errorf("map group: %w", status.Error(codes.Internal, "map group"))
	}

	return &metadatalifecyclev1.ReplaceGroupResponse{Group: result}, nil
}

func (g *GroupServer) DeleteGroup(ctx context.Context, request *metadatalifecyclev1.DeleteGroupRequest) (*metadatalifecyclev1.DeleteGroupResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetGroupId())
	if err != nil || id == uuid.Nil {
		return nil, fmt.Errorf("invalid group id: %w", status.Error(codes.InvalidArgument, "invalid group id"))
	}

	if err := g.groups.Delete(ctx, actor, id); err != nil {
		return nil, toGroupMutationError("delete group", err)
	}

	return &metadatalifecyclev1.DeleteGroupResponse{}, nil
}

func toGroupError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidGroup):
		return fmt.Errorf("get group: %w", status.Error(codes.InvalidArgument, "invalid group id"))
	case errors.Is(err, domain.ErrGroupNotFound):
		return fmt.Errorf("get group: %w", status.Error(codes.NotFound, "group not found"))
	default:
		return fmt.Errorf("get group: %w", status.Error(codes.Internal, "get group"))
	}
}

func toGroupMutationError(operation string, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidGroup):
		return fmt.Errorf("%s: %w", operation, status.Error(codes.InvalidArgument, "invalid group"))
	case errors.Is(err, domain.ErrGroupNotFound):
		return fmt.Errorf("%s: %w", operation, status.Error(codes.NotFound, "group not found"))
	case errors.Is(err, domain.ErrGroupForbidden):
		return fmt.Errorf("%s: %w", operation, status.Error(codes.PermissionDenied, "group operation is not permitted"))
	case errors.Is(err, domain.ErrGroupConflict):
		return fmt.Errorf("%s: %w", operation, status.Error(codes.AlreadyExists, "group operation conflicts with existing data"))
	default:
		return fmt.Errorf("%s: %w", operation, status.Error(codes.Internal, operation))
	}
}

func toProtoGroup(value domain.Group) (*metadatalifecyclev1.Group, error) {
	concept, err := toProtoTopLevelConcept(value.Concept)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.Group{
		Id:          value.ID.String(),
		Name:        value.Name,
		Description: value.Description,
		Label:       value.Label,
		Logo:        value.Logo,
		Page:        value.Page,
		Color:       value.Color,
		Position:    value.Position,
		Concept:     concept,
	}, nil
}

func toProtoTopLevelConcept(value string) (metadatalifecyclev1.TopLevelConcept, error) {
	switch domain.TopLevelConcept(value) {
	case domain.TopLevelConceptData:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_DATA, nil
	case domain.TopLevelConceptFacilities:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_FACILITIES, nil
	case domain.TopLevelConceptSoftware:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_SOFTWARE, nil
	default:
		return 0, status.Error(codes.Internal, "map group concept")
	}
}
