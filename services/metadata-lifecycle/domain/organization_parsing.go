package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// NewOrganizationInput validates and normalizes editable organization fields.
func NewOrganizationInput(value OrganizationInput) (OrganizationInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	value.Street = optionalTrimmedString(value.Street)
	value.Locality = optionalTrimmedString(value.Locality)
	value.PostalCode = optionalTrimmedString(value.PostalCode)
	value.Country = optionalTrimmedString(value.Country)
	value.LogoURL = optionalTrimmedString(value.LogoURL)
	value.PageURL = optionalTrimmedString(value.PageURL)
	value.Email = optionalTrimmedString(value.Email)
	value.LEICode = optionalTrimmedString(value.LEICode)
	if value.Name == "" {
		return OrganizationInput{}, ErrInvalidOrganization
	}
	if err := validateUUIDList(value.Parents, "parent"); err != nil {
		return OrganizationInput{}, fmt.Errorf("organization parents: %w: %w", err, ErrInvalidOrganization)
	}
	if err := validateUUIDList(value.ContactPoints, "contact point"); err != nil {
		return OrganizationInput{}, fmt.Errorf("organization contact points: %w: %w", err, ErrInvalidOrganization)
	}
	for i := range value.Identifiers {
		identifierInput, err := NewOrganizationIdentifierInput(value.Identifiers[i])
		if err != nil {
			return OrganizationInput{}, ErrInvalidOrganization
		}
		value.Identifiers[i] = identifierInput
	}
	return value, nil
}

// NewOrganization creates a complete canonical organization from its identity and input.
func NewOrganization(id uuid.UUID, value OrganizationInput, identifiers []OrganizationIdentifier) (Organization, error) {
	if id == uuid.Nil {
		return Organization{}, ErrInvalidOrganization
	}
	value, err := NewOrganizationInput(value)
	if err != nil {
		return Organization{}, err
	}
	for i, identifier := range identifiers {
		identifiers[i], err = NewOrganizationIdentifier(identifiers[i].ID, OrganizationIdentifierInput{
			Type:  identifier.Type,
			Value: identifier.Value,
		})
		if err != nil {
			return Organization{}, ErrInvalidOrganization
		}
	}
	return Organization{
		ID:            id,
		Name:          value.Name,
		Street:        value.Street,
		Locality:      value.Locality,
		PostalCode:    value.PostalCode,
		Country:       value.Country,
		LogoURL:       value.LogoURL,
		PageURL:       value.PageURL,
		Email:         value.Email,
		LEICode:       value.LEICode,
		Identifiers:   identifiers,
		Parents:       slices.Clone(value.Parents),
		ContactPoints: slices.Clone(value.ContactPoints),
	}, nil
}

// NewOrganizationReference creates a valid relationship projection.
func NewOrganizationReference(id uuid.UUID, name string) (OrganizationReference, error) {
	if id == uuid.Nil {
		return OrganizationReference{}, ErrInvalidOrganization
	}
	return OrganizationReference{ID: id, Name: name}, nil
}

// NewOrganizationIdentifierInput validates caller-supplied identifier fields.
func NewOrganizationIdentifierInput(value OrganizationIdentifierInput) (OrganizationIdentifierInput, error) {
	value.Type = strings.TrimSpace(value.Type)
	value.Value = strings.TrimSpace(value.Value)
	if value.Type == "" || value.Value == "" {
		return OrganizationIdentifierInput{}, ErrInvalidOrganization
	}
	return value, nil
}

// NewOrganizationIdentifier creates a persisted identifier from its identity and input.
func NewOrganizationIdentifier(id uuid.UUID, value OrganizationIdentifierInput) (OrganizationIdentifier, error) {
	if id == uuid.Nil {
		return OrganizationIdentifier{}, ErrInvalidOrganization
	}
	value, err := NewOrganizationIdentifierInput(value)
	if err != nil {
		return OrganizationIdentifier{}, err
	}
	return OrganizationIdentifier{ID: id, Type: value.Type, Value: value.Value}, nil
}
