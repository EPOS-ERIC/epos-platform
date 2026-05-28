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

	detail := &metadataclient.AccessDetailResponse{
		Id:    "org-1",
		Shape: "organization",
		Href:  "/api/v2/access/organization/org-1",
		Properties: map[string]any{
			"legalName": "BGS - British Geological Survey",
			"logo":      "https://example.org/logo.png",
			"url":       "https://example.org",
			"email":     []any{"info@example.org"},
			"telephone": []any{"+44 131 667 1000"},
			"identifier": []any{
				map[string]any{"id": "propertyvalue-1", "type": "propertyvalue"},
			},
			"address":      map[string]any{"id": "postaladdress-1", "type": "postaladdress"},
			"memberOf":     []any{map[string]any{"id": "org-parent", titleKey: "Parent org"}},
			"contactPoint": []any{map[string]any{"id": "cp-1", titleKey: "Contact 1", "href": "http://upstream/contact/1"}},
		},
	}

	organization, err := mapOrganization(context.Background(), detail, func(ctx context.Context, shape string, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx
		switch {
		case shape == "propertyvalue" && id == "propertyvalue-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"propertyID": "viaf",
				"pg_value": []any{
					map[string]any{"id": "http://viaf.org/viaf/136554353/", "type": "organization"},
				},
			}}, nil
		case shape == "postaladdress" && id == "postaladdress-1":
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"streetAddress":   "The Lyell Centre, Research Avenue South",
				"addressLocality": "Edinburgh",
				"postalCode":      "EH14 4AP",
				"addressCountry":  "GB",
			}}, nil
		default:
			return nil, fmt.Errorf("unexpected fetch %s %s", shape, id)
		}
	})
	if err != nil {
		t.Fatalf("mapOrganization() error = %v", err)
	}

	if organization.ID != "org-1" {
		t.Fatalf("organization.ID = %q", organization.ID)
	}
	if organization.Name != "BGS - British Geological Survey" {
		t.Fatalf("organization.Name = %q", organization.Name)
	}
	if organization.Logo != "https://example.org/logo.png" {
		t.Fatalf("organization.Logo = %q", organization.Logo)
	}
	if organization.URL != "https://example.org" {
		t.Fatalf("organization.URL = %q", organization.URL)
	}
	if len(organization.Email) != 1 || organization.Email[0] != "info@example.org" {
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
	if len(organization.MemberOf) != 1 || organization.MemberOf[0].Href != "v1/organization/org-parent" {
		t.Fatalf("organization.MemberOf = %#v", organization.MemberOf)
	}
	if len(organization.ContactPoints) != 1 || organization.ContactPoints[0].Href != "http://upstream/contact/1" {
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
			"legalName": "BGS",
			"address":   map[string]any{"id": "postaladdress-1", "type": "postaladdress"},
		},
	}, func(ctx context.Context, shape string, id string) (*metadataclient.AccessDetailResponse, error) {
		_ = ctx
		if shape == "postaladdress" && id == "postaladdress-1" {
			return &metadataclient.AccessDetailResponse{Properties: map[string]any{
				"addressLocality": "Edinburgh",
				"addressCountry":  "GB",
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
