package resources

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapConcept(t *testing.T) {
	t.Parallel()

	concept, err := mapConcept(loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/concept/software-jup-conc-multi.json"))
	if err != nil {
		t.Fatalf("mapConcept() error = %v", err)
	}

	if concept.ID != "category:software_jup_conc_multi/v1.0" {
		t.Fatalf("concept.ID = %q", concept.ID)
	}
	if concept.Name != "Multidisciplinary software" {
		t.Fatalf("concept.Name = %q", concept.Name)
	}
	if concept.Description != "Multidisciplinary software" {
		t.Fatalf("concept.Description = %q", concept.Description)
	}
	if concept.InScheme == nil || concept.InScheme.Href != "v1/resources/conceptscheme/category:software_jup/v1.0" {
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
