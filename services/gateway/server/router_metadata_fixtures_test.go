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
		"/api/v2/access/search": "testdata/metadata-lifecycle/access/search/dataset-first-page.json",
		"/api/v2/access/organization/http%3A%2F%2Fviaf.org%2Fviaf%2F136554353%2F%2Fv1.0":                                                           "testdata/metadata-lifecycle/access/organization/bgs.json",
		"/api/v2/access/concept/category%3Ageomagneticdata%2Fv1.0":                                                                                 "testdata/metadata-lifecycle/access/concept/geomagneticdata.json",
		"/api/v2/access/dataset/https%3A%2F%2Fwww.epos-eu.org%2Fepos-dcat-ap%2FGeoElectroMagnetism%2FWP13-DDSS-001%2FINTERMAGNET%2FDataset%2Fv1.0": "testdata/metadata-lifecycle/access/dataset/intermagnet.json",
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/search?type=dataset", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "INTERMAGNET Geomagnetic Observatory Data") {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Geomagnetic data") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func newFixtureRouter(t *testing.T, routes map[string]string) http.Handler {
	t.Helper()

	server := testfixtures.NewMetadataServer(t, routes)
	t.Cleanup(server.Close)

	client := testfixtures.NewMetadataClient(server.URL)
	return NewRouter(discovery.NewService(client, 8), resources.NewService(client))
}
