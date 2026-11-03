package grpcapi

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatav1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
)

func (s *ResourceServer) GetDistributionFormats(ctx context.Context, request *metadatav1.GetDistributionFormatsRequest) (*metadatav1.GetDistributionFormatsResponse, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}

	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	id, err := parseUUIDAt(request.GetDistributionId(), "distribution id", "distribution_id")
	if err != nil {
		return nil, err
	}

	formats, err := s.datasets.DistributionFormats(ctx, actor, id)
	if err != nil {
		return nil, err
	}

	return &metadatav1.GetDistributionFormatsResponse{Formats: formats}, nil
}
