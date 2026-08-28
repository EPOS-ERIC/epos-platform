// Package organizations provides PostgreSQL persistence for organizations.
package organizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dbtx"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
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
	input domain.OrganizationInput,
) (domain.Organization, error) {
	var result domain.Organization
	err := dbtx.Run(ctx, r.db, r.Queries, func(q *sqlc.Queries) error {
		parentRows, err := q.ListOrganizationsByIDs(ctx, input.Parents)
		if err != nil {
			return fmt.Errorf("check organization parents: %w", err)
		}
		if len(parentRows) != len(input.Parents) {
			return domain.ErrOrganizationNotFound
		}

		contactRows, err := q.ListContactPointIDs(ctx, input.ContactPoints)
		if err != nil {
			return fmt.Errorf("check contact points: %w", err)
		}
		if len(contactRows) != len(input.ContactPoints) {
			return domain.ErrOrganizationNotFound
		}

		organizationID := uuid.New()
		_, err = q.CreateOrganization(ctx, toCreateOrganizationParams(organizationID, input))
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return domain.ErrOrganizationConflict
			}
			return fmt.Errorf("insert organization: %w", err)
		}

		for _, identifier := range input.Identifiers {
			identifierID := uuid.New()
			err := q.CreateOrganizationIdentifier(ctx, sqlc.CreateOrganizationIdentifierParams{
				ID:             identifierID,
				Type:           identifier.Type,
				Value:          identifier.Value,
				OrganizationID: &organizationID,
			})
			if err != nil {
				if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
					return domain.ErrOrganizationConflict
				}

				return fmt.Errorf("insert organization identifier: %w", err)
			}
		}

		for _, parentID := range input.Parents {
			if err := q.CreateOrganizationParent(ctx, sqlc.CreateOrganizationParentParams{
				ChildOrganizationID:  organizationID,
				ParentOrganizationID: parentID,
			}); err != nil {
				return fmt.Errorf("link organization parent: %w", err)
			}
		}

		for _, contactPointID := range input.ContactPoints {
			if err := q.CreateOrganizationContactPoint(ctx, sqlc.CreateOrganizationContactPointParams{
				OrganizationID: organizationID,
				ContactPointID: contactPointID,
			}); err != nil {
				return fmt.Errorf("link organization contact point: %w", err)
			}
		}

		row, err := q.GetOrganization(ctx, organizationID)
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
		return domain.Organization{}, fmt.Errorf("create organization transaction: %w", err)
	}

	return result, nil
}

func (r *Repo) Replace(
	ctx context.Context,
	id uuid.UUID,
	input domain.OrganizationInput,
) (domain.Organization, error) {
	var result domain.Organization
	err := dbtx.Run(ctx, r.db, r.Queries, func(q *sqlc.Queries) error {
		_, err := q.GetOrganizationForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrOrganizationNotFound
		}
		if err != nil {
			return fmt.Errorf("lock organization: %w", err)
		}

		parentRows, err := q.ListOrganizationsByIDs(ctx, input.Parents)
		if err != nil {
			return fmt.Errorf("check organization parents: %w", err)
		}
		if len(parentRows) != len(input.Parents) {
			return domain.ErrOrganizationNotFound
		}

		contactRows, err := q.ListContactPointIDs(ctx, input.ContactPoints)
		if err != nil {
			return fmt.Errorf("check contact points: %w", err)
		}
		if len(contactRows) != len(input.ContactPoints) {
			return domain.ErrOrganizationNotFound
		}

		_, err = q.ReplaceOrganization(ctx, toReplaceOrganizationParams(id, input))
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return domain.ErrOrganizationConflict
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

		if err := insertOrganizationChildren(ctx, q, id, input); err != nil {
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
		return domain.Organization{}, fmt.Errorf("replace organization transaction: %w", err)
	}

	return result, nil
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := r.DeleteOrganization(ctx, id)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}
	if deleted == 0 {
		return domain.ErrOrganizationNotFound
	}

	return nil
}

func insertOrganizationChildren(ctx context.Context, q *sqlc.Queries, id uuid.UUID, input domain.OrganizationInput) error {
	for _, identifier := range input.Identifiers {
		identifierID := uuid.New()
		err := q.CreateOrganizationIdentifier(ctx, sqlc.CreateOrganizationIdentifierParams{
			ID:             identifierID,
			Type:           identifier.Type,
			Value:          identifier.Value,
			OrganizationID: &id,
		})
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return domain.ErrOrganizationConflict
			}

			return fmt.Errorf("insert organization identifier: %w", err)
		}
	}

	for _, parentID := range input.Parents {
		err := q.CreateOrganizationParent(ctx, sqlc.CreateOrganizationParentParams{
			ChildOrganizationID:  id,
			ParentOrganizationID: parentID,
		})
		if err != nil {
			return fmt.Errorf("link organization parent: %w", err)
		}
	}

	for _, contactPointID := range input.ContactPoints {
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

func (r *Repo) List(ctx context.Context) ([]domain.Organization, error) {
	rows, err := r.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return r.loadWithQueries(ctx, r.Queries, rows)
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	row, err := r.GetOrganization(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Organization{}, domain.ErrOrganizationNotFound
	}
	if err != nil {
		return domain.Organization{}, fmt.Errorf("get organization: %w", err)
	}

	result, err := r.loadWithQueries(ctx, r.Queries, []sqlc.Organization{row})
	if err != nil {
		return domain.Organization{}, err
	}

	return result[0], nil
}

// doing this to avoid too many queries
func (r *Repo) loadWithQueries(ctx context.Context, q *sqlc.Queries, rows []sqlc.Organization) ([]domain.Organization, error) {
	if len(rows) == 0 {
		return []domain.Organization{}, nil
	}

	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
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

	identifiersByOrganization := make(map[uuid.UUID][]domain.OrganizationIdentifier, len(rows))
	parentsByOrganization := make(map[uuid.UUID][]uuid.UUID, len(rows))
	contactsByOrganization := make(map[uuid.UUID][]uuid.UUID, len(rows))
	for _, item := range identifiers {
		if item.OrganizationID == nil {
			continue
		}
		identifiersByOrganization[*item.OrganizationID] = append(identifiersByOrganization[*item.OrganizationID], toDomainOrganizationIdentifier(item))
	}
	for _, item := range parents {
		parentsByOrganization[item.ChildOrganizationID] = append(parentsByOrganization[item.ChildOrganizationID], item.ParentOrganizationID)
	}
	for _, item := range contactPoints {
		contactsByOrganization[item.OrganizationID] = append(contactsByOrganization[item.OrganizationID], item.ContactPointID)
	}

	result := make([]domain.Organization, 0, len(rows))
	for _, row := range rows {
		organization, err := toDomainOrganization(
			row,
			identifiersByOrganization[row.ID],
			parentsByOrganization[row.ID],
			contactsByOrganization[row.ID],
		)
		if err != nil {
			return nil, fmt.Errorf("construct organization %s: %w", row.ID, err)
		}
		result = append(result, organization)
	}

	return result, nil
}
