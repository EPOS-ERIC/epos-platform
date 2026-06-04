package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/testfixtures"
)

const (
	currentAHEADDatasetID             = "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/events/v1.0"
	currentAHEADDistributionID        = "https://www.epos-eu.org/epos-dcat-ap/Seismology/Dataset/AHEAD/events/distribution/restful/v1.0"
	currentAHEADWebserviceID          = "https://www.epos-eu.org/epos-dcat-ap/Seismology/WebService/AHEAD/restful/fdsnws-event/v1.0"
	currentAHEADOperationID           = "https://www.epos-eu.org/epos-dcat-ap/Seismology/WebService/AHEAD/restful/fdsnws-event/Operation/v1.0"
	currentAHEADIriTemplateID         = "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplate-383ba416/v1.0"
	currentAHEADAPIDocumentationID    = "https://www.epos-eu.org/epos-dcat-ap/Seismology/WebService/AHEAD/APIDocumentation/fdsnws-event/v1.0"
	currentAHEADPeriodOfTimeID        = "https://epos-eu.org/anthropogenic_hazards/software/bnode-periodoftime-854f13c1/v1.0"
	currentAHEADWebserviceLocationID  = "https://epos-eu.org/anthropogenic_hazards/software/bnode-location-2df7160e/v1.0"
	currentAHEADLocationID            = "https://epos-eu.org/anthropogenic_hazards/software/bnode-location-f75195ba/v1.0"
	currentAHEADAnnotationID          = "https://epos-eu.org/anthropogenic_hazards/software/bnode-annotation-bb2b59f0/v1.0"
	currentAHEADPublisherID           = "PIC:999472675/v1.0"
	currentAHEADUppsalaID             = "ROR:048a87296/v1.0"
	currentAHEADDatasetIdentifierDDSS = "https://epos-eu.org/anthropogenic_hazards/software/bnode-identifier-c4177063/v1.0"
	currentAHEADDatasetIdentifierDOI  = "https://epos-eu.org/anthropogenic_hazards/software/bnode-identifier-c4212fb5/v1.0"
)

const (
	currentAHEADWebserviceHref   = "v1/webservice/" + currentAHEADWebserviceID
	currentAHEADDistributionHref = "v1/distribution/" + currentAHEADDistributionID
	currentAHEADPublisherHref    = "v1/organization/" + currentAHEADPublisherID
)

func loadAccessDetailFixture(t *testing.T, relativePath string) *metadataclient.AccessDetailResponse {
	t.Helper()
	return testfixtures.LoadAccessDetail(t, relativePath)
}

func newFixtureAccessDetailFetcher(t *testing.T, fixtures map[string]string) accessDetailFetcher {
	t.Helper()

	loaded := make(map[string]*metadataclient.AccessDetailResponse, len(fixtures))
	for key, fixturePath := range fixtures {
		loaded[key] = loadAccessDetailFixture(t, fixturePath)
	}

	return func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx

		key := fixtureKey(shape, id)
		detail, ok := loaded[key]
		if !ok {
			return nil, fmt.Errorf("missing fixture for %s", key)
		}

		return detail, nil
	}
}

func fixtureKey(shape, id string) string {
	return shape + "::" + id
}

func assertContactPointHrefs(t *testing.T, got []api.ResourceRef, wantA, wantB string) {
	t.Helper()

	if len(got) != 2 {
		t.Fatalf("contact points = %#v", got)
	}

	seen := map[string]int{
		wantA: 1,
		wantB: 1,
	}

	for _, contactPoint := range got {
		if seen[contactPoint.Href] == 0 {
			t.Fatalf("contact points = %#v", got)
		}
		seen[contactPoint.Href]--
	}

	if seen[wantA] != 0 || seen[wantB] != 0 {
		t.Fatalf("contact points = %#v", got)
	}
}

func fdsnEventFixturePaths() map[string]string {
	return map[string]string{
		fixtureKey("dataset", currentAHEADDatasetID):                                                                                  "testdata/metadata-lifecycle/access/dataset/intermagnet.json",
		fixtureKey("webservice", currentAHEADWebserviceID):                                                                            "testdata/metadata-lifecycle/access/webservice/intermagnet.json",
		fixtureKey("operation", currentAHEADOperationID):                                                                              "testdata/metadata-lifecycle/access/operation/intermagnet.json",
		fixtureKey("iritemplate", currentAHEADIriTemplateID):                                                                          "testdata/metadata-lifecycle/access/iritemplate/intermagnet.json",
		fixtureKey("periodoftime", currentAHEADPeriodOfTimeID):                                                                        "testdata/metadata-lifecycle/access/periodoftime/intermagnet.json",
		fixtureKey("location", currentAHEADWebserviceLocationID):                                                                      "testdata/metadata-lifecycle/access/location/webservice.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-bec6757f/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/nodata.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-6e1d03bf/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/limit.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-cc040606/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/includeallmagnitudes.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-ae32613b/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/includeallorigins.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-e296da48/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/orderby.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-c6edf57b/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/maxmagnitude.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-01f77125/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/minmagnitude.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-79219970/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/maxradiuskm.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-acd0818a/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/longitude.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-2fa4d342/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/latitude.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-99212efd/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/maxlongitude.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-59522638/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/minlongitude.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-c2ea443d/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/maxlatitude.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-c19cb3e0/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/minlatitude.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-b0c41f63/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/eventid.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-2f9a6fda/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/endtime.json",
		fixtureKey("iritemplatemapping", "https://epos-eu.org/anthropogenic_hazards/software/bnode-iritemplatemapping-9f1f2b7f/v1.0"): "testdata/metadata-lifecycle/access/iritemplatemapping/starttime.json",
		fixtureKey("apidocumentation", currentAHEADAPIDocumentationID):                                                                "testdata/metadata-lifecycle/access/apidocumentation/intermagnet.json",
	}
}
