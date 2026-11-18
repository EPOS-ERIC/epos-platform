package organizations

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// toCreateOrganizationParams builds the params for inserting a new organization row.
func toCreateOrganizationParams(id uuid.UUID, input domain.OrganizationInput) sqlc.CreateOrganizationParams {
	return sqlc.CreateOrganizationParams{
		ID:         id,
		Name:       input.Name,
		Street:     input.Street,
		Locality:   input.Locality,
		PostalCode: input.PostalCode,
		Country:    input.Country,
		LogoUrl:    input.LogoURL,
		PageUrl:    input.PageURL,
		Email:      input.Email,
	}
}

// toReplaceOrganizationParams builds the params for a full-replace update of an existing organization row.
func toReplaceOrganizationParams(id uuid.UUID, input domain.OrganizationInput) sqlc.ReplaceOrganizationParams {
	return sqlc.ReplaceOrganizationParams{
		ID:         id,
		Name:       input.Name,
		Street:     input.Street,
		Locality:   input.Locality,
		PostalCode: input.PostalCode,
		Country:    input.Country,
		LogoUrl:    input.LogoURL,
		PageUrl:    input.PageURL,
		Email:      input.Email,
	}
}

// toDomainOrganization assembles a persisted row plus its related rows into a validated domain.Organization.
func toDomainOrganization(
	row sqlc.Organization,
	identifiers []domain.OrganizationIdentifier,
	parents []uuid.UUID,
) (domain.Organization, error) {
	organization := domain.Organization{
		ID:          row.ID,
		Name:        row.Name,
		Street:      row.Street,
		Locality:    row.Locality,
		PostalCode:  row.PostalCode,
		Country:     row.Country,
		LogoURL:     row.LogoUrl,
		PageURL:     row.PageUrl,
		Email:       row.Email,
		Identifiers: identifiers,
		Parents:     parents,
	}
	if err := organization.Normalize(); err != nil {
		return domain.Organization{}, fmt.Errorf("bad organization: %w", err)
	}

	return organization, nil
}

// toDomainOrganizationIdentifier converts a joined identifier row into its domain type.
func toDomainOrganizationIdentifier(item sqlc.ListOrganizationIdentifiersRow) domain.OrganizationIdentifier {
	return domain.OrganizationIdentifier{
		ID:    item.OrganizationIdentifier.ID,
		Type:  item.OrganizationIdentifier.Type,
		Value: item.OrganizationIdentifier.Value,
	}
}
