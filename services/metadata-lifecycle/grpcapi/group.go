package grpcapi

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/groups"
	"github.com/google/uuid"
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
		return nil, err
	}

	result := make([]*metadatalifecyclev1.Group, len(values))
	for i, value := range values {
		result[i] = toProtoGroup(value)
	}

	return &metadatalifecyclev1.ListGroupsResponse{Groups: result}, nil
}

func (g *GroupServer) GetGroup(ctx context.Context, request *metadatalifecyclev1.GetGroupRequest) (*metadatalifecyclev1.GetGroupResponse, error) {
	id, err := uuid.Parse(request.GetGroupId())
	if err != nil || id == uuid.Nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "invalid group id")
	}

	value, err := g.groups.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.GetGroupResponse{Group: toProtoGroup(value)}, nil
}

func (g *GroupServer) CreateGroup(ctx context.Context, request *metadatalifecyclev1.CreateGroupRequest) (*metadatalifecyclev1.CreateGroupResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	input, err := groupInput(request.GetName(), request.Description, request.Label, request.Logo, request.Page, request.Color, request.GetPosition(), request.GetConceptId())
	if err != nil {
		return nil, err
	}
	group, err := g.groups.Create(ctx, actor, input)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.CreateGroupResponse{Group: toProtoGroup(group)}, nil
}

func (g *GroupServer) ReplaceGroup(ctx context.Context, request *metadatalifecyclev1.ReplaceGroupRequest) (*metadatalifecyclev1.ReplaceGroupResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetGroupId())
	if err != nil || id == uuid.Nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "invalid group id")
	}

	input, err := groupInput(request.GetName(), request.Description, request.Label, request.Logo, request.Page, request.Color, request.GetPosition(), request.GetConceptId())
	if err != nil {
		return nil, err
	}
	group, err := g.groups.Replace(ctx, actor, id, input)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.ReplaceGroupResponse{Group: toProtoGroup(group)}, nil
}

func groupInput(name string, description, label, logo, page, color *string, position int32, concept string) (domain.GroupInput, error) {
	conceptID, err := uuid.Parse(concept)
	if err != nil || conceptID == uuid.Nil {
		return domain.GroupInput{}, apperror.Field("concept_id", "Concept ID must be a valid UUID.", err)
	}

	result, err := domain.NewGroupInput(domain.GroupInput{
		Name:        name,
		Description: description,
		Label:       label,
		Logo:        logo,
		Page:        page,
		Color:       color,
		Position:    position,
		ConceptID:   conceptID,
	})
	if err != nil {
		return domain.GroupInput{}, fmt.Errorf("validate group input: %w", err)
	}

	return result, nil
}

func (g *GroupServer) DeleteGroup(ctx context.Context, request *metadatalifecyclev1.DeleteGroupRequest) (*metadatalifecyclev1.DeleteGroupResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetGroupId())
	if err != nil || id == uuid.Nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "invalid group id")
	}

	if err := g.groups.Delete(ctx, actor, id); err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.DeleteGroupResponse{}, nil
}

func toProtoGroup(value domain.Group) *metadatalifecyclev1.Group {
	return &metadatalifecyclev1.Group{
		Id:          value.ID.String(),
		Name:        value.Name,
		Description: value.Description,
		Label:       value.Label,
		Logo:        value.Logo,
		Page:        value.Page,
		Color:       value.Color,
		Position:    value.Position,
		ConceptId:   value.ConceptID.String(),
	}
}
