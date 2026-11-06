package domain

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

var keywordPattern = regexp.MustCompile(`^[A-Za-z0-9 _:-]+$`)

func normalizeKeywords(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || !keywordPattern.MatchString(value) {
			return nil, fmt.Errorf("keyword %q is invalid", value)
		}
		result = append(result, value)
	}
	return result, nil
}

func NewIdentifierInput(value IdentifierInput) (IdentifierInput, error) {
	value.Type = IdentifierType(strings.TrimSpace(string(value.Type)))
	value.Value = strings.TrimSpace(value.Value)
	if value.Type == "" {
		return IdentifierInput{}, invalidField("type", "Type must not be blank.", nil)
	}
	if value.Value == "" {
		return IdentifierInput{}, invalidField("value", "Value must not be blank.", nil)
	}
	return value, nil
}

func validateIdentifier(value Identifier, requireID bool) error {
	if requireID && value.ID == uuid.Nil {
		return fmt.Errorf("identifier ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if strings.TrimSpace(string(value.Type)) == "" || strings.TrimSpace(value.Value) == "" {
		return fmt.Errorf("identifier type and value are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

func validateContactPoints(values []ContactPoint, label string) error {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if value.ID == uuid.Nil || strings.TrimSpace(value.Email) == "" {
			return fmt.Errorf("%s requires ID and email: %w", label, apperror.Default(apperror.CodeValidationFailed))
		}
		if _, exists := seen[value.ID]; exists {
			return fmt.Errorf("duplicate %s %q: %w", label, value.ID, apperror.Default(apperror.CodeValidationFailed))
		}
		seen[value.ID] = struct{}{}
	}
	return nil
}

func validateOrganizations(values []OrganizationMetadata, relation string) error {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if err := validateOrganization(value); err != nil {
			return fmt.Errorf("resource %s: %w", relation, err)
		}
		if _, exists := seen[value.ID]; exists {
			return fmt.Errorf("duplicate resource %s %q: %w", relation, value.ID, apperror.Default(apperror.CodeValidationFailed))
		}
		seen[value.ID] = struct{}{}
	}
	return nil
}

func validateOrganization(value OrganizationMetadata) error {
	if value.ID == uuid.Nil || strings.TrimSpace(value.Name) == "" {
		return fmt.Errorf("organization requires ID and name: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	seen := make(map[uuid.UUID]struct{}, len(value.Identifiers))
	for _, identifier := range value.Identifiers {
		if err := validateIdentifier(identifier, true); err != nil {
			return fmt.Errorf("organization identifier: %w", err)
		}
		if _, exists := seen[identifier.ID]; exists {
			return fmt.Errorf("duplicate organization identifier %q: %w", identifier.ID, apperror.Default(apperror.CodeValidationFailed))
		}
		seen[identifier.ID] = struct{}{}
	}
	return validateContactPoints(value.ContactPoints, "organization contact point")
}

// Validate validates organization metadata embedded in an import or resource.
func (value OrganizationMetadata) Validate() error {
	return validateOrganization(value)
}

func validateCategories(values []CategoryMetadata, relation string) error {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if err := validateCategory(value); err != nil {
			return fmt.Errorf("resource %s: %w", relation, err)
		}
		if _, exists := seen[value.ID]; exists {
			return fmt.Errorf("duplicate resource %s %q: %w", relation, value.ID, apperror.Default(apperror.CodeValidationFailed))
		}
		seen[value.ID] = struct{}{}
	}
	return nil
}

func validateCategory(value CategoryMetadata) error {
	if value.ID == uuid.Nil || strings.TrimSpace(value.Name) == "" {
		return fmt.Errorf("category requires ID and name: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value.Scheme.ID != uuid.Nil || value.Scheme.Name != "" || value.Scheme.Concept != "" {
		if _, err := NewGroup(value.Scheme.ID, GroupInput{
			Name:        value.Scheme.Name,
			Description: value.Scheme.Description,
			Label:       value.Scheme.Label,
			Logo:        value.Scheme.Logo,
			Page:        value.Scheme.Page,
			Color:       value.Scheme.Color,
			Position:    value.Scheme.Position,
			Concept:     value.Scheme.Concept,
		}); err != nil {
			return fmt.Errorf("category scheme: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
	}
	for _, child := range value.Children {
		if err := validateCategory(child); err != nil {
			return err
		}
	}
	return nil
}

// Validate validates category metadata embedded in an import or resource.
func (value CategoryMetadata) Validate() error {
	return validateCategory(value)
}
