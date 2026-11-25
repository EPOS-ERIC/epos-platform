package domain

import (
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// SyncedOrganization contains fields owned by the remote organization source.
type SyncedOrganization struct {
	NID        int64
	ParentNIDs []int64
	Input      OrganizationInput
}

// Organization is a complete canonical organization returned by the domain.
type Organization struct {
	ID          uuid.UUID
	Name        string
	Street      *string
	Locality    *string
	PostalCode  *string
	Country     *string
	LogoURL     *string
	PageURL     *string
	Email       *string
	Identifiers []OrganizationIdentifier
	Parents     []uuid.UUID
}

// OrganizationIdentifier is a persisted organization identifier.
type OrganizationIdentifier struct {
	ID    uuid.UUID
	Type  string
	Value string
}

// OrganizationInput contains fields editable through organization CRUD.
type OrganizationInput struct {
	Name        string
	Street      *string
	Locality    *string
	PostalCode  *string
	Country     *string
	LogoURL     *string
	PageURL     *string
	Email       *string
	Identifiers []OrganizationIdentifierInput
	Parents     []uuid.UUID
}

// OrganizationIdentifierInput is an identifier supplied by a caller.
type OrganizationIdentifierInput struct {
	Type  string
	Value string
}

// Normalize validates and normalizes editable organization fields.
func (value *OrganizationInput) Normalize() error {
	value.Name = strings.TrimSpace(value.Name)
	value.Street = optionalTrimmedString(value.Street)
	value.Locality = optionalTrimmedString(value.Locality)
	value.PostalCode = optionalTrimmedString(value.PostalCode)
	value.Country = optionalTrimmedString(value.Country)
	value.Email = optionalTrimmedString(value.Email)
	if value.Name == "" {
		return apperror.Field("name", "Name must not be blank.", nil)
	}
	if err := normalizeOptionalHTTPURL(value.LogoURL, "logo_url"); err != nil {
		return err
	}
	if err := normalizeOptionalHTTPURL(value.PageURL, "page_url"); err != nil {
		return err
	}
	if err := validateUUIDListField(value.Parents, "parents", "Parent organization"); err != nil {
		return err
	}
	for i := range value.Identifiers {
		if err := value.Identifiers[i].Normalize(); err != nil {
			return fmt.Errorf("validate organization identifier: %w", apperror.PrefixTarget(err, fmt.Sprintf("identifiers[%d]", i)))
		}
	}
	return nil
}

// Normalize validates a persisted organization.
func (value *Organization) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.ID == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}
	input := OrganizationInput{
		Name:       value.Name,
		Street:     value.Street,
		Locality:   value.Locality,
		PostalCode: value.PostalCode,
		Country:    value.Country,
		LogoURL:    value.LogoURL,
		PageURL:    value.PageURL,
		Email:      value.Email,
		Parents:    value.Parents,
	}
	if err := input.Normalize(); err != nil {
		return err
	}
	value.Name = input.Name
	value.Street = input.Street
	value.Locality = input.Locality
	value.PostalCode = input.PostalCode
	value.Country = input.Country
	value.LogoURL = input.LogoURL
	value.PageURL = input.PageURL
	value.Email = input.Email
	for i := range value.Identifiers {
		if err := value.Identifiers[i].Normalize(); err != nil {
			return apperror.Default(apperror.CodeValidationFailed)
		}
	}
	return nil
}

// Normalize validates caller-supplied identifier fields.
func (value *OrganizationIdentifierInput) Normalize() error {
	value.Type = strings.TrimSpace(value.Type)
	value.Value = strings.TrimSpace(value.Value)
	if value.Type == "" {
		return apperror.Field("type", "Type must not be blank.", nil)
	}
	if value.Value == "" {
		return apperror.Field("value", "Value must not be blank.", nil)
	}
	return nil
}

// Normalize validates a persisted organization identifier.
func (value *OrganizationIdentifier) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.ID == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}
	input := OrganizationIdentifierInput{Type: value.Type, Value: value.Value}
	if err := input.Normalize(); err != nil {
		return err
	}
	value.Type = input.Type
	value.Value = input.Value
	return nil
}
