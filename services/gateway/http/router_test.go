package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/resources"
)

func TestRouterServesStubbedAPIEndpoints(t *testing.T) {
	t.Parallel()

	router := NewRouter(discovery.NewService(nil), resources.NewService())

	tests := []struct {
		name   string
		path   string
		status int
	}{
		{name: "search", path: "/v1/search", status: http.StatusNotImplemented},
		{name: "concept tree", path: "/v1/concept", status: http.StatusNotImplemented},
		{name: "concept by id", path: "/v1/concept/123", status: http.StatusNotImplemented},
		{name: "dataset by id", path: "/v1/dataset/123", status: http.StatusNotImplemented},
		{name: "distribution by id", path: "/v1/distribution/123", status: http.StatusNotImplemented},
		{name: "software application by id", path: "/v1/software-application/123", status: http.StatusNotImplemented},
		{name: "software source code by id", path: "/v1/software-source-code/123", status: http.StatusNotImplemented},
		{name: "openapi spec", path: "/openapi.yaml", status: http.StatusOK},
		{name: "swagger redirect", path: "/swagger", status: http.StatusMovedPermanently},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != tc.status {
				t.Fatalf("status = %d, want %d", resp.Code, tc.status)
			}
		})
	}
}
