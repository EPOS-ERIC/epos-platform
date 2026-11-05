// Package resource contains shared SQLC-to-domain mappings for resource metadata.
package resource

import (
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// ToOrganizationMetadata maps a persisted organization to resource metadata.
func ToOrganizationMetadata(value sqlc.Organization) domain.OrganizationMetadata {
	return domain.OrganizationMetadata{
		ID:            value.ID,
		Name:          value.Name,
		Street:        value.Street,
		Locality:      value.Locality,
		PostalCode:    value.PostalCode,
		Country:       value.Country,
		LogoURL:       value.LogoUrl,
		PageURL:       value.PageUrl,
		Email:         value.Email,
		LEICode:       value.LeiCode,
		ContactPoints: nil,
		Identifiers:   nil,
	}
}

// ToContactPoint maps a persisted contact point to resource metadata.
func ToContactPoint(value sqlc.ContactPoint) domain.ContactPoint {
	return domain.ContactPoint{
		ID:    value.ID,
		Email: value.Email,
		Role:  value.Role,
	}
}

// ToIdentifier maps a persisted identifier to resource metadata.
func ToIdentifier(value sqlc.Identifier) domain.Identifier {
	return domain.Identifier{
		ID:         value.ID,
		ResourceID: value.ResourceID,
		Type:       domain.IdentifierType(value.Type),
		Value:      value.Value,
	}
}

// ToCategoryMetadata maps a persisted category and owning group to resource metadata.
func ToCategoryMetadata(id uuid.UUID, name string, group sqlc.Group) domain.CategoryMetadata {
	return domain.CategoryMetadata{
		ID:       id,
		Name:     name,
		Scheme:   ToGroup(group),
		Children: nil,
	}
}

// ToGroup maps a persisted group to resource metadata.
func ToGroup(value sqlc.Group) domain.Group {
	return domain.Group{
		ID:          value.ID,
		Name:        value.Name,
		Description: value.Description,
		Label:       value.Label,
		Logo:        value.Logo,
		Page:        value.Page,
		Color:       value.Color,
		Position:    value.Position,
		Concept:     string(value.Concept),
	}
}
