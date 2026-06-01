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
	fetch := newFixtureAccessDetailFetcher(t, map[string]string{
		fixtureKey(operationType, "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Operation/v1.0"):                    "testdata/metadata-lifecycle/access/operation/intermagnet.json",
		fixtureKey(iriTemplateType, "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/bnode-iritemplate-2ff35bd4/v1.0"): "testdata/metadata-lifecycle/access/iritemplate/intermagnet.json",
		fixtureKey(iriTemplateMappingType, "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-afe775b2/v1.0"):                                           "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-format.json",
		fixtureKey(iriTemplateMappingType, "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-d1713110/v1.0"):                                           "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-publication-state.json",
		fixtureKey(iriTemplateMappingType, "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-25a4198f/v1.0"):                                           "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-end-date.json",
		fixtureKey(iriTemplateMappingType, "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-73bc40eb/v1.0"):                                           "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-start-date.json",
		fixtureKey(iriTemplateMappingType, "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-e890a680/v1.0"):                                           "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-samples-per-day.json",
		fixtureKey(iriTemplateMappingType, "mailto:ochambers@bgs.ac.uk/bnode-iritemplatemapping-44b3a135/v1.0"):                                           "testdata/metadata-lifecycle/access/iritemplatemapping/intermagnet-observatory-iaga-code.json",
	})

	distribution, err := mapDistribution(context.Background(), detail, fetch)
	if err != nil {
		t.Fatalf("mapDistribution() error = %v", err)
	}

	if distribution.ID != "https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Distribution/v1.0" {
		t.Fatalf("distribution.ID = %q", distribution.ID)
	}
	if distribution.Name != "INTERMAGNET Geomagnetic Observatory Data" {
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
	if distribution.License != "https://creativecommons.org/licenses/by-nc/4.0/" {
		t.Fatalf("distribution.License = %q", distribution.License)
	}
	if distribution.Type != api.DistributionKindWebService {
		t.Fatalf("distribution.Type = %q", distribution.Type)
	}
	assertTimeEqual(t, distribution.Issued, "2018-08-31T00:00:00Z")
	assertTimeEqual(t, distribution.Modified, "2018-08-31T00:00:00Z")
	if distribution.Webservice == nil || distribution.Webservice.Href != "v1/webservice/https://www.epos-eu.org/epos-dcat-ap/GeoElectroMagnetism/WP13-DDSS-001/INTERMAGNET/Webservice/v1.0" {
		t.Fatalf("distribution.Webservice = %#v", distribution.Webservice)
	}
	if distribution.Operation == nil {
		t.Fatal("distribution.Operation = nil")
	}
	if distribution.Operation.Method != "GET" {
		t.Fatalf("distribution.Operation.Method = %q", distribution.Operation.Method)
	}
	if len(distribution.Operation.Returns) != 6 {
		t.Fatalf("distribution.Operation.Returns = %#v", distribution.Operation.Returns)
	}
	if distribution.Operation.IriTemplate == nil {
		t.Fatal("distribution.Operation.IriTemplate = nil")
	}
	if distribution.Operation.IriTemplate.Template == "" {
		t.Fatalf("distribution.Operation.IriTemplate = %#v", distribution.Operation.IriTemplate)
	}
	if len(distribution.Operation.IriTemplate.Mappings) != 6 {
		t.Fatalf("distribution.Operation.IriTemplate.Mappings = %#v", distribution.Operation.IriTemplate.Mappings)
	}
	var foundRequired bool
	for _, mapping := range distribution.Operation.IriTemplate.Mappings {
		if mapping.Variable == "observatoryIagaCode" {
			foundRequired = mapping.Required && mapping.Label == "Observatory IAGA code"
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
