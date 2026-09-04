package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
	"github.com/google/uuid"
)

func toAPIProtoContactPoint(v *metadatalifecyclev1.ContactPoint) (gatewayapi.ContactPoint, error) {
	if v == nil {
		return gatewayapi.ContactPoint{}, fmt.Errorf("decode metadata-lifecycle contact point: response is empty")
	}
	id, err := parseUUID("contact_point.id", v.Id)
	if err != nil {
		return gatewayapi.ContactPoint{}, err
	}
	return gatewayapi.ContactPoint{ID: id, Email: v.Email, Role: v.Role}, nil
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

func toAPIProtoOrganization(v *metadatalifecyclev1.Organization) (gatewayapi.CanonicalOrganization, error) {
	if v == nil {
		return gatewayapi.CanonicalOrganization{}, fmt.Errorf("decode metadata-lifecycle organization: response is empty")
	}
	id, err := parseUUID("organization.id", v.Id)
	if err != nil {
		return gatewayapi.CanonicalOrganization{}, err
	}
	r := gatewayapi.CanonicalOrganization{ID: id, Name: v.Name}
	setProtoNilString(&r.Address.Street, v.Street)
	setProtoNilString(&r.Address.Locality, v.Locality)
	setProtoNilString(&r.Address.PostalCode, v.PostalCode)
	setProtoNilString(&r.Address.Country, v.Country)
	setProtoOptString(&r.LogoURL, v.LogoUrl)
	setProtoOptString(&r.PageURL, v.PageUrl)
	setProtoOptString(&r.Email, v.Email)
	setProtoOptString(&r.LeiCode, v.LeiCode)
	for _, identifier := range v.Identifiers {
		identifierID, err := parseUUID("organization.identifier.id", identifier.Id)
		if err != nil {
			return gatewayapi.CanonicalOrganization{}, err
		}
		r.Identifiers = append(r.Identifiers, gatewayapi.OrganizationIdentifier{ID: identifierID, Type: identifier.Type, Value: identifier.Value})
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

func setProtoNilString(out *gatewayapi.NilString, value *string) {
	if value == nil {
		out.SetToNull()
		return
	}
	out.SetTo(*value)
}

func setProtoOptString(out *gatewayapi.OptNilString, value *string) {
	if value == nil {
		return
	}
	out.SetTo(*value)
}

func toProtoCreateOrganizationRequest(q *gatewayapi.CreateOrganizationRequest) *metadatalifecyclev1.CreateOrganizationRequest {
	r := &metadatalifecyclev1.CreateOrganizationRequest{Name: string(q.Name), LogoUrl: toProtoOptString(q.LogoURL), PageUrl: toProtoOptString(q.PageURL), Email: toProtoOptString(q.Email), LeiCode: toProtoOptString(q.LeiCode), Parents: uuidStringValues(q.Parents), ContactPointIds: uuidStringValues(q.ContactPointIds)}
	if q.Address.Set {
		r.Street = toProtoNilString(q.Address.Value.Street)
		r.Locality = toProtoNilString(q.Address.Value.Locality)
		r.PostalCode = toProtoNilString(q.Address.Value.PostalCode)
		r.Country = toProtoNilString(q.Address.Value.Country)
	}
	r.Identifiers = protoIdentifierInputs(q.Identifiers)
	return r
}

func toProtoReplaceOrganizationRequest(q *gatewayapi.ReplaceOrganizationRequest, id string) *metadatalifecyclev1.ReplaceOrganizationRequest {
	return &metadatalifecyclev1.ReplaceOrganizationRequest{OrganizationId: id, Name: string(q.Name), Street: toProtoNilString(q.Address.Street), Locality: toProtoNilString(q.Address.Locality), PostalCode: toProtoNilString(q.Address.PostalCode), Country: toProtoNilString(q.Address.Country), LogoUrl: toProtoNilString(q.LogoURL), PageUrl: toProtoNilString(q.PageURL), Email: toProtoNilString(q.Email), LeiCode: toProtoNilString(q.LeiCode), Identifiers: protoIdentifierInputs(q.Identifiers), Parents: uuidStringValues(q.Parents), ContactPointIds: uuidStringValues(q.ContactPointIds)}
}

func toProtoOptString(value gatewayapi.OptNilString) *string {
	return toClientStringPtr(value)
}

func toProtoNilString(value gatewayapi.NilString) *string {
	if value.Null {
		return nil
	}
	return &value.Value
}

func uuidStringValues(values []uuid.UUID) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.String()
	}
	return result
}

func protoIdentifierInputs(values []gatewayapi.OrganizationIdentifierRequest) []*metadatalifecyclev1.OrganizationIdentifierInput {
	result := make([]*metadatalifecyclev1.OrganizationIdentifierInput, len(values))
	for i, value := range values {
		result[i] = &metadatalifecyclev1.OrganizationIdentifierInput{Type: string(value.Type), Value: string(value.Value)}
	}
	return result
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
