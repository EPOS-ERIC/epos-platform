package resources

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapConcept(t *testing.T) {
	t.Parallel()

	concept, err := mapConcept(&metadataclient.AccessDetailResponse{
		Id:    "category:paleotsunamidepositscatalogue/v1.0",
		Shape: "concept",
		Properties: map[string]any{
			"definition": "TCS Subdomain",
			"prefLabel":  "Paleotsunami Deposits Catalogue",
			"inScheme": map[string]any{
				"id":   "category:tsunami/v1.0",
				"type": "conceptscheme",
				"href": "http://localhost:8080/api/v2/access/conceptscheme/category%3Atsunami%2Fv1.0",
			},
			"narrower": map[string]any{
				"id":   "category:paleotsunamideposits/v1.0",
				"type": "concept",
				"href": "http://localhost:8080/api/v2/access/concept/category%3Apaleotsunamideposits%2Fv1.0",
			},
		},
	})
	if err != nil {
		t.Fatalf("mapConcept() error = %v", err)
	}

	if concept.ID != "category:paleotsunamidepositscatalogue/v1.0" {
		t.Fatalf("concept.ID = %q", concept.ID)
	}
	if concept.Name != "Paleotsunami Deposits Catalogue" {
		t.Fatalf("concept.Name = %q", concept.Name)
	}
	if concept.Description != "TCS Subdomain" {
		t.Fatalf("concept.Description = %q", concept.Description)
	}
	if concept.InScheme == nil || concept.InScheme.Href != "v1/conceptscheme/category:tsunami/v1.0" {
		t.Fatalf("concept.InScheme = %#v", concept.InScheme)
	}
	if len(concept.Children) != 1 || concept.Children[0].ID != "category:paleotsunamideposits/v1.0" {
		t.Fatalf("concept.Children = %#v", concept.Children)
	}
	if concept.Children[0].Href != "v1/concept/category:paleotsunamideposits/v1.0" {
		t.Fatalf("concept.Children[0].Href = %q", concept.Children[0].Href)
	}
	if concept.Children[0].Name != "" {
		t.Fatalf("concept.Children[0].Name = %q", concept.Children[0].Name)
	}
}

func TestMapConceptMissingName(t *testing.T) {
	t.Parallel()

	_, err := mapConcept(&metadataclient.AccessDetailResponse{Properties: map[string]any{}})
	if err == nil {
		t.Fatal("mapConcept() error = nil")
	}
}
