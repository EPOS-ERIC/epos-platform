package grpcapi

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/activity"
)

type ActivityServer struct {
	metadatalifecyclev1.UnimplementedActivityServiceServer
	activity *activity.Service
}

func NewActivity(activity *activity.Service) *ActivityServer {
	return &ActivityServer{activity: activity}
}

func (s *ActivityServer) ListActivity(ctx context.Context, request *metadatalifecyclev1.ListActivityRequest) (*metadatalifecyclev1.ListActivityResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "activity request is required")
	}

	actor, err := optionalActorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}

	filter, err := toDomainActivityFilter(request)
	if err != nil {
		return nil, err
	}

	events, err := s.activity.List(ctx, *actor, filter)
	if err != nil {
		return nil, err
	}

	response := &metadatalifecyclev1.ListActivityResponse{Items: make([]*metadatalifecyclev1.ActivityEvent, 0, len(events))}
	for _, event := range events {
		item, err := toProtoActivityEvent(event)
		if err != nil {
			return nil, err
		}
		response.Items = append(response.Items, item)
	}

	return response, nil
}
