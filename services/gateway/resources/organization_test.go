package resources

import (
	"context"
	"fmt"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func TestMapOrganization(t *testing.T) {
	t.Parallel()

	detail := loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/organization/bgs.json")
	fetch := newFixtureAccessDetailFetcher(t, map[string]string{
		fixtureKey("propertyvalue", "http://viaf.org/viaf/136554353/bnode-propertyvalue-5bbce760/v1.0"): "testdata/metadata-lifecycle/access/propertyvalue/bgs-viaf.json",
		fixtureKey("postaladdress", "http://viaf.org/viaf/136554353/bnode-postaladdress-064b9034/v1.0"): "testdata/metadata-lifecycle/access/postaladdress/bgs.json",
	})

	organization, err := mapOrganization(context.Background(), detail, fetch)
	if err != nil {
		t.Fatalf("mapOrganization() error = %v", err)
	}

	if organization.ID != "http://viaf.org/viaf/136554353//v1.0" {
		t.Fatalf("organization.ID = %q", organization.ID)
	}
	if organization.Name != "BGS - British Geological Survey, UK Research and Innovation" {
		t.Fatalf("organization.Name = %q", organization.Name)
	}
	if organization.Logo != "https://www.bgs.ac.uk/wp-content/uploads/2020/03/BGS-logo-large.png" {
		t.Fatalf("organization.Logo = %q", organization.Logo)
	}
	if organization.URL != "https://www.bgs.ac.uk/" {
		t.Fatalf("organization.URL = %q", organization.URL)
	}
	if len(organization.Email) != 1 || organization.Email[0] != "enquiries@bgs.ac.uk" {
		t.Fatalf("organization.Email = %#v", organization.Email)
	}
	if len(organization.Telephone) != 1 || organization.Telephone[0] != "+44 131 667 1000" {
		t.Fatalf("organization.Telephone = %#v", organization.Telephone)
	}
	if len(organization.Identifiers) != 1 || organization.Identifiers[0] != (api.Identifier{Type: "viaf", Value: "http://viaf.org/viaf/136554353/"}) {
		t.Fatalf("organization.Identifiers = %#v", organization.Identifiers)
	}
	if organization.Address != "The Lyell Centre, Research Avenue South, Edinburgh, EH14 4AP, GB" {
		t.Fatalf("organization.Address = %q", organization.Address)
	}
	if len(organization.MemberOf) != 1 || organization.MemberOf[0].Href != "v1/organization/http://viaf.org/viaf/148537429//v1.0" {
		t.Fatalf("organization.MemberOf = %#v", organization.MemberOf)
	}
	if len(organization.ContactPoints) != 0 {
		t.Fatalf("organization.ContactPoints = %#v", organization.ContactPoints)
	}
}

func TestMapOrganizationMissingName(t *testing.T) {
	t.Parallel()

	_, err := mapOrganization(context.Background(), &metadataclient.AccessDetailResponse{Properties: map[string]any{}}, nil)
	if err == nil {
		t.Fatal("mapOrganization() error = nil")
	}
}

func TestMapOrganizationPartialAddress(t *testing.T) {
	t.Parallel()

	organization, err := mapOrganization(context.Background(), &metadataclient.AccessDetailResponse{
		Properties: map[string]any{
			"schema:legalName": "BGS",
			"schema:address":   map[string]any{"id": "postaladdress-1", "type": "postaladdress"},
		},
	}, func(ctx context.Context, shape string, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx
		if shape == "postaladdress" && id == "postaladdress-1" {
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"schema:addressLocality": "Edinburgh",
				"schema:addressCountry":  "GB",
			}}, nil
		}

		return nil, fmt.Errorf("unexpected fetch %s %s", shape, id)
	})
	if err != nil {
		t.Fatalf("mapOrganization() error = %v", err)
	}

	if organization.Address != "Edinburgh, GB" {
		t.Fatalf("organization.Address = %q", organization.Address)
	}
}
