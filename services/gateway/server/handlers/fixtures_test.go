package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/discovery"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/testfixtures"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/resources"
	"github.com/go-chi/chi/v5"
)

func newResourcesHandler(t *testing.T, routes map[string]string) *Resources {
	t.Helper()

	server := testfixtures.NewMetadataServer(t, routes)
	t.Cleanup(server.Close)

	client := testfixtures.NewMetadataClient(server.URL)
	return NewResources(resources.NewService(client))
}

func newDiscoveryHandler(t *testing.T, routes map[string]string) *Discovery {
	t.Helper()

	server := testfixtures.NewMetadataServer(t, routes)
	t.Cleanup(server.Close)

	client := testfixtures.NewMetadataClient(server.URL)
	return NewDiscovery(discovery.NewService(client, 8))
}

func requestWithURLParam(method string, target string, key string, value string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func resourceFixtureRoutes() map[string]string {
	return map[string]string{
		"/api/v2/access/softwareapplication/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2FCatalogToXls%2Fv1.0":           "testdata/metadata-lifecycle/access/softwareapplication/catalog-to-xls.json",
		"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fzenodo.17598403":                                             "testdata/metadata-lifecycle/access/softwaresourcecode/zenodo.json",
		"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fbnode-propertyvalue-0ebd586f":                                "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-0ebd586f.json",
		"/api/v2/access/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fbnode-propertyvalue-4c77d796":                                "testdata/metadata-lifecycle/access/softwaresourcecode/bnode-propertyvalue-4c77d796.json",
		"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-4c77d796%2Fv1.0": "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-doi.json",
		"/api/v2/access/propertyvalue/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2Fbnode-propertyvalue-b54da560%2Fv1.0": "testdata/metadata-lifecycle/access/propertyvalue/catalog-to-xls-ddss-id.json",
		"/api/v2/access/organization/ROR%3A048a87296%2Fv1.0":                                                                              "testdata/metadata-lifecycle/access/organization/uppsala.json",
		"/api/v2/access/propertyvalue/ROR%3A048a87296%2Fbnode-propertyvalue-14ab692e%2Fv1.0":                                              "testdata/metadata-lifecycle/access/propertyvalue/uppsala-url.json",
		"/api/v2/access/postaladdress/ROR%3A048a87296%2Fbnode-postaladdress-be765c92%2Fv1.0":                                              "testdata/metadata-lifecycle/access/postaladdress/uppsala.json",
		"/api/v2/access/conceptscheme/category%3Asoftware_jup%2Fv1.0":                                                                     "testdata/metadata-lifecycle/access/conceptscheme/software-jup.json",
		"/api/v2/access/concept/category%3Asoftware_jup_conc_multi%2Fv1.0":                                                                "testdata/metadata-lifecycle/access/concept/software-jup-conc-multi.json",
		"/api/v2/access/concept/category%3Asoftware_appl_conc_multi%2Fv1.0":                                                               "testdata/metadata-lifecycle/access/concept/software-appl-conc-multi.json",
		"/api/v2/access/concept/category%3Afacets%2Fsoftware-theme%2Fv1.0":                                                                "testdata/metadata-lifecycle/access/concept/software-theme.json",
	}
}

func conceptTreeFixtureRoutes() map[string]string {
	return map[string]string{
		"/api/v2/access/search/conceptscheme":                              "testdata/metadata-lifecycle/access/search/conceptscheme-software-first-page.json",
		"/api/v2/access/conceptscheme/category%3Asoftware_jup%2Fv1.0":      "testdata/metadata-lifecycle/access/conceptscheme/software-jup.json",
		"/api/v2/access/concept/category%3Afacets%2Fsoftware-theme%2Fv1.0": "testdata/metadata-lifecycle/access/concept/software-theme.json",
	}
}
