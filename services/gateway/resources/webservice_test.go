package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapWebservice(t *testing.T) {
	t.Parallel()

	detail := loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/webservice/intermagnet.json")
	fetch := newFixtureAccessDetailFetcher(t, map[string]string{
		fixtureKey("webservice", "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Webservice/v1.0"):                    "testdata/metadata-lifecycle/access/webservice/intermagnet.json",
		fixtureKey("periodoftime", "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/bnode-periodoftime-0f4b2f01/v1.0"): "testdata/metadata-lifecycle/access/periodoftime/intermagnet.json",
		fixtureKey("location", "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/bnode-location-630d862a/v1.0"):         "testdata/metadata-lifecycle/access/location/intermagnet.json",
		fixtureKey("apidocumentation", "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/APIDocumentation/v1.0"):        "testdata/metadata-lifecycle/access/apidocumentation/intermagnet.json",
		fixtureKey("operation", "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Operation/v1.0"):                      "testdata/metadata-lifecycle/access/operation/intermagnet.json",
		fixtureKey("iritemplate", "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/bnode-iritemplate-2ff35bd4/v1.0"):   "testdata/metadata-lifecycle/access/iritemplate/intermagnet.json",
		fixtureKey("iritemplatemapping", "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-afe775b2/v1.0"):                                             "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-format.json",
		fixtureKey("iritemplatemapping", "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-d1713110/v1.0"):                                             "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-publication-state.json",
		fixtureKey("iritemplatemapping", "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-25a4198f/v1.0"):                                             "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-end-date.json",
		fixtureKey("iritemplatemapping", "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-73bc40eb/v1.0"):                                             "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-start-date.json",
		fixtureKey("iritemplatemapping", "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-e890a680/v1.0"):                                             "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-samples-per-day.json",
		fixtureKey("iritemplatemapping", "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-44b3a135/v1.0"):                                             "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-observatory-iaga-code.json",
	})

	webservice, err := mapWebservice(context.Background(), detail, fetch)
	if err != nil {
		t.Fatalf("mapWebservice() error = %v", err)
	}

	if webservice.ID != "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Webservice/v1.0" {
		t.Fatalf("webservice.ID = %q", webservice.ID)
	}
	if webservice.Name != "Web service for INTERMAGNET data" {
		t.Fatalf("webservice.Name = %q", webservice.Name)
	}
	if webservice.Description == "" {
		t.Fatalf("webservice.Description = %q", webservice.Description)
	}
	if webservice.Entrypoint != "https://imag-data.bgs.ac.uk/GIN_V1/" {
		t.Fatalf("webservice.Entrypoint = %q", webservice.Entrypoint)
	}
	if webservice.License != "https://creativecommons.org/licenses/by-nc/4.0/" {
		t.Fatalf("webservice.License = %q", webservice.License)
	}
	assertTimeEqual(t, webservice.Issued, "2018-08-31T00:00:00Z")
	assertTimeEqual(t, webservice.Modified, "2018-08-31T00:00:00Z")
	if webservice.Provider.Href != "v1/organization/http://viaf.org/viaf/136554353//v1.0" {
		t.Fatalf("webservice.Provider = %#v", webservice.Provider)
	}
	if len(webservice.ContactPoints) != 1 || webservice.ContactPoints[0].Href != "http://localhost:8080/api/v2/access/contactpoint/mailto%3Aochambers%40bgs.ac.uk%2FcontactPoint%2Fv1.0" {
		t.Fatalf("webservice.ContactPoints = %#v", webservice.ContactPoints)
	}
	if webservice.Spatial != "POLYGON((180.0 -90.0, -180.0 -90.0, -180.0 90.0, 180.0 90.0, 180.0 -90.0))" {
		t.Fatalf("webservice.Spatial = %q", webservice.Spatial)
	}
	if webservice.Temporal == nil {
		t.Fatal("webservice.Temporal = nil")
	}
	assertTimeEqual(t, webservice.Temporal.Start, "1991-01-01T00:00:00Z")
	if webservice.Temporal.End != nil {
		t.Fatalf("webservice.Temporal.End = %#v", webservice.Temporal.End)
	}
	if len(webservice.Concepts) != 0 {
		t.Fatalf("webservice.Concepts = %#v", webservice.Concepts)
	}
	if len(webservice.Identifiers) != 0 {
		t.Fatalf("webservice.Identifiers = %#v", webservice.Identifiers)
	}
	if webservice.APIDocumentation == nil || webservice.APIDocumentation.Title != "INTERMAGNET web service documentation" {
		t.Fatalf("webservice.APIDocumentation = %#v", webservice.APIDocumentation)
	}
	if len(webservice.Operations) != 1 {
		t.Fatalf("webservice.Operations = %#v", webservice.Operations)
	}
	if webservice.Operations[0].Method != "GET" {
		t.Fatalf("webservice.Operations[0] = %#v", webservice.Operations[0])
	}
	if len(webservice.Operations[0].Returns) != 6 {
		t.Fatalf("webservice.Operations[0].Returns = %#v", webservice.Operations[0].Returns)
	}
	if webservice.Operations[0].IriTemplate == nil || webservice.Operations[0].IriTemplate.Template == "" {
		t.Fatalf("webservice.Operations[0].IriTemplate = %#v", webservice.Operations[0].IriTemplate)
	}
	if len(webservice.Operations[0].IriTemplate.Mappings) != 6 {
		t.Fatalf("webservice.Operations[0].IriTemplate.Mappings = %#v", webservice.Operations[0].IriTemplate.Mappings)
	}
}

func TestMapWebserviceMissingName(t *testing.T) {
	t.Parallel()

	_, err := mapWebservice(context.Background(), &metadataclient.AccessDetailResponse{Properties: map[string]any{}}, nil)
	if err == nil {
		t.Fatal("mapWebservice() error = nil")
	}
}

func TestMapWebserviceInvalidOperationMappingRequired(t *testing.T) {
	t.Parallel()

	_, err := mapWebservice(context.Background(), &metadataclient.AccessDetailResponse{
		Properties: map[string]any{
			"schema:name": "Webservice title",
			"hydra:supportedOperation": []any{
				map[string]any{"id": "operation-1", "type": "operation"},
			},
		},
	}, func(ctx context.Context, shape string, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx
		switch {
		case shape == "operation" && id == "operation-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"hydra:pg_property": map[string]any{"id": "iri-template-1", "type": "iritemplate"},
			}}, nil
		case shape == "iritemplate" && id == "iri-template-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"template": "https://example.org/query{?station}",
				"mapping":  []any{map[string]any{"id": "mapping-1", "type": "iritemplatemapping"}},
			}}, nil
		case shape == "iritemplatemapping" && id == "mapping-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"variable": "station",
				"label":    "Station code",
				"range":    "xsd:string",
				"required": "not-a-bool",
			}}, nil
		default:
			return nil, fmt.Errorf("unexpected fetch %s %s", shape, id)
		}
	})
	if err == nil {
		t.Fatal("mapWebservice() error = nil")
	}
}
