package organizations

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toDomainOrganizationInput(value organizationRequest) (domain.OrganizationInput, error) {
	input, err := domain.NewOrganizationInput(domain.OrganizationInput{
		Name:          value.Name,
		Street:        value.Address.Street,
		Locality:      value.Address.Locality,
		PostalCode:    value.Address.PostalCode,
		Country:       value.Address.Country,
		LogoURL:       value.LogoURL,
		PageURL:       value.PageURL,
		Email:         value.Email,
		LEICode:       value.LEICode,
		Identifiers:   mapSlice(value.Identifiers, toIdentifierInput),
		Parents:       value.Parents,
		ContactPoints: value.ContactPointIDs,
	})
	if err != nil {
		return domain.OrganizationInput{}, fmt.Errorf("map organization request: %w", err)
	}
	return input, nil
}

func mapSlice[In, Out any](values []In, f func(In) Out) []Out {
	if values == nil {
		return nil
	}
	result := make([]Out, 0, len(values))
	for _, value := range values {
		result = append(result, f(value))
	}
	return result
}

func toIdentifierInput(value identifierRequest) domain.OrganizationIdentifierInput {
	return domain.OrganizationIdentifierInput(value)
}

func toResponse(value domain.Organization) organizationResponse {
	identifiers := make([]identifier, 0, len(value.Identifiers))
	for _, item := range value.Identifiers {
		identifiers = append(identifiers, identifier(item))
	}
	// normalize to empty lists when nil
	parents := make([]uuid.UUID, len(value.Parents))
	copy(parents, value.Parents)
	contactPointIDs := make([]uuid.UUID, len(value.ContactPoints))
	copy(contactPointIDs, value.ContactPoints)

	return organizationResponse{
		ID:   value.ID,
		Name: value.Name,
		Address: address{
			Street:     value.Street,
			Locality:   value.Locality,
			PostalCode: value.PostalCode,
			Country:    value.Country,
		},
		LogoURL:               value.LogoURL,
		PageURL:               value.PageURL,
		Email:                 value.Email,
		LEICode:               value.LEICode,
		Identifiers:           identifiers,
		ParentOrganizationIDs: parents,
		ContactPointIDs:       contactPointIDs,
	}
}
