package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/testfixtures"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/resources"
)

func TestRouterSearchUsesFixtureResponses(t *testing.T) {
	t.Parallel()

	router := newFixtureRouter(t, map[string]string{
		"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/softwareapplication-first-page.json",
		"/api/v2/access/softwareapplication/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2FCatalogToXls%2Fv1.0":           "testdata/metadata-lifecycle/access/softwareapplication/catalog-to-xls.json",
		"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-4c77d796%2Fv1.0": "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-doi.json",
		"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-b54da560%2Fv1.0": "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-ddss-id.json",
		"/api/v2/access/organization/ROR%3A048a87296%2Fv1.0":                                                                              "testdata/metadata-lifecycle/access/organization/uppsala.json",
		"/api/v2/access/concept/category%3Asoftware_appl_conc_multi%2Fv1.0":                                                               "testdata/metadata-lifecycle/access/concept/software-appl-conc-multi.json",
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/search?type=softwareapplication", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Catalog to XLS converter") {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Uppsala University") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func newFixtureRouter(t *testing.T, routes map[string]string) http.Handler {
	t.Helper()

	server := testfixtures.NewMetadataServer(t, routes)
	t.Cleanup(server.Close)

	client := testfixtures.NewMetadataClient(server.URL)
	return NewRouter(discovery.NewService(client, 8), resources.NewService(client), "")
}
