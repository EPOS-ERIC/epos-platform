package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapWebservice(t *testing.T) {
	t.Parallel()

	detail := &metadataclient.AccessDetailResponse{
		Id:    "webservice-1",
		Shape: "webservice",
		Href:  "/api/v2/access/webservice/webservice-1",
		Properties: map[string]any{
			"name":          "Webservice title",
			"description":   "Webservice description",
			"datePublished": "2024-03-04T05:06:07Z",
			"dateModified":  "2024-04-05T06:07:08Z",
			"license":       "https://creativecommons.org/licenses/by-nc/4.0/",
			"entrypoint":    "https://example.org/api",
			"keywords":      []any{"geomagnetism", "magnetism"},
			"identifier":    []any{"identifier-1"},
			"provider": []any{
				map[string]any{"id": "org-1", titleKey: "Org 1"},
				map[string]any{"id": "org-2", titleKey: "Org 2"},
			},
			"contactPoint": []any{map[string]any{"id": "cp-1", titleKey: "Contact 1", "href": "http://upstream/contact/1"}},
			"temporal":     map[string]any{"id": "temporal-1", typeKey: "periodoftime"},
			"spatial":      map[string]any{"id": "location-1", typeKey: "location"},
			"theme":        []any{map[string]any{"id": "concept-1", titleKey: "Concept 1"}},
			"conformsTo":   map[string]any{"id": "api-doc-1", "type": "apidocumentation"},
			"supportedOperation": []any{
				map[string]any{"id": "operation-1", "type": "operation"},
			},
		},
	}

	webservice, err := mapWebservice(context.Background(), detail, func(ctx context.Context, shape string, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx
		switch {
		case shape == "periodoftime" && id == "temporal-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{"start date": "2020-01-01T00:00:00Z", "end date": "2020-12-31T00:00:00Z"}}, nil
		case shape == "location" && id == "location-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{"geometry": "POINT(1 2)"}}, nil
		case shape == "apidocumentation" && id == "api-doc-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				titleKey:      "API docs",
				"description": "Operation documentation",
				"entrypoint":  "https://example.org/docs",
			}}, nil
		case shape == "operation" && id == "operation-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"method":     "GET",
				"returns":    []any{"text/csv", "application/json"},
				"pgProperty": map[string]any{"id": "iri-template-1", "type": "iritemplate"},
			}}, nil
		case shape == "iritemplate" && id == "iri-template-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"template": "https://example.org/query{?station}",
				"mapping":  []any{map[string]any{"id": "mapping-1", "type": "iritemplatemapping"}},
			}}, nil
		case shape == "iritemplatemapping" && id == "mapping-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"variable":   "station",
				"pg_label":   "Station code",
				"range":      "xsd:string",
				"required":   "true",
				"paramValue": []any{"AAA", "BBB"},
			}}, nil
		default:
			return nil, fmt.Errorf("unexpected fetch %s %s", shape, id)
		}
	})
	if err != nil {
		t.Fatalf("mapWebservice() error = %v", err)
	}

	if webservice.ID != "webservice-1" {
		t.Fatalf("webservice.ID = %q", webservice.ID)
	}
	if webservice.Name != "Webservice title" {
		t.Fatalf("webservice.Name = %q", webservice.Name)
	}
	if webservice.Description != "Webservice description" {
		t.Fatalf("webservice.Description = %q", webservice.Description)
	}
	if webservice.Entrypoint != "https://example.org/api" {
		t.Fatalf("webservice.Entrypoint = %q", webservice.Entrypoint)
	}
	if webservice.License != "https://creativecommons.org/licenses/by-nc/4.0/" {
		t.Fatalf("webservice.License = %q", webservice.License)
	}
	assertTimeEqual(t, webservice.Issued, "2024-03-04T05:06:07Z")
	assertTimeEqual(t, webservice.Modified, "2024-04-05T06:07:08Z")
	if webservice.Provider.Href != "v1/organization/org-1" {
		t.Fatalf("webservice.Provider = %#v", webservice.Provider)
	}
	if len(webservice.ContactPoints) != 1 || webservice.ContactPoints[0].Href != "http://upstream/contact/1" {
		t.Fatalf("webservice.ContactPoints = %#v", webservice.ContactPoints)
	}
	if webservice.Spatial != "POINT(1 2)" {
		t.Fatalf("webservice.Spatial = %q", webservice.Spatial)
	}
	if webservice.Temporal == nil {
		t.Fatal("webservice.Temporal = nil")
	}
	assertTimeEqual(t, webservice.Temporal.Start, "2020-01-01T00:00:00Z")
	assertTimeEqual(t, webservice.Temporal.End, "2020-12-31T00:00:00Z")
	if len(webservice.Concepts) != 1 || webservice.Concepts[0].Href != "v1/concept/concept-1" {
		t.Fatalf("webservice.Concepts = %#v", webservice.Concepts)
	}
	if len(webservice.Identifiers) != 1 || webservice.Identifiers[0].Value != "identifier-1" {
		t.Fatalf("webservice.Identifiers = %#v", webservice.Identifiers)
	}
	if webservice.APIDocumentation == nil || webservice.APIDocumentation.Entrypoint != "https://example.org/docs" {
		t.Fatalf("webservice.APIDocumentation = %#v", webservice.APIDocumentation)
	}
	if len(webservice.Operations) != 1 {
		t.Fatalf("webservice.Operations = %#v", webservice.Operations)
	}
	if webservice.Operations[0].Method != "GET" {
		t.Fatalf("webservice.Operations[0] = %#v", webservice.Operations[0])
	}
	if len(webservice.Operations[0].Returns) != 2 {
		t.Fatalf("webservice.Operations[0].Returns = %#v", webservice.Operations[0].Returns)
	}
	if webservice.Operations[0].IriTemplate == nil || webservice.Operations[0].IriTemplate.Template != "https://example.org/query{?station}" {
		t.Fatalf("webservice.Operations[0].IriTemplate = %#v", webservice.Operations[0].IriTemplate)
	}
	if len(webservice.Operations[0].IriTemplate.Mappings) != 1 || webservice.Operations[0].IriTemplate.Mappings[0].Label != "Station code" {
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
			"name": "Webservice title",
			"supportedOperation": []any{
				map[string]any{"id": "operation-1", "type": "operation"},
			},
		},
	}, func(ctx context.Context, shape string, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx
		switch {
		case shape == "operation" && id == "operation-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"pg_property": map[string]any{"id": "iri-template-1", "type": "iritemplate"},
			}}, nil
		case shape == "iritemplate" && id == "iri-template-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"template": "https://example.org/query{?station}",
				"mapping":  []any{map[string]any{"id": "mapping-1", "type": "iritemplatemapping"}},
			}}, nil
		case shape == "iritemplatemapping" && id == "mapping-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"variable": "station",
				"pg_label": "Station code",
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
