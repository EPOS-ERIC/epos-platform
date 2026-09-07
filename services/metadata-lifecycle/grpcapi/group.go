package grpcapi

import (
	"context"
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
		return nil, wrapHandlerError(err)
	}

	result := make([]*metadatalifecyclev1.Group, len(values))
	for i, value := range values {
		mapped, err := toProtoGroup(value)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
	}

	return &metadatalifecyclev1.ListGroupsResponse{Groups: result}, nil
}

func (g *GroupServer) GetGroup(ctx context.Context, request *metadatalifecyclev1.GetGroupRequest) (*metadatalifecyclev1.GetGroupResponse, error) {
	id, err := uuid.Parse(request.GetGroupId())
	if err != nil || id == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "invalid group id")
	}

	value, err := g.groups.Get(ctx, id)
	if err != nil {
		return nil, wrapHandlerError(err)
	}

	group, err := toProtoGroup(value)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.GetGroupResponse{Group: group}, nil
}

func (g *GroupServer) CreateGroup(ctx context.Context, request *metadatalifecyclev1.CreateGroupRequest) (*metadatalifecyclev1.CreateGroupResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	concept, err := toDomainTopLevelConcept(request.GetConcept())
	if err != nil {
		return nil, err
	}
	input, err := groupInput(request.GetName(), request.Description, request.Label, request.Logo, request.Page, request.Color, request.GetPosition(), concept)
	if err != nil {
		return nil, err
	}
	group, err := g.groups.Create(ctx, actor, input)
	if err != nil {
		return nil, wrapHandlerError(err)
	}

	result, err := toProtoGroup(group)
	if err != nil {
		return nil, err
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
		return nil, status.Error(codes.InvalidArgument, "invalid group id")
	}

	concept, err := toDomainTopLevelConcept(request.GetConcept())
	if err != nil {
		return nil, err
	}
	input, err := groupInput(request.GetName(), request.Description, request.Label, request.Logo, request.Page, request.Color, request.GetPosition(), concept)
	if err != nil {
		return nil, err
	}
	group, err := g.groups.Replace(ctx, actor, id, input)
	if err != nil {
		return nil, wrapHandlerError(err)
	}

	result, err := toProtoGroup(group)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.ReplaceGroupResponse{Group: result}, nil
}

func groupInput(name string, description, label, logo, page, color *string, position int32, concept string) (domain.GroupInput, error) {
	result, err := domain.NewGroupInput(domain.GroupInput{
		Name:        name,
		Description: description,
		Label:       label,
		Logo:        logo,
		Page:        page,
		Color:       color,
		Position:    position,
		Concept:     concept,
	})
	if err != nil {
		return domain.GroupInput{}, fmt.Errorf("validate group input: %w", err)
	}
	return result, nil
}

func toDomainTopLevelConcept(value metadatalifecyclev1.TopLevelConcept) (string, error) {
	switch value {
	case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_DATA:
		return string(domain.TopLevelConceptData), nil
	case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_FACILITIES:
		return string(domain.TopLevelConceptFacilities), nil
	case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_SOFTWARE:
		return string(domain.TopLevelConceptSoftware), nil
	default:
		return "", domain.ErrInvalidGroup
	}
}

func (g *GroupServer) DeleteGroup(ctx context.Context, request *metadatalifecyclev1.DeleteGroupRequest) (*metadatalifecyclev1.DeleteGroupResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetGroupId())
	if err != nil || id == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "invalid group id")
	}

	if err := g.groups.Delete(ctx, actor, id); err != nil {
		return nil, wrapHandlerError(err)
	}

	return &metadatalifecyclev1.DeleteGroupResponse{}, nil
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
		return 0, fmt.Errorf("unsupported top-level concept %q", value)
	}
}
