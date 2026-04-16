package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayserver"
	metadatalifecycle "github.com/EPOS-ERIC/epos-platform/services/gateway/services/metadata"
)

type apiAdapter struct {
	metadataService *metadatalifecycle.Service
}

func newAPIAdapter(metadataService *metadatalifecycle.Service) *apiAdapter {
	return &apiAdapter{
		metadataService: metadataService,
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
	return s.notImplemented()
}

func (s *apiAdapter) GetCategoryById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
}

func (s *apiAdapter) GetDatasetById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
}

func (s *apiAdapter) GetDistributionById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
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
	return s.notImplemented()
}

func (s *apiAdapter) GetSoftwareApplicationById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
}

func (s *apiAdapter) GetSoftwareSourceCodeById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
}
