// Package organizations provides PostgreSQL persistence for organizations.
package organizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/organizations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	*sqlc.Queries
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{Queries: sqlc.New(db)}
}

func (r *Repo) List(ctx context.Context) ([]organizations.Organization, error) {
	rows, err := r.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return r.load(ctx, rows)
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (organizations.Organization, error) {
	row, err := r.GetOrganization(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return organizations.Organization{}, organizations.ErrNotFound
	}
	if err != nil {
		return organizations.Organization{}, fmt.Errorf("get organization: %w", err)
	}

	result, err := r.load(ctx, []sqlc.Organization{row})
	if err != nil {
		return organizations.Organization{}, err
	}

	return result[0], nil
}

// doing this to avoid too many queries (ogranization*identifiers*contactpoints)
func (r *Repo) load(ctx context.Context, rows []sqlc.Organization) ([]organizations.Organization, error) {
	result := make([]organizations.Organization, 0, len(rows))
	if len(rows) == 0 {
		return result, nil
	}

	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
		result = append(result, mapOrganization(row))
	}

	identifiers, err := r.ListOrganizationIdentifiers(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list organization identifiers: %w", err)
	}
	parents, err := r.ListOrganizationParents(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list organization parents: %w", err)
	}
	contactPoints, err := r.ListOrganizationContactPointIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list organization contact points: %w", err)
	}

	byID := make(map[uuid.UUID]int, len(result))
	for i := range result {
		byID[result[i].ID] = i
	}
	for _, item := range identifiers {
		if item.OrganizationID == nil {
			continue
		}
		organizationIndex := byID[*item.OrganizationID]
		result[organizationIndex].Identifiers = append(result[organizationIndex].Identifiers, organizations.Identifier{
			ID:        item.Identifier.ID,
			SourceIRI: item.Identifier.SourceIri,
			Type:      item.Identifier.Type,
			Value:     item.Identifier.Value,
		})
	}
	for _, item := range parents {
		result[byID[item.ChildOrganizationID]].ParentOrganizationIDs = append(result[byID[item.ChildOrganizationID]].ParentOrganizationIDs, item.ParentOrganizationID)
	}
	for _, item := range contactPoints {
		result[byID[item.OrganizationID]].ContactPointIDs = append(result[byID[item.OrganizationID]].ContactPointIDs, item.ContactPointID)
	}

	return result, nil
}

func mapOrganization(row sqlc.Organization) organizations.Organization {
	return organizations.Organization{
		ID:                    row.ID,
		SourceIRI:             row.SourceIri,
		Name:                  row.Name,
		Street:                row.Street,
		Locality:              row.Locality,
		PostalCode:            row.PostalCode,
		Country:               row.Country,
		LogoURL:               row.LogoUrl,
		PageURL:               row.PageUrl,
		Email:                 row.Email,
		LEICode:               row.LeiCode,
		Identifiers:           make([]organizations.Identifier, 0),
		ParentOrganizationIDs: make([]uuid.UUID, 0),
		ContactPointIDs:       make([]uuid.UUID, 0),
	}
}
