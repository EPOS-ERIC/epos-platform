package metadatasync

import "testing"

func TestOrganizationMappingOmitsBlankAddressFields(t *testing.T) {
	node := remoteOrganization{
		NID:   []remoteInt{{Value: 1903}},
		Title: []remoteString{{Value: "Example"}},
		Address: []remoteAddress{{
			Street:   "",
			Locality: " \t",
			Country:  "IT",
		}},
	}

	org, err := node.toDomain()
	if err != nil {
		t.Fatal(err)
	}
	if org.Street != nil || org.Locality != nil || org.PostalCode != nil {
		t.Fatal("blank optional address fields must be absent")
	}
	if org.Country == nil || *org.Country != "IT" {
		t.Fatal("populated country must be preserved")
	}
}
