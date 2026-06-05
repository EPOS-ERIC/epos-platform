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

	detail := loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/distribution/intermagnet.json")
	fixtures := fdsnEventFixturePaths()
	fixtures[fixtureKey("distribution", currentAHEADDistributionID)] = "testdata/metadata-lifecycle/access/distribution/intermagnet.json"
	fetch := newFixtureAccessDetailFetcher(t, fixtures)

	distribution, err := mapDistribution(context.Background(), detail, fetch)
	if err != nil {
		t.Fatalf("mapDistribution() error = %v", err)
	}

	if distribution.ID != currentAHEADDistributionID {
		t.Fatalf("distribution.ID = %q", distribution.ID)
	}
	if distribution.Name != "TEST1 - Parameters of historical earthquakes (1000-1899) - FDSN event" {
		t.Fatalf("distribution.Name = %q", distribution.Name)
	}
	if distribution.Description == "" {
		t.Fatalf("distribution.Description = %q", distribution.Description)
	}
	if distribution.DownloadURL != "" {
		t.Fatalf("distribution.DownloadURL = %q", distribution.DownloadURL)
	}
	if distribution.Format != "http://publications.europa.eu/resource/authority/file-type/XML" {
		t.Fatalf("distribution.Format = %q", distribution.Format)
	}
	if distribution.License != "https://creativecommons.org/licenses/by/4.0/" {
		t.Fatalf("distribution.License = %q", distribution.License)
	}
	if distribution.Type != api.DistributionKindWebService {
		t.Fatalf("distribution.Type = %q", distribution.Type)
	}
	assertTimeEqual(t, distribution.Issued, "2017-10-25T00:00:00Z")
	assertTimeEqual(t, distribution.Modified, "2017-10-25T00:00:00Z")
	if distribution.Webservice == nil || distribution.Webservice.Href != currentAHEADWebserviceHref {
		t.Fatalf("distribution.Webservice = %#v", distribution.Webservice)
	}
	if distribution.Operation == nil {
		t.Fatal("distribution.Operation = nil")
	}
	if distribution.Operation.Method != "GET" {
		t.Fatalf("distribution.Operation.Method = %q", distribution.Operation.Method)
	}
	if len(distribution.Operation.Returns) != 1 || distribution.Operation.Returns[0] != "application/xml" {
		t.Fatalf("distribution.Operation.Returns = %#v", distribution.Operation.Returns)
	}
	if distribution.Operation.IriTemplate == nil {
		t.Fatal("distribution.Operation.IriTemplate = nil")
	}
	if distribution.Operation.IriTemplate.Template == "" {
		t.Fatalf("distribution.Operation.IriTemplate = %#v", distribution.Operation.IriTemplate)
	}
	if len(distribution.Operation.IriTemplate.Mappings) != 17 {
		t.Fatalf("distribution.Operation.IriTemplate.Mappings = %#v", distribution.Operation.IriTemplate.Mappings)
	}
	var foundRequired bool
	for _, mapping := range distribution.Operation.IriTemplate.Mappings {
		if mapping.Variable == "starttime" {
			foundRequired = mapping.Required && mapping.Label == "Start time"
		}
	}
	if !foundRequired {
		t.Fatalf("distribution.Operation.IriTemplate.Mappings = %#v", distribution.Operation.IriTemplate.Mappings)
	}
	if distribution.Operation.APIDocumentation != nil {
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
			titleKey:   distributionTitle,
			"dct:type": []any{genericDistributionTypeIRI, "unsupported-type"},
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
			titleKey:     "Downloadable distribution",
			"dct:type":   []any{string(api.DistributionKindDownloadableFile)},
			accessURLKey: []any{"https://example.org/download/file.xml"},
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
			titleKey:     distributionTitle,
			accessURLKey: map[string]any{"id": operationID1, typeKey: operationType},
		},
	}, func(ctx context.Context, shape string, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx
		switch {
		case shape == operationType && id == operationID1:
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"hydra:pg_property": map[string]any{"id": iriTemplateID, typeKey: iriTemplateType},
			}}, nil
		case shape == iriTemplateType && id == iriTemplateID:
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"template": "https://example.org/query{?station}",
				"mapping":  []any{map[string]any{"id": mappingID1, typeKey: iriTemplateMappingType}},
			}}, nil
		case shape == iriTemplateMappingType && id == mappingID1:
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				variableKey: "station",
				"label":     "Station code",
				rangeKey:    xsdStringRange,
				requiredKey: "not-a-bool",
			}}, nil
		default:
			return nil, fmt.Errorf("unexpected fetch %s %s", shape, id)
		}
	})
	if err == nil {
		t.Fatal("mapDistribution() error = nil")
	}
}
