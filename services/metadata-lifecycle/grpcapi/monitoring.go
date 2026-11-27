package grpcapi

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
)

func (s *ResourceServer) ListMonitoringDistributions(ctx context.Context, _ *metadatalifecyclev1.ListMonitoringDistributionsRequest) (*metadatalifecyclev1.ListMonitoringDistributionsResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	concepts, err := s.datasets.MonitoringDistributions(ctx, actor)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.ListMonitoringDistributionsResponse{
		Concepts: toProtoMonitoringConcepts(concepts),
	}, nil
}
