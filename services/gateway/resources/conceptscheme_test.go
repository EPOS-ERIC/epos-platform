package resources

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapConceptScheme(t *testing.T) {
	t.Parallel()

	conceptScheme, err := mapConceptScheme(loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/conceptscheme/software-jup.json"))
	if err != nil {
		t.Fatalf("mapConceptScheme() error = %v", err)
	}

	if conceptScheme.ID != "category:software_jup/v1.0" {
		t.Fatalf("conceptScheme.ID = %q", conceptScheme.ID)
	}
	if conceptScheme.Label != "Jupyter Notebooks" {
		t.Fatalf("conceptScheme.Label = %q", conceptScheme.Label)
	}
	if conceptScheme.Name != "Jupyter Notebooks" {
		t.Fatalf("conceptScheme.Name = %q", conceptScheme.Name)
	}
	if conceptScheme.Description != "Jupyter Notebook Domain" {
		t.Fatalf("conceptScheme.Description = %q", conceptScheme.Description)
	}
	if conceptScheme.Logo != "https://raw.githubusercontent.com/epos-eu/platform-gui-icons/3c13924d1f3483a9e1f1941ba094f0937602d0a9/jupyter-svgrepo-com.svg" {
		t.Fatalf("conceptScheme.Logo = %q", conceptScheme.Logo)
	}
	if conceptScheme.Homepage != "" {
		t.Fatalf("conceptScheme.Homepage = %q", conceptScheme.Homepage)
	}
	if conceptScheme.Color != "#F37726" {
		t.Fatalf("conceptScheme.Color = %q", conceptScheme.Color)
	}
	if conceptScheme.OrderNumber != 1 {
		t.Fatalf("conceptScheme.OrderNumber = %d", conceptScheme.OrderNumber)
	}
	if len(conceptScheme.TopConcepts) != 1 || conceptScheme.TopConcepts[0].Href != "v1/concept/category:facets/software-theme/v1.0" {
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
