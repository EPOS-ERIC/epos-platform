package handlers

import (
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func toAPIContactPoint(v metadatalifecycleclient.ContactPoint) (gatewayapi.ContactPoint, error) {
	id, err := parseUUID("contact_point.id", v.Id)
	if err != nil {
		return gatewayapi.ContactPoint{}, err
	}
	return gatewayapi.ContactPoint{
		ID:    id,
		Email: v.Email,
		Role:  v.Role,
	}, nil
}

func toAPIOrganization(v metadatalifecycleclient.CanonicalOrganization) (gatewayapi.CanonicalOrganization, error) {
	id, err := parseUUID("organization.id", v.Id)
	if err != nil {
		return gatewayapi.CanonicalOrganization{}, err
	}
	r := gatewayapi.CanonicalOrganization{
		Name:    v.Name,
		ID:      id,
		Address: gatewayapi.OrganizationAddress{},
	}
	toAPIOrganizationAddress(&r.Address, v.Address)
	setAPINilString(&r.LogoURL, v.LogoUrl)
	setAPINilString(&r.PageURL, v.PageUrl)
	setAPINilString(&r.Email, v.Email)
	setAPINilString(&r.LeiCode, v.LeiCode)
	for _, identifier := range v.Identifiers {
		identifierID, err := parseUUID("organization.identifier.id", identifier.Id)
		if err != nil {
			return gatewayapi.CanonicalOrganization{}, err
		}
		r.Identifiers = append(r.Identifiers, gatewayapi.OrganizationIdentifier{
			ID:    identifierID,
			Type:  identifier.Type,
			Value: identifier.Value,
		})
	}
	for _, parent := range v.Parents {
		parentID, err := parseUUID("organization.parent_id", parent)
		if err != nil {
			return gatewayapi.CanonicalOrganization{}, err
		}
		r.Parents = append(r.Parents, parentID)
	}
	for _, contactPoint := range v.ContactPointIds {
		contactPointID, err := parseUUID("organization.contact_point_id", contactPoint)
		if err != nil {
			return gatewayapi.CanonicalOrganization{}, err
		}
		r.ContactPointIds = append(r.ContactPointIds, contactPointID)
	}
	return r, nil
}

func toAPIOrganizationAddress(out *gatewayapi.OrganizationAddress, address metadatalifecycleclient.CanonicalOrganizationAddress) {
	if value := address.Street.Get(); value != nil {
		out.Street.SetTo(*value)
	}
	if value := address.Locality.Get(); value != nil {
		out.Locality.SetTo(*value)
	}
	if value := address.PostalCode.Get(); value != nil {
		out.PostalCode.SetTo(*value)
	}
	if value := address.Country.Get(); value != nil {
		out.Country.SetTo(*value)
	}
}

func setAPINilString(out *gatewayapi.OptNilString, value metadatalifecycleclient.NullableString) {
	if value := value.Get(); value != nil {
		out.SetTo(*value)
	}
}
