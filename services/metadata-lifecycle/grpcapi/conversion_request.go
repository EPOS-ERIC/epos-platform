package grpcapi

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
)

func (s *ResourceServer) RequestDistributionConversion(ctx context.Context, request *metadatalifecyclev1.RequestDistributionConversionRequest) (*metadatalifecyclev1.RequestDistributionConversionResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "conversion request is required")
	}

	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	id, err := parseUUIDAt(request.GetDistributionId(), "distribution id", "distributionId")
	if err != nil {
		return nil, err
	}

	input, err := toDomainConversionRequestInput(request)
	if err != nil {
		return nil, err
	}

	notifications, err := s.datasets.RequestConversion(ctx, actor, id, input)
	if err != nil {
		return nil, err
	}

	values, err := mapSliceErr(notifications, toProtoNotification)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.RequestDistributionConversionResponse{Notifications: values}, nil
}
