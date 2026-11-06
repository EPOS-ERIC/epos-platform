package grpcapi

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/contactpoints"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type ContactPointServer struct {
	metadatalifecyclev1.UnimplementedContactPointServiceServer
	contactPoints *contactpoints.Service
}

func NewContactPoint(contactPoints *contactpoints.Service) *ContactPointServer {
	return &ContactPointServer{contactPoints: contactPoints}
}

func (c *ContactPointServer) ListContactPoints(ctx context.Context, _ *metadatalifecyclev1.ListContactPointsRequest) (*metadatalifecyclev1.ListContactPointsResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	values, err := c.contactPoints.List(ctx, actor)
	if err != nil {
		return nil, err
	}
	result := make([]*metadatalifecyclev1.ContactPoint, len(values))
	for i, value := range values {
		result[i] = toProtoContactPoint(value)
	}
	return &metadatalifecyclev1.ListContactPointsResponse{ContactPoints: result}, nil
}

func (c *ContactPointServer) GetContactPoint(ctx context.Context, request *metadatalifecyclev1.GetContactPointRequest) (*metadatalifecyclev1.GetContactPointResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseContactPointID(request.GetContactPointId())
	if err != nil {
		return nil, err
	}
	value, err := c.contactPoints.Get(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.GetContactPointResponse{ContactPoint: toProtoContactPoint(value)}, nil
}

func (c *ContactPointServer) CreateContactPoint(ctx context.Context, request *metadatalifecyclev1.CreateContactPointRequest) (*metadatalifecyclev1.CreateContactPointResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	input, err := contactPointInput(request.GetEmail())
	if err != nil {
		return nil, err
	}
	value, err := c.contactPoints.Create(ctx, actor, input)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.CreateContactPointResponse{ContactPoint: toProtoContactPoint(value)}, nil
}

func (c *ContactPointServer) ReplaceContactPoint(ctx context.Context, request *metadatalifecyclev1.ReplaceContactPointRequest) (*metadatalifecyclev1.ReplaceContactPointResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseContactPointID(request.GetContactPointId())
	if err != nil {
		return nil, err
	}
	input, err := contactPointInput(request.GetEmail())
	if err != nil {
		return nil, err
	}
	value, err := c.contactPoints.Replace(ctx, actor, id, input)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.ReplaceContactPointResponse{ContactPoint: toProtoContactPoint(value)}, nil
}

func (c *ContactPointServer) DeleteContactPoint(ctx context.Context, request *metadatalifecyclev1.DeleteContactPointRequest) (*metadatalifecyclev1.DeleteContactPointResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseContactPointID(request.GetContactPointId())
	if err != nil {
		return nil, err
	}
	if err := c.contactPoints.Delete(ctx, actor, id); err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.DeleteContactPointResponse{}, nil
}

func contactPointInput(email string) (domain.ContactPointInput, error) {
	result, err := domain.NewContactPointInput(domain.ContactPointInput{Email: email})
	if err != nil {
		return domain.ContactPointInput{}, fmt.Errorf("validate contact point input: %w", err)
	}
	return result, nil
}

func parseContactPointID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperror.New(apperror.CodeValidationFailed, "invalid contact point id")
	}
	return id, nil
}

func toProtoContactPoint(value domain.ContactPoint) *metadatalifecyclev1.ContactPoint {
	return &metadatalifecyclev1.ContactPoint{
		Id:    value.ID.String(),
		Email: value.Email,
	}
}
