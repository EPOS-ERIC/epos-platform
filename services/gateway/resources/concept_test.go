package resources

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapConcept(t *testing.T) {
	t.Parallel()

	concept, err := mapConcept(loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/concept/geomagneticdata.json"))
	if err != nil {
		t.Fatalf("mapConcept() error = %v", err)
	}

	if concept.ID != "category:geomagneticdata/v1.0" {
		t.Fatalf("concept.ID = %q", concept.ID)
	}
	if concept.Name != "Geomagnetic data" {
		t.Fatalf("concept.Name = %q", concept.Name)
	}
	if concept.Description != "TCS Subdomain" {
		t.Fatalf("concept.Description = %q", concept.Description)
	}
	if concept.InScheme == nil || concept.InScheme.Href != "v1/conceptscheme/category:geomagneticobservatis/v1.0" {
		t.Fatalf("concept.InScheme = %#v", concept.InScheme)
	}
	if len(concept.Children) != 0 {
		t.Fatalf("concept.Children = %#v", concept.Children)
	}
}

func TestMapConceptMissingName(t *testing.T) {
	t.Parallel()

	_, err := mapConcept(&metadataclient.AccessDetailResponse{Properties: map[string]any{}})
	if err == nil {
		t.Fatal("mapConcept() error = nil")
	}
}
