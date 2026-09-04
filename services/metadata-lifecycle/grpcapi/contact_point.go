package grpcapi

import (
	"context"
	"errors"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/contactpoints"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, toContactPointError("list contact points", err)
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
		return nil, toContactPointError("get contact point", err)
	}
	return &metadatalifecyclev1.GetContactPointResponse{ContactPoint: toProtoContactPoint(value)}, nil
}

func (c *ContactPointServer) CreateContactPoint(ctx context.Context, request *metadatalifecyclev1.CreateContactPointRequest) (*metadatalifecyclev1.CreateContactPointResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	value, err := c.contactPoints.Create(ctx, actor, domain.ContactPointInput{Email: request.GetEmail(), Role: request.GetRole()})
	if err != nil {
		return nil, toContactPointError("create contact point", err)
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
	value, err := c.contactPoints.Replace(ctx, actor, id, domain.ContactPointInput{Email: request.GetEmail(), Role: request.GetRole()})
	if err != nil {
		return nil, toContactPointError("replace contact point", err)
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
		return nil, toContactPointError("delete contact point", err)
	}
	return &metadatalifecyclev1.DeleteContactPointResponse{}, nil
}

func parseContactPointID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "invalid contact point id")
	}
	return id, nil
}

func toProtoContactPoint(value domain.ContactPoint) *metadatalifecyclev1.ContactPoint {
	return &metadatalifecyclev1.ContactPoint{Id: value.ID.String(), Email: value.Email, Role: value.Role}
}

func toContactPointError(operation string, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidContactPoint):
		return status.Error(codes.InvalidArgument, "invalid contact point")
	case errors.Is(err, domain.ErrContactPointForbidden):
		return status.Error(codes.PermissionDenied, "contact point operation is not permitted")
	case errors.Is(err, domain.ErrContactPointNotFound):
		return status.Error(codes.NotFound, "contact point not found")
	case errors.Is(err, domain.ErrContactPointConflict):
		return status.Error(codes.AlreadyExists, "contact point already exists")
	default:
		return status.Error(codes.Internal, operation)
	}
}
