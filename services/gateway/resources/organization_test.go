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

	detail := loadAccessDetailFixture(t, "testdata/metadata-lifecycle/access/organization/uppsala.json")
	fetch := newFixtureAccessDetailFetcher(t, map[string]string{
		fixtureKey("propertyvalue", "ROR:048a87296/bnode-propertyvalue-14ab692e/v1.0"): "testdata/metadata-lifecycle/access/propertyvalue/uppsala-url.json",
		fixtureKey("postaladdress", "ROR:048a87296/bnode-postaladdress-be765c92/v1.0"): "testdata/metadata-lifecycle/access/postaladdress/uppsala.json",
	})

	organization, err := mapOrganization(context.Background(), detail, fetch)
	if err != nil {
		t.Fatalf("mapOrganization() error = %v", err)
	}

	if organization.ID != "ROR:048a87296/v1.0" {
		t.Fatalf("organization.ID = %q", organization.ID)
	}
	if organization.Name != "Uppsala University" {
		t.Fatalf("organization.Name = %q", organization.Name)
	}
	if organization.Logo != "https://www.uu.se/images/18.17dda5f1791cdbd287d9b55/1622452923523/uu-logo-red.svg" {
		t.Fatalf("organization.Logo = %q", organization.Logo)
	}
	if organization.URL != "http://www.uu.se" {
		t.Fatalf("organization.URL = %q", organization.URL)
	}
	if len(organization.Email) != 0 {
		t.Fatalf("organization.Email = %#v", organization.Email)
	}
	if len(organization.Telephone) != 0 {
		t.Fatalf("organization.Telephone = %#v", organization.Telephone)
	}
	if len(organization.Identifiers) != 1 || organization.Identifiers[0] != (api.Identifier{Type: "URL", Value: "https://ror.org/048a87296"}) {
		t.Fatalf("organization.Identifiers = %#v", organization.Identifiers)
	}
	if organization.Address != "Dag Hammarskjölds väg 7, Uppsala, 751 05, Sweden" {
		t.Fatalf("organization.Address = %q", organization.Address)
	}
	if len(organization.MemberOf) != 0 {
		t.Fatalf("organization.MemberOf = %#v", organization.MemberOf)
	}
	if len(organization.ContactPoints) != 1 || organization.ContactPoints[0].Href != "http://localhost:8080/api/v2/access/contactpoint/https%3A%2F%2Fror.org%2F048a87296%2FlegalContact%2Fv1.0" {
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
