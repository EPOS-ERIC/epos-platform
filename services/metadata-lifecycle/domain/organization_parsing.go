package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
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
	if value.Name == "" {
		return OrganizationInput{}, invalidField("name", "Name must not be blank.", nil)
	}
	if err := validateUUIDListField(value.Parents, "parents", "Parent organization"); err != nil {
		return OrganizationInput{}, err
	}
	for i := range value.Identifiers {
		identifierInput, err := NewOrganizationIdentifierInput(value.Identifiers[i])
		if err != nil {
			return OrganizationInput{}, fmt.Errorf("validate organization identifier: %w", apperror.PrefixTarget(err, fmt.Sprintf("identifiers[%d]", i)))
		}
		value.Identifiers[i] = identifierInput
	}
	return value, nil
}

// NewOrganization creates a complete canonical organization from its identity and input.
func NewOrganization(id uuid.UUID, value OrganizationInput, identifiers []OrganizationIdentifier) (result Organization, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if id == uuid.Nil {
		return Organization{}, apperror.Default(apperror.CodeValidationFailed)
	}
	value, err = NewOrganizationInput(value)
	if err != nil {
		return Organization{}, err
	}
	for i, identifier := range identifiers {
		identifiers[i], err = NewOrganizationIdentifier(identifiers[i].ID, OrganizationIdentifierInput{
			Type:  identifier.Type,
			Value: identifier.Value,
		})
		if err != nil {
			return Organization{}, apperror.Default(apperror.CodeValidationFailed)
		}
	}
	return Organization{
		ID:          id,
		Name:        value.Name,
		Street:      value.Street,
		Locality:    value.Locality,
		PostalCode:  value.PostalCode,
		Country:     value.Country,
		LogoURL:     value.LogoURL,
		PageURL:     value.PageURL,
		Email:       value.Email,
		Identifiers: identifiers,
		Parents:     slices.Clone(value.Parents),
	}, nil
}

// NewOrganizationIdentifierInput validates caller-supplied identifier fields.
func NewOrganizationIdentifierInput(value OrganizationIdentifierInput) (OrganizationIdentifierInput, error) {
	value.Type = strings.TrimSpace(value.Type)
	value.Value = strings.TrimSpace(value.Value)
	if value.Type == "" || value.Value == "" {
		if value.Type == "" {
			return OrganizationIdentifierInput{}, invalidField("type", "Type must not be blank.", nil)
		}
		return OrganizationIdentifierInput{}, invalidField("value", "Value must not be blank.", nil)
	}
	return value, nil
}

// NewOrganizationIdentifier creates a persisted identifier from its identity and input.
func NewOrganizationIdentifier(id uuid.UUID, value OrganizationIdentifierInput) (result OrganizationIdentifier, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if id == uuid.Nil {
		return OrganizationIdentifier{}, apperror.Default(apperror.CodeValidationFailed)
	}
	value, err = NewOrganizationIdentifierInput(value)
	if err != nil {
		return OrganizationIdentifier{}, err
	}
	return OrganizationIdentifier{ID: id, Type: value.Type, Value: value.Value}, nil
}
