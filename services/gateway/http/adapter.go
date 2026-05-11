package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayserver"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/resources"
)

type apiAdapter struct {
	discoveryService *discovery.Service
	resourcesService *resources.Service
}

func newAPIAdapter(discoveryService *discovery.Service, resourcesService *resources.Service) *apiAdapter {
	return &apiAdapter{
		discoveryService: discoveryService,
		resourcesService: resourcesService,
	}
}

// TODO: replace notImplemented responses as endpoints are wired to upstream services.
func (s *apiAdapter) notImplemented() (gatewayserver.ImplResponse, error) {
	return gatewayserver.Response(http.StatusNotImplemented, gatewayserver.ErrorResponse{
		ErrorCode:    "not_implemented",
		ErrorMessage: "endpoint not implemented",
	}), nil
}

func (s *apiAdapter) GetCategoryTree(_ context.Context) (gatewayserver.ImplResponse, error) {
	return s.notImplementedFromError(s.discoveryService.GetCategoryTree())
}

func (s *apiAdapter) GetCategoryById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplementedFromError(s.resourcesService.GetCategoryByID())
}

func (s *apiAdapter) GetDatasetById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplementedFromError(s.resourcesService.GetDatasetByID())
}

func (s *apiAdapter) GetDistributionById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplementedFromError(s.resourcesService.GetDistributionByID())
}

func (s *apiAdapter) SearchMetadata(
	_ context.Context,
	_ string,
	_ int32,
	_ int32,
	_ []gatewayserver.ResourceType,
	_ string,
	_ time.Time,
	_ time.Time,
	_ []string,
	_ []gatewayserver.VersioningStatus,
	_ []string,
) (gatewayserver.ImplResponse, error) {
	return s.notImplementedFromError(s.discoveryService.Search())
}

func (s *apiAdapter) GetSoftwareApplicationById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplementedFromError(s.resourcesService.GetSoftwareApplicationByID())
}

func (s *apiAdapter) GetSoftwareSourceCodeById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplementedFromError(s.resourcesService.GetSoftwareSourceCodeByID())
}

func (s *apiAdapter) notImplementedFromError(err error) (gatewayserver.ImplResponse, error) {
	if err == nil {
		return s.notImplemented()
	}

	if errors.Is(err, domain.ErrNotImplemented) {
		return s.notImplemented()
	}

	return gatewayserver.ImplResponse{}, err
}
