package grpcapi

import (
	"context"
	"errors"
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DiscoveryServer struct {
	metadatalifecyclev1.UnimplementedDiscoveryServiceServer
	discovery *discovery.Service
}

func NewDiscovery(discovery *discovery.Service) *DiscoveryServer {
	return &DiscoveryServer{discovery: discovery}
}

func (s *DiscoveryServer) SearchResources(ctx context.Context, request *metadatalifecyclev1.SearchResourcesRequest) (*metadatalifecyclev1.SearchResourcesResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "discovery request is required") //nolint:wrapcheck
	}

	actor, err := optionalActorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	filter, err := toDomainDiscoveryFilter(request)
	if err != nil {
		return nil, err
	}

	items, err := s.discovery.Search(ctx, actor, filter)
	if err != nil {
		return nil, discoveryError(err)
	}

	response, err := toProtoDiscoveryResponse(items)
	if err != nil {
		return nil, discoveryError(err)
	}
	return response, nil
}

func discoveryError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidDiscovery):
		return fmt.Errorf("discovery request: %w", status.Error(codes.InvalidArgument, "invalid discovery request"))
	case errors.Is(err, domain.ErrDiscoveryDenied):
		return fmt.Errorf("discovery authorization: %w", status.Error(codes.Unauthenticated, "authentication is required for non-published discovery results"))
	default:
		return fmt.Errorf("discovery search: %w", status.Error(codes.Internal, "discovery search failed"))
	}
}
