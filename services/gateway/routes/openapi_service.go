package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayserver"
)

type openAPIService struct{}

func newOpenAPIService() *openAPIService {
	return &openAPIService{}
}

// TODO: implement the handlers and remove this
func (s *openAPIService) notImplemented() (gatewayserver.ImplResponse, error) {
	return gatewayserver.Response(http.StatusNotImplemented, gatewayserver.ErrorResponse{
		ErrorCode:    "not_implemented",
		ErrorMessage: "endpoint not implemented",
	}), nil
}

func (s *openAPIService) GetCategoryTree(_ context.Context) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
}

func (s *openAPIService) GetCategoryById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
}

func (s *openAPIService) GetDatasetById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
}

func (s *openAPIService) GetDistributionById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
}

func (s *openAPIService) SearchMetadata(
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

func (s *openAPIService) GetSoftwareApplicationById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
}

func (s *openAPIService) GetSoftwareSourceCodeById(_ context.Context, _ string) (gatewayserver.ImplResponse, error) {
	return s.notImplemented()
}
