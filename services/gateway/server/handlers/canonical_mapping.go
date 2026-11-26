package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func toDomainCreateOrganization(q *gatewayapi.CreateOrganizationRequest) domain.OrganizationInput {
	v := domain.OrganizationInput{
		Name:    string(q.Name),
		LogoURL: toProtoOptURIString(q.LogoURL),
		PageURL: toProtoOptURIString(q.PageURL),
		Email:   toClientStringPtr(q.Email),
		Parents: uuidStrings(q.Parents),
	}
	if q.Address.Set {
		v.Street = toProtoNilString(q.Address.Value.Street)
		v.Locality = toProtoNilString(q.Address.Value.Locality)
		v.PostalCode = toProtoNilString(q.Address.Value.PostalCode)
		v.Country = toProtoNilString(q.Address.Value.Country)
	}

	for _, id := range q.Identifiers {
		v.Identifiers = append(v.Identifiers, domain.OrganizationIdentifier{
			Type:  string(id.Type),
			Value: string(id.Value),
		})
	}
	return v
}

func toDomainReplaceOrganization(q *gatewayapi.ReplaceOrganizationRequest) domain.OrganizationInput {
	v := domain.OrganizationInput{
		Name:       string(q.Name),
		Street:     toProtoNilString(q.Address.Street),
		Locality:   toProtoNilString(q.Address.Locality),
		PostalCode: toProtoNilString(q.Address.PostalCode),
		Country:    toProtoNilString(q.Address.Country),
		LogoURL:    toProtoNilURIString(q.LogoURL),
		PageURL:    toProtoNilURIString(q.PageURL),
		Email:      toProtoNilString(q.Email),
		Parents:    uuidStrings(q.Parents),
	}

	for _, id := range q.Identifiers {
		v.Identifiers = append(v.Identifiers, domain.OrganizationIdentifier{
			Type:  string(id.Type),
			Value: string(id.Value),
		})
	}
	return v
}

func toAPIProtoContactPoint(v *metadatalifecyclev1.ContactPoint) (gatewayapi.ContactPoint, error) {
	if v == nil {
		return gatewayapi.ContactPoint{}, fmt.Errorf("decode metadata-lifecycle contact point: response is empty")
	}

	id, err := parseUUID("contact_point.id", v.Id)
	if err != nil {
		return gatewayapi.ContactPoint{}, err
	}
	return gatewayapi.ContactPoint{ID: id, Email: v.Email}, nil
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
	r.Address.Street = toAPIRequiredNilString(v.Street)
	r.Address.Locality = toAPIRequiredNilString(v.Locality)
	r.Address.PostalCode = toAPIRequiredNilString(v.PostalCode)
	r.Address.Country = toAPIRequiredNilString(v.Country)

	if err := setProtoOptURI(&r.LogoURL, v.LogoUrl); err != nil {
		return gatewayapi.CanonicalOrganization{}, fmt.Errorf("decode organization.logo_url: %w", err)
	}
	if err := setProtoOptURI(&r.PageURL, v.PageUrl); err != nil {
		return gatewayapi.CanonicalOrganization{}, fmt.Errorf("decode organization.page_url: %w", err)
	}
	r.Email = toAPINilString(v.Email)

	for _, identifier := range v.Identifiers {
		identifierID, err := parseUUID("organization.identifier.id", identifier.Id)
		if err != nil {
			return gatewayapi.CanonicalOrganization{}, err
		}
		r.Identifiers = append(r.Identifiers, gatewayapi.Identifier{
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

	return r, nil
}

// TODO: change these functions to be pure and without side effects
func setProtoOptURI(out *gatewayapi.OptNilURI, value *string) error {
	if value == nil {
		return nil
	}

	parsed, err := parseURI(*value)
	if err != nil {
		return err
	}
	out.SetTo(parsed)
	return nil
}

func toProtoOptURIString(value gatewayapi.OptNilURI) *string {
	if !value.Set || value.Null {
		return nil
	}

	result := value.Value.String()
	return &result
}

func toProtoNilString(value gatewayapi.NilString) *string {
	if value.Null {
		return nil
	}

	return &value.Value
}

func toProtoNilURIString(value gatewayapi.NilURI) *string {
	if value.Null {
		return nil
	}

	result := value.Value.String()
	return &result
}
