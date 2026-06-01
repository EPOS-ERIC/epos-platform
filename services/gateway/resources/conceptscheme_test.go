package resources

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapConceptScheme(t *testing.T) {
	t.Parallel()

	conceptScheme, err := mapConceptScheme(&metadataclient.AccessDetailResponse{
		Id:    "category:tsunami/v1.0",
		Shape: "conceptscheme",
		Properties: map[string]any{
			"type":                   "http://www.w3.org/2004/02/skos/core#ConceptScheme",
			"foaf:logo":              "assets/img/logo/TSU_logo.png",
			"skos:prefLabel":         "TSU",
			"dct:title":              "Tsunami",
			"dct:description":        "TCS Domain",
			"foaf:homepage":          "https://www.epos-eu.org/tcs/tsunami",
			"schema:color":           "#6f9ea8",
			"schema:orderItemNumber": "10",
			"skos:hasTopConcept": []any{
				map[string]any{"id": "category:facets/dataset-theme", "type": "concept"},
				map[string]any{"id": "category:facets/dataset-theme/v1.0", "type": "concept"},
			},
		},
	})
	if err != nil {
		t.Fatalf("mapConceptScheme() error = %v", err)
	}

	if conceptScheme.ID != "category:tsunami/v1.0" {
		t.Fatalf("conceptScheme.ID = %q", conceptScheme.ID)
	}
	if conceptScheme.Label != "TSU" {
		t.Fatalf("conceptScheme.Label = %q", conceptScheme.Label)
	}
	if conceptScheme.Name != "Tsunami" {
		t.Fatalf("conceptScheme.Name = %q", conceptScheme.Name)
	}
	if conceptScheme.Description != "TCS Domain" {
		t.Fatalf("conceptScheme.Description = %q", conceptScheme.Description)
	}
	if conceptScheme.Logo != "assets/img/logo/TSU_logo.png" {
		t.Fatalf("conceptScheme.Logo = %q", conceptScheme.Logo)
	}
	if conceptScheme.Homepage != "https://www.epos-eu.org/tcs/tsunami" {
		t.Fatalf("conceptScheme.Homepage = %q", conceptScheme.Homepage)
	}
	if conceptScheme.Color != "#6f9ea8" {
		t.Fatalf("conceptScheme.Color = %q", conceptScheme.Color)
	}
	if conceptScheme.OrderNumber != 10 {
		t.Fatalf("conceptScheme.OrderNumber = %d", conceptScheme.OrderNumber)
	}
	if len(conceptScheme.TopConcepts) != 2 || conceptScheme.TopConcepts[1].Href != "v1/concept/category:facets/dataset-theme/v1.0" {
		t.Fatalf("conceptScheme.TopConcepts = %#v", conceptScheme.TopConcepts)
	}
}

func TestMapConceptSchemeMissingLabel(t *testing.T) {
	t.Parallel()

	_, err := mapConceptScheme(&metadataclient.AccessDetailResponse{Properties: map[string]any{}})
	if err == nil {
		t.Fatal("mapConceptScheme() error = nil")
	}
}

func TestMapConceptSchemeMissingPrefLabelDoesNotFallBackToTitle(t *testing.T) {
	t.Parallel()

	_, err := mapConceptScheme(&metadataclient.AccessDetailResponse{
		Id:    "category:icsdprocessing/v1.0",
		Shape: "conceptscheme",
		Properties: map[string]any{
			"dct:title": "Processing",
		},
	})
	if err == nil {
		t.Fatal("mapConceptScheme() error = nil")
	}
}
