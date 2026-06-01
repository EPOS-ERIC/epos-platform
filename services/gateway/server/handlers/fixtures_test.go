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
		"/api/v2/access/dataset/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2FDataset%2Fv1.0":                          "testdata/metadata-lifecycle/access/dataset/intermagnet.json",
		"/api/v2/access/distribution/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2FDistribution%2Fv1.0":                "testdata/metadata-lifecycle/access/distribution/intermagnet.json",
		"/api/v2/access/webservice/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2FWebservice%2Fv1.0":                    "testdata/metadata-lifecycle/access/webservice/intermagnet.json",
		"/api/v2/access/organization/http%3A%2F%2Fviaf.org%2Fviaf%2F136554353%2F%2Fv1.0":                                                                                    "testdata/metadata-lifecycle/access/organization/bgs.json",
		"/api/v2/access/concept/category%3Ageomagneticdata%2Fv1.0":                                                                                                          "testdata/metadata-lifecycle/access/concept/geomagneticdata.json",
		"/api/v2/access/conceptscheme/category%3Ageomagneticobservatis%2Fv1.0":                                                                                              "testdata/metadata-lifecycle/access/conceptscheme/geomagneticobservations.json",
		"/api/v2/access/periodoftime/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2Fbnode-periodoftime-0f4b2f01%2Fv1.0": "testdata/metadata-lifecycle/access/periodoftime/intermagnet.json",
		"/api/v2/access/location/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2Fbnode-location-630d862a%2Fv1.0":         "testdata/metadata-lifecycle/access/location/intermagnet.json",
		"/api/v2/access/annotation/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2Fbnode-annotation-0970cbef%2Fv1.0":     "testdata/metadata-lifecycle/access/annotation/intermagnet.json",
		"/api/v2/access/identifier/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2Fbnode-identifier-34096b33%2Fv1.0":     "testdata/metadata-lifecycle/access/identifier/intermagnet-doi.json",
		"/api/v2/access/identifier/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2Fbnode-identifier-70ea05a7%2Fv1.0":     "testdata/metadata-lifecycle/access/identifier/intermagnet-timespan-doi.json",
		"/api/v2/access/identifier/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2Fbnode-identifier-b1f9aa83%2Fv1.0":     "testdata/metadata-lifecycle/access/identifier/intermagnet-ddss-id.json",
		"/api/v2/access/operation/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2FOperation%2Fv1.0":                      "testdata/metadata-lifecycle/access/operation/intermagnet.json",
		"/api/v2/access/apidocumentation/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2FAPIDocumentation%2Fv1.0":        "testdata/metadata-lifecycle/access/apidocumentation/intermagnet.json",
		"/api/v2/access/iritemplate/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2Fbnode-iritemplate-2ff35bd4%2Fv1.0":   "testdata/metadata-lifecycle/access/iritemplate/intermagnet.json",
		"/api/v2/access/iritemplatemapping/mailto%3Aochambers%40bgs.ac.uk%2Fbnode-iritemplatemapping-afe775b2%2Fv1.0":                                                       "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-format.json",
		"/api/v2/access/iritemplatemapping/mailto%3Aochambers%40bgs.ac.uk%2Fbnode-iritemplatemapping-d1713110%2Fv1.0":                                                       "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-publication-state.json",
		"/api/v2/access/iritemplatemapping/mailto%3Aochambers%40bgs.ac.uk%2Fbnode-iritemplatemapping-25a4198f%2Fv1.0":                                                       "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-end-date.json",
		"/api/v2/access/iritemplatemapping/mailto%3Aochambers%40bgs.ac.uk%2Fbnode-iritemplatemapping-73bc40eb%2Fv1.0":                                                       "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-start-date.json",
		"/api/v2/access/iritemplatemapping/mailto%3Aochambers%40bgs.ac.uk%2Fbnode-iritemplatemapping-e890a680%2Fv1.0":                                                       "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-samples-per-day.json",
		"/api/v2/access/iritemplatemapping/mailto%3Aochambers%40bgs.ac.uk%2Fbnode-iritemplatemapping-44b3a135%2Fv1.0":                                                       "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-observatory-iaga-code.json",
		"/api/v2/access/propertyvalue/http%3A%2F%2Fviaf.org%2Fviaf%2F136554353%2Fbnode-propertyvalue-5bbce760%2Fv1.0":                                                       "testdata/metadata-lifecycle/access/propertyvalue/bgs-viaf.json",
		"/api/v2/access/postaladdress/http%3A%2F%2Fviaf.org%2Fviaf%2F136554353%2Fbnode-postaladdress-064b9034%2Fv1.0":                                                       "testdata/metadata-lifecycle/access/postaladdress/bgs.json",
	}
}

func conceptTreeFixtureRoutes() map[string]string {
	return map[string]string{
		"/api/v2/access/search/conceptscheme":                                 "testdata/metadata-lifecycle/access/search/conceptscheme-first-page.json",
		"/api/v2/access/conceptscheme/category%3Aanthropogenichazards%2Fv1.0": "testdata/metadata-lifecycle/access/conceptscheme/anthropogenichazards.json",
		"/api/v2/access/concept/category%3Afacets%2Fdataset-theme%2Fv1.0":     "testdata/metadata-lifecycle/access/concept/dataset-theme.json",
	}
}
