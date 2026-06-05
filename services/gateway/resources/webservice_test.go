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
	fixtures := fdsnEventFixturePaths()
	fixtures[fixtureKey("webservice", currentAHEADWebserviceID)] = "testdata/metadata-lifecycle/access/webservice/intermagnet.json"
	fetch := newFixtureAccessDetailFetcher(t, fixtures)

	webservice, err := mapWebservice(context.Background(), detail, fetch)
	if err != nil {
		t.Fatalf("mapWebservice() error = %v", err)
	}

	if webservice.ID != currentAHEADWebserviceID {
		t.Fatalf("webservice.ID = %q", webservice.ID)
	}
	if webservice.Name != "TEST1- Historical earthquakes 1000-1899 (FDSN-event)" {
		t.Fatalf("webservice.Name = %q", webservice.Name)
	}
	if webservice.Description == "" {
		t.Fatalf("webservice.Description = %q", webservice.Description)
	}
	if webservice.Entrypoint != "https://www.emidius.eu/fdsnws/event/1/application.wadl" {
		t.Fatalf("webservice.Entrypoint = %q", webservice.Entrypoint)
	}
	if webservice.License != "https://creativecommons.org/licenses/by/4.0/" {
		t.Fatalf("webservice.License = %q", webservice.License)
	}
	assertTimeEqual(t, webservice.Issued, "2012-12-01T00:00:00Z")
	assertTimeEqual(t, webservice.Modified, "2018-09-10T00:00:00Z")
	if webservice.Provider.Href != currentAHEADPublisherHref {
		t.Fatalf("webservice.Provider = %#v", webservice.Provider)
	}
	if webservice.Spatial != "POLYGON((-33.5 73.5,34.0 73.5,34.0 33.5,-33.5 33.5,-33.5 73.5))" {
		t.Fatalf("webservice.Spatial = %q", webservice.Spatial)
	}
	if webservice.Temporal == nil {
		t.Fatal("webservice.Temporal = nil")
	}
	assertTimeEqual(t, webservice.Temporal.Start, "1000-01-01T00:00:00Z")
	assertTimeEqual(t, webservice.Temporal.End, "1899-12-31T23:59:59Z")
	if len(webservice.Concepts) != 0 {
		t.Fatalf("webservice.Concepts = %#v", webservice.Concepts)
	}
	if len(webservice.Identifiers) != 0 {
		t.Fatalf("webservice.Identifiers = %#v", webservice.Identifiers)
	}
	if webservice.APIDocumentation == nil || webservice.APIDocumentation.Title != "ahead-fdsnws-event documentation" {
		t.Fatalf("webservice.APIDocumentation = %#v", webservice.APIDocumentation)
	}
	if len(webservice.Operations) != 1 {
		t.Fatalf("webservice.Operations = %#v", webservice.Operations)
	}
	if webservice.Operations[0].Method != "GET" {
		t.Fatalf("webservice.Operations[0] = %#v", webservice.Operations[0])
	}
	if len(webservice.Operations[0].Returns) != 1 || webservice.Operations[0].Returns[0] != "application/xml" {
		t.Fatalf("webservice.Operations[0].Returns = %#v", webservice.Operations[0].Returns)
	}
	if webservice.Operations[0].IriTemplate == nil || webservice.Operations[0].IriTemplate.Template == "" {
		t.Fatalf("webservice.Operations[0].IriTemplate = %#v", webservice.Operations[0].IriTemplate)
	}
	if len(webservice.Operations[0].IriTemplate.Mappings) != 17 {
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
