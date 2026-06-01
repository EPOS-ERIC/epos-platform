package resources

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapConceptScheme(t *testing.T) {
	t.Parallel()

	conceptScheme, err := mapConceptScheme(loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/conceptscheme/geomagneticobservations.json"))
	if err != nil {
		t.Fatalf("mapConceptScheme() error = %v", err)
	}

	if conceptScheme.ID != "category:geomagneticobservatis/v1.0" {
		t.Fatalf("conceptScheme.ID = %q", conceptScheme.ID)
	}
	if conceptScheme.Label != "GEO" {
		t.Fatalf("conceptScheme.Label = %q", conceptScheme.Label)
	}
	if conceptScheme.Name != "Geomagnetic Observations" {
		t.Fatalf("conceptScheme.Name = %q", conceptScheme.Name)
	}
	if conceptScheme.Description != "TCS Domain" {
		t.Fatalf("conceptScheme.Description = %q", conceptScheme.Description)
	}
	if conceptScheme.Logo != "assets/img/logo/GEOMAG_logo.png" {
		t.Fatalf("conceptScheme.Logo = %q", conceptScheme.Logo)
	}
	if conceptScheme.Homepage != "https://www.epos-eu.org/tcs/geomagnetic-observations" {
		t.Fatalf("conceptScheme.Homepage = %q", conceptScheme.Homepage)
	}
	if conceptScheme.Color != "#2e3486" {
		t.Fatalf("conceptScheme.Color = %q", conceptScheme.Color)
	}
	if conceptScheme.OrderNumber != 6 {
		t.Fatalf("conceptScheme.OrderNumber = %d", conceptScheme.OrderNumber)
	}
	if len(conceptScheme.TopConcepts) != 1 || conceptScheme.TopConcepts[0].Href != "v1/concept/category:facets/dataset-theme/v1.0" {
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
