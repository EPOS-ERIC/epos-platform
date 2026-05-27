package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapDistribution(t *testing.T) {
	t.Parallel()

	detail := &metadataclient.AccessDetailResponse{
		Id:    "distribution-1",
		Shape: "distribution",
		Href:  "/api/v2/access/distribution/distribution-1",
		Properties: map[string]any{
			titleKey:      "Distribution title",
			"description": "Distribution description",
			"issued":      "2024-03-04T05:06:07Z",
			"modified":    "2024-04-05T06:07:08Z",
			"license":     "https://creativecommons.org/licenses/by-nc/4.0/",
			"format":      "application/xml",
			"type": []any{
				genericDistributionTypeIRI,
				string(api.DistributionKindWebService),
			},
			"conformsTo": map[string]any{"id": "webservice-1", titleKey: "Webservice 1"},
			"accessURL":  map[string]any{"id": "operation-1", "type": "operation"},
		},
	}

	distribution, err := mapDistribution(context.Background(), detail, func(ctx context.Context, shape string, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx
		switch {
		case shape == "operation" && id == "operation-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"method":        "GET",
				"returns":       []any{"text/csv", "application/json"},
				"pgProperty":    map[string]any{"id": "iri-template-1", "type": "iritemplate"},
				"documentation": map[string]any{"id": "api-doc-1", titleKey: "API docs"},
			}}, nil
		case shape == "iritemplate" && id == "iri-template-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"template": "https://example.org/query{?station}",
				"mapping": []any{
					map[string]any{"id": "mapping-1", "type": "iritemplatemapping"},
					map[string]any{"id": "mapping-2", "type": "iritemplatemapping"},
				},
			}}, nil
		case shape == "iritemplatemapping" && id == "mapping-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"variable":     "station",
				"pg_label":     "Station code",
				"range":        "xsd:string",
				"required":     "true",
				"defaultValue": "ABC",
				"paramValue":   []any{"ABC", "DEF"},
			}}, nil
		case shape == "iritemplatemapping" && id == "mapping-2":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"variable":       "channel",
				"label":          "Channel",
				"range":          "xsd:string",
				"property":       "http://schema.org/name",
				"required":       "false",
				"minValue":       "1",
				"maxValue":       "3",
				"multipleValues": "true",
				"readonlyValue":  "false",
				"valuePattern":   "^[A-Z]+$",
			}}, nil
		case shape == "apidocumentation" && id == "api-doc-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				titleKey:      "API docs",
				"description": "Operation documentation",
				"entrypoint":  "https://example.org/docs",
			}}, nil
		default:
			return nil, fmt.Errorf("unexpected fetch %s %s", shape, id)
		}
	})
	if err != nil {
		t.Fatalf("mapDistribution() error = %v", err)
	}

	if distribution.ID != "distribution-1" {
		t.Fatalf("distribution.ID = %q", distribution.ID)
	}
	if distribution.Name != "Distribution title" {
		t.Fatalf("distribution.Name = %q", distribution.Name)
	}
	if distribution.Description != "Distribution description" {
		t.Fatalf("distribution.Description = %q", distribution.Description)
	}
	if distribution.DownloadURL != "" {
		t.Fatalf("distribution.DownloadURL = %q", distribution.DownloadURL)
	}
	if distribution.Format != "application/xml" {
		t.Fatalf("distribution.Format = %q", distribution.Format)
	}
	if distribution.License != "https://creativecommons.org/licenses/by-nc/4.0/" {
		t.Fatalf("distribution.License = %q", distribution.License)
	}
	if distribution.Type != api.DistributionKindWebService {
		t.Fatalf("distribution.Type = %q", distribution.Type)
	}
	assertTimeEqual(t, distribution.Issued, "2024-03-04T05:06:07Z")
	assertTimeEqual(t, distribution.Modified, "2024-04-05T06:07:08Z")
	if distribution.Webservice == nil || distribution.Webservice.Href != "v1/webservice/webservice-1" {
		t.Fatalf("distribution.Webservice = %#v", distribution.Webservice)
	}
	if distribution.Operation == nil {
		t.Fatal("distribution.Operation = nil")
	}
	if distribution.Operation.Method != "GET" {
		t.Fatalf("distribution.Operation.Method = %q", distribution.Operation.Method)
	}
	if len(distribution.Operation.Returns) != 2 || distribution.Operation.Returns[0] != "text/csv" || distribution.Operation.Returns[1] != "application/json" {
		t.Fatalf("distribution.Operation.Returns = %#v", distribution.Operation.Returns)
	}
	if distribution.Operation.IriTemplate == nil {
		t.Fatal("distribution.Operation.IriTemplate = nil")
	}
	if len(distribution.Operation.IriTemplate.Mappings) != 2 {
		t.Fatalf("distribution.Operation.IriTemplate.Mappings = %#v", distribution.Operation.IriTemplate.Mappings)
	}
	if distribution.Operation.IriTemplate.Mappings[0].Label != "Station code" {
		t.Fatalf("distribution.Operation.IriTemplate.Mappings[0] = %#v", distribution.Operation.IriTemplate.Mappings[0])
	}
	if !distribution.Operation.IriTemplate.Mappings[1].MultipleValues {
		t.Fatalf("distribution.Operation.IriTemplate.Mappings[1] = %#v", distribution.Operation.IriTemplate.Mappings[1])
	}
	if distribution.Operation.APIDocumentation == nil || distribution.Operation.APIDocumentation.Entrypoint != "https://example.org/docs" {
		t.Fatalf("distribution.Operation.APIDocumentation = %#v", distribution.Operation.APIDocumentation)
	}
}

func TestMapDistributionMissingTitle(t *testing.T) {
	t.Parallel()

	_, err := mapDistribution(context.Background(), &metadataclient.AccessDetailResponse{Properties: map[string]any{}}, nil)
	if err == nil {
		t.Fatal("mapDistribution() error = nil")
	}
}

func TestMapDistributionInvalidType(t *testing.T) {
	t.Parallel()

	_, err := mapDistribution(context.Background(), &metadataclient.AccessDetailResponse{
		Properties: map[string]any{
			titleKey: "Distribution title",
			"type":   []any{genericDistributionTypeIRI, "unsupported-type"},
		},
	}, nil)
	if err == nil {
		t.Fatal("mapDistribution() error = nil")
	}
}

func TestMapDistributionDownloadableFileDownloadURL(t *testing.T) {
	t.Parallel()

	distribution, err := mapDistribution(context.Background(), &metadataclient.AccessDetailResponse{
		Id: "distribution-download-1",
		Properties: map[string]any{
			titleKey:    "Downloadable distribution",
			"type":      []any{string(api.DistributionKindDownloadableFile)},
			"accessURL": []any{"https://example.org/download/file.xml"},
		},
	}, nil)
	if err != nil {
		t.Fatalf("mapDistribution() error = %v", err)
	}

	if distribution.DownloadURL != "https://example.org/download/file.xml" {
		t.Fatalf("distribution.DownloadURL = %q", distribution.DownloadURL)
	}
	if distribution.Operation != nil {
		t.Fatalf("distribution.Operation = %#v", distribution.Operation)
	}
	if distribution.Webservice != nil {
		t.Fatalf("distribution.Webservice = %#v", distribution.Webservice)
	}
}

func TestMapDistributionInvalidMappingRequired(t *testing.T) {
	t.Parallel()

	_, err := mapDistribution(context.Background(), &metadataclient.AccessDetailResponse{
		Properties: map[string]any{
			titleKey:    "Distribution title",
			"accessURL": map[string]any{"id": "operation-1", "type": "operation"},
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
		t.Fatal("mapDistribution() error = nil")
	}
}
