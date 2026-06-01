package resources

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

const (
	titleKey               = "dct:title"
	typeKey                = "type"
	descriptionKey         = "dct:description"
	accessURLKey           = "dcat:accessURL"
	operationID1           = "operation-1"
	operationType          = "operation"
	iriTemplateType        = "iritemplate"
	iriTemplateID          = "iri-template-1"
	iriTemplateMappingType = "iritemplatemapping"
	mappingID1             = "mapping-1"
	variableKey            = "variable"
	stationCodeLabel       = "Station code"
	rangeKey               = "range"
	xsdStringRange         = "xsd:string"
	requiredKey            = "required"
	distributionTitle      = "Distribution title"
	identifierShape        = "identifier"
)

func TestMapDataset(t *testing.T) {
	t.Parallel()

	detail := &metadataclient.AccessDetailResponse{
		Id:    "dataset-1",
		Shape: "dataset",
		Href:  "/api/v2/access/dataset/dataset-1",
		Properties: map[string]any{
			titleKey:                   "Dataset title",
			descriptionKey:             "Dataset description",
			"dct:created":              "2024-01-02T03:04:05Z",
			"dct:modified":             "2024-02-03T04:05:06Z",
			"dct:issued":               "2024-03-04T05:06:07Z",
			"owl:versionInfo":          "v1 info",
			"dcat:keyword":             []any{"a", "b"},
			"dcat:theme":               []any{map[string]any{"id": "concept-1", titleKey: "Concept 1"}},
			"dct:publisher":            []any{map[string]any{"id": "org-1", titleKey: "Org 1"}},
			"dcat:contactPoint":        []any{map[string]any{"id": "cp-1", titleKey: "Contact 1", "href": "http://upstream/contact/1"}},
			"dcat:distribution":        []any{map[string]any{"id": "dist-1", titleKey: "Distribution 1"}},
			"dct:accessRights":         "open",
			"dct:accrualPeriodicity":   "monthly",
			"dct:isVersionOf":          map[string]any{"id": "dataset-parent", titleKey: "Parent"},
			"adms:identifier":          []any{map[string]any{"id": "identifier-1", typeKey: identifierShape}},
			"dct:temporal":             map[string]any{"id": "temporal-1", typeKey: "periodoftime"},
			"dct:spatial":              map[string]any{"id": "location-1", typeKey: "location"},
			"dqv:hasQualityAnnotation": map[string]any{"id": "annotation-1", typeKey: "annotation"},
		},
	}

	dataset, err := mapDataset(context.Background(), detail, func(ctx context.Context, shape string, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx
		switch {
		case shape == identifierShape && id == "identifier-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{"epos:schemeAgency": "doi", "skos:notation": "10.1234/test"}}, nil
		case shape == "periodoftime" && id == "temporal-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{"schema:startDate": "2020-01-01T00:00:00Z", "schema:endDate": "2020-12-31T00:00:00Z"}}, nil
		case shape == "location" && id == "location-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{"locn:geometry": "POINT(1 2)"}}, nil
		case shape == "annotation" && id == "annotation-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{"oa:hasBody": "QA body"}}, nil
		default:
			return nil, fmt.Errorf("unexpected fetch %s %s", shape, id)
		}
	})
	if err != nil {
		t.Fatalf("mapDataset() error = %v", err)
	}

	if dataset.ID != "dataset-1" {
		t.Fatalf("dataset.ID = %q", dataset.ID)
	}
	if dataset.Name != "Dataset title" {
		t.Fatalf("dataset.Name = %q", dataset.Name)
	}
	if dataset.Description != "Dataset description" {
		t.Fatalf("dataset.Description = %q", dataset.Description)
	}
	if dataset.VersionInfo != "v1 info" {
		t.Fatalf("dataset.VersionInfo = %q", dataset.VersionInfo)
	}
	if dataset.AccessRights != "open" {
		t.Fatalf("dataset.AccessRights = %q", dataset.AccessRights)
	}
	if dataset.AccrualPeriodicity != "monthly" {
		t.Fatalf("dataset.AccrualPeriodicity = %q", dataset.AccrualPeriodicity)
	}
	if dataset.QualityAnnotation != "QA body" {
		t.Fatalf("dataset.QualityAnnotation = %q", dataset.QualityAnnotation)
	}
	if dataset.Spatial != "POINT(1 2)" {
		t.Fatalf("dataset.Spatial = %q", dataset.Spatial)
	}
	if len(dataset.Identifiers) != 1 || dataset.Identifiers[0] != (api.Identifier{Type: "doi", Value: "10.1234/test"}) {
		t.Fatalf("dataset.Identifiers = %#v", dataset.Identifiers)
	}
	if dataset.Temporal == nil {
		t.Fatal("dataset.Temporal = nil")
	}
	assertTimeEqual(t, dataset.Created, "2024-01-02T03:04:05Z")
	assertTimeEqual(t, dataset.Modified, "2024-02-03T04:05:06Z")
	assertTimeEqual(t, dataset.Issued, "2024-03-04T05:06:07Z")
	assertTimeEqual(t, dataset.Temporal.Start, "2020-01-01T00:00:00Z")
	assertTimeEqual(t, dataset.Temporal.End, "2020-12-31T00:00:00Z")
	if len(dataset.Concepts) != 1 || dataset.Concepts[0].Href != "v1/concept/concept-1" {
		t.Fatalf("dataset.Concepts = %#v", dataset.Concepts)
	}
	if len(dataset.Publisher) != 1 || dataset.Publisher[0].Href != "v1/organization/org-1" {
		t.Fatalf("dataset.Publisher = %#v", dataset.Publisher)
	}
	if len(dataset.Distributions) != 1 || dataset.Distributions[0].Href != "v1/distribution/dist-1" {
		t.Fatalf("dataset.Distributions = %#v", dataset.Distributions)
	}
	if len(dataset.ContactPoints) != 1 || dataset.ContactPoints[0].Href != "http://upstream/contact/1" {
		t.Fatalf("dataset.ContactPoints = %#v", dataset.ContactPoints)
	}
	if dataset.IsVersionOf == nil || dataset.IsVersionOf.Href != "v1/dataset/dataset-parent" {
		t.Fatalf("dataset.IsVersionOf = %#v", dataset.IsVersionOf)
	}
}

func assertTimeEqual(t *testing.T, got *time.Time, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("time is nil, want %s", want)
	}

	parsedWant, err := time.Parse(time.RFC3339, want)
	if err != nil {
		t.Fatalf("parse want time: %v", err)
	}

	if !got.Equal(parsedWant) {
		t.Fatalf("time = %s, want %s", got.Format(time.RFC3339), parsedWant.Format(time.RFC3339))
	}
}
