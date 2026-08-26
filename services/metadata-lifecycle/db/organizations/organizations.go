// Package organizations provides PostgreSQL persistence for organizations.
package organizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dbtx"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/organizations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO: optimize this package and the sql queries, both for performance and for consistency in implementation

type Repo struct {
	*sqlc.Queries
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{Queries: sqlc.New(db), db: db}
}

func (r *Repo) Create(
	ctx context.Context,
	organization organizations.Organization,
	identifiers []organizations.Identifier,
	parents,
	contactPointIDs []uuid.UUID,
) (organizations.Organization, error) {
	var result organizations.Organization
	err := dbtx.Run(ctx, r.db, r.Queries, func(q *sqlc.Queries) error {
		parentRows, err := q.ListOrganizationsByIDs(ctx, parents)
		if err != nil {
			return fmt.Errorf("check organization parents: %w", err)
		}
		if len(parentRows) != len(parents) {
			return organizations.ErrInvalid
		}

		contactRows, err := q.ListContactPointIDs(ctx, contactPointIDs)
		if err != nil {
			return fmt.Errorf("check contact points: %w", err)
		}
		if len(contactRows) != len(contactPointIDs) {
			return organizations.ErrInvalid
		}

		_, err = q.CreateOrganization(ctx, sqlc.CreateOrganizationParams{
			ID:         organization.ID,
			Name:       organization.Name,
			Street:     organization.Street,
			Locality:   organization.Locality,
			PostalCode: organization.PostalCode,
			Country:    organization.Country,
			LogoUrl:    organization.LogoURL,
			PageUrl:    organization.PageURL,
			Email:      organization.Email,
			LeiCode:    organization.LEICode,
		})
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return organizations.ErrConflict
			}
			return fmt.Errorf("insert organization: %w", err)
		}

		for _, identifier := range identifiers {
			err := q.CreateOrganizationIdentifier(ctx, sqlc.CreateOrganizationIdentifierParams{
				ID:             identifier.ID,
				Type:           identifier.Type,
				Value:          identifier.Value,
				OrganizationID: &organization.ID,
			})
			if err != nil {
				if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
					return organizations.ErrConflict
				}

				return fmt.Errorf("insert organization identifier: %w", err)
			}
		}

		for _, parentID := range parents {
			if err := q.CreateOrganizationParent(ctx, sqlc.CreateOrganizationParentParams{
				ChildOrganizationID:  organization.ID,
				ParentOrganizationID: parentID,
			}); err != nil {
				return fmt.Errorf("link organization parent: %w", err)
			}
		}

		for _, contactPointID := range contactPointIDs {
			if err := q.CreateOrganizationContactPoint(ctx, sqlc.CreateOrganizationContactPointParams{
				OrganizationID: organization.ID,
				ContactPointID: contactPointID,
			}); err != nil {
				return fmt.Errorf("link organization contact point: %w", err)
			}
		}

		row, err := q.GetOrganization(ctx, organization.ID)
		if err != nil {
			return fmt.Errorf("reload organization: %w", err)
		}

		loaded, err := r.loadWithQueries(ctx, q, []sqlc.Organization{row})
		if err != nil {
			return err
		}

		result = loaded[0]

		return nil
	})
	if err != nil {
		return organizations.Organization{}, fmt.Errorf("create organization transaction: %w", err)
	}

	return result, nil
}

func (r *Repo) Replace(
	ctx context.Context,
	id uuid.UUID,
	request organizations.ReplaceRequest,
	identifiers []organizations.Identifier,
	parents,
	contactPointIDs []uuid.UUID,
) (organizations.Organization, error) {
	var result organizations.Organization
	err := dbtx.Run(ctx, r.db, r.Queries, func(q *sqlc.Queries) error {
		_, err := q.GetOrganizationForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return organizations.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("lock organization: %w", err)
		}

		parentRows, err := q.ListOrganizationsByIDs(ctx, parents)
		if err != nil {
			return fmt.Errorf("check organization parents: %w", err)
		}
		if len(parentRows) != len(parents) {
			return organizations.ErrInvalid
		}

		contactRows, err := q.ListContactPointIDs(ctx, contactPointIDs)
		if err != nil {
			return fmt.Errorf("check contact points: %w", err)
		}
		if len(contactRows) != len(contactPointIDs) {
			return organizations.ErrInvalid
		}

		_, err = q.ReplaceOrganization(ctx, sqlc.ReplaceOrganizationParams{
			ID:         id,
			Name:       request.Name,
			Street:     request.Street,
			Locality:   request.Locality,
			PostalCode: request.PostalCode,
			Country:    request.Country,
			LogoUrl:    request.LogoURL,
			PageUrl:    request.PageURL,
			Email:      request.Email,
			LeiCode:    request.LEICode,
		})
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return organizations.ErrConflict
			}

			return fmt.Errorf("update organization: %w", err)
		}

		if err := q.DeleteOrganizationIdentifiers(ctx, &id); err != nil {
			return fmt.Errorf("replace organization identifiers: %w", err)
		}

		if err := q.DeleteOrganizationParents(ctx, id); err != nil {
			return fmt.Errorf("replace organization parents: %w", err)
		}

		if err := q.DeleteOrganizationContactPoints(ctx, id); err != nil {
			return fmt.Errorf("replace organization contact points: %w", err)
		}

		if err := insertOrganizationChildren(ctx, q, id, identifiers, parents, contactPointIDs); err != nil {
			return err
		}

		row, err := q.GetOrganization(ctx, id)
		if err != nil {
			return fmt.Errorf("reload organization: %w", err)
		}

		loaded, err := r.loadWithQueries(ctx, q, []sqlc.Organization{row})
		if err != nil {
			return err
		}

		result = loaded[0]

		return nil
	})
	if err != nil {
		return organizations.Organization{}, fmt.Errorf("replace organization transaction: %w", err)
	}

	return result, nil
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := r.DeleteOrganization(ctx, id)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}
	if deleted == 0 {
		return organizations.ErrNotFound
	}

	return nil
}

func insertOrganizationChildren(ctx context.Context, q *sqlc.Queries, id uuid.UUID, identifiers []organizations.Identifier, parents, contactPointIDs []uuid.UUID) error {
	for _, identifier := range identifiers {
		err := q.CreateOrganizationIdentifier(ctx, sqlc.CreateOrganizationIdentifierParams{
			ID:             identifier.ID,
			Type:           identifier.Type,
			Value:          identifier.Value,
			OrganizationID: &id,
		})
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return organizations.ErrConflict
			}

			return fmt.Errorf("insert organization identifier: %w", err)
		}
	}

	for _, parentID := range parents {
		err := q.CreateOrganizationParent(ctx, sqlc.CreateOrganizationParentParams{
			ChildOrganizationID:  id,
			ParentOrganizationID: parentID,
		})
		if err != nil {
			return fmt.Errorf("link organization parent: %w", err)
		}
	}

	for _, contactPointID := range contactPointIDs {
		err := q.CreateOrganizationContactPoint(ctx, sqlc.CreateOrganizationContactPointParams{
			OrganizationID: id,
			ContactPointID: contactPointID,
		})
		if err != nil {
			return fmt.Errorf("link organization contact point: %w", err)
		}
	}

	return nil
}

func (r *Repo) List(ctx context.Context) ([]organizations.Organization, error) {
	rows, err := r.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return r.loadWithQueries(ctx, r.Queries, rows)
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (organizations.Organization, error) {
	row, err := r.GetOrganization(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return organizations.Organization{}, organizations.ErrNotFound
	}
	if err != nil {
		return organizations.Organization{}, fmt.Errorf("get organization: %w", err)
	}

	result, err := r.loadWithQueries(ctx, r.Queries, []sqlc.Organization{row})
	if err != nil {
		return organizations.Organization{}, err
	}

	return result[0], nil
}

// doing this to avoid too many queries (ogranization*identifiers*contactpoints)
func (r *Repo) loadWithQueries(ctx context.Context, q *sqlc.Queries, rows []sqlc.Organization) ([]organizations.Organization, error) {
	result := make([]organizations.Organization, 0, len(rows))
	if len(rows) == 0 {
		return result, nil
	}

	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
		result = append(result, mapOrganization(row))
	}

	identifiers, err := q.ListOrganizationIdentifiers(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list organization identifiers: %w", err)
	}
	parents, err := q.ListOrganizationParents(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list organization parents: %w", err)
	}
	contactPoints, err := q.ListOrganizationContactPointIDs(ctx, ids)
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
			ID:    item.Identifier.ID,
			Type:  item.Identifier.Type,
			Value: item.Identifier.Value,
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
