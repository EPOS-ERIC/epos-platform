// Package organizations provides PostgreSQL persistence for organizations.
package organizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
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
	return &Repo{
		Queries: sqlc.New(db),
		db:      db,
	}
}

// Sync atomically replaces the remote-owned fields and hierarchy.
func (r *Repo) Sync(ctx context.Context, values []domain.SyncedOrganization) error {
	return dbtx.RunWithTx(ctx, r.db, r.Queries, func(q *sqlc.Queries, tx pgx.Tx) error {
		// Lock to serialize snapshot writes across gateway replicas.
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('organization-sync'))"); err != nil {
			return fmt.Errorf("lock organization sync: %w", err)
		}

		ids := make(map[int64]uuid.UUID, len(values))
		nids := make([]int64, 0, len(values))
		for _, value := range values {
			input := value.Input
			id, err := q.UpsertSyncedOrganization(ctx, sqlc.UpsertSyncedOrganizationParams{
				RemoteNid:  &value.NID,
				Name:       input.Name,
				Street:     input.Street,
				Locality:   input.Locality,
				PostalCode: input.PostalCode,
				Country:    input.Country,
				LogoUrl:    input.LogoURL,
				PageUrl:    input.PageURL,
				Email:      input.Email,
			})
			if err != nil {
				return fmt.Errorf("upsert organization %d: %w", value.NID, err)
			}

			ids[value.NID] = id
			nids = append(nids, value.NID)
		}

		if err := q.DeleteAbsentSyncedOrganizations(ctx, nids); err != nil {
			return fmt.Errorf("delete absent organizations: %w", err)
		}

		// one identifier lookup per organization; batch if snapshot size makes this slow.
		for _, organization := range values {
			if err := syncIdentifiers(ctx, q, ids[organization.NID], organization.Input.Identifiers); err != nil {
				return fmt.Errorf("sync organization %d identifiers: %w", organization.NID, err)
			}
		}

		for _, value := range values {
			if err := q.DeleteOrganizationParents(ctx, ids[value.NID]); err != nil {
				return fmt.Errorf("clear organization parents: %w", err)
			}
		}

		for _, value := range values {
			for _, parent := range value.ParentNIDs {
				if err := q.CreateOrganizationParent(ctx, sqlc.CreateOrganizationParentParams{
					ChildOrganizationID:  ids[value.NID],
					ParentOrganizationID: ids[parent],
				}); err != nil {
					return fmt.Errorf("link organization %d parent %d: %w", value.NID, parent, err)
				}
			}
		}

		return nil
	})
}

// syncIdentifiers preserves IDs of unchanged identifiers and removes absent ones.
func syncIdentifiers(ctx context.Context, q *sqlc.Queries, organizationID uuid.UUID, identifiers []domain.OrganizationIdentifierInput) error {
	wanted := make(map[domain.OrganizationIdentifierInput]bool, len(identifiers))
	for _, identifier := range identifiers {
		wanted[identifier] = true
	}

	existing, err := q.ListOrganizationIdentifiers(ctx, []uuid.UUID{organizationID})
	if err != nil {
		return fmt.Errorf("list synced organization identifiers: %w", err)
	}

	for _, row := range existing {
		identifier := row.OrganizationIdentifier
		key := domain.OrganizationIdentifierInput{
			Type:  identifier.Type,
			Value: identifier.Value,
		}
		if wanted[key] {
			delete(wanted, key)
		} else if err := q.DeleteOrganizationIdentifier(ctx, identifier.ID); err != nil {
			return fmt.Errorf("delete synced organization identifier: %w", err)
		}
	}

	for identifier := range wanted {
		if err := q.CreateOrganizationIdentifier(ctx, sqlc.CreateOrganizationIdentifierParams{
			ID:             uuid.New(),
			OrganizationID: organizationID,
			Type:           identifier.Type,
			Value:          identifier.Value,
		}); err != nil {
			return fmt.Errorf("create synced organization identifier: %w", err)
		}
	}

	return nil
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
		if err := missingReferenceError(input.Parents, organizationIDs(parentRows), "parents", "Parent organization"); err != nil {
			return err
		}

		organizationID := uuid.New()
		_, err = q.CreateOrganization(ctx, toCreateOrganizationParams(organizationID, input))
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return apperror.Wrap(apperror.CodeConflict, "", fmt.Errorf("insert organization: %w", err))
			}
			return fmt.Errorf("insert organization: %w", err)
		}

		for _, identifier := range input.Identifiers {
			identifierID := uuid.New()
			err := q.CreateOrganizationIdentifier(ctx, sqlc.CreateOrganizationIdentifierParams{
				ID:             identifierID,
				Type:           identifier.Type,
				Value:          identifier.Value,
				OrganizationID: organizationID,
			})
			if err != nil {
				if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
					return apperror.Wrap(apperror.CodeConflict, "", fmt.Errorf("insert organization identifier: %w", err))
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
			return apperror.Default(apperror.CodeNotFound)
		}
		if err != nil {
			return fmt.Errorf("lock organization: %w", err)
		}

		parentRows, err := q.ListOrganizationsByIDs(ctx, input.Parents)
		if err != nil {
			return fmt.Errorf("check organization parents: %w", err)
		}
		if err := missingReferenceError(input.Parents, organizationIDs(parentRows), "parents", "Parent organization"); err != nil {
			return err
		}

		_, err = q.ReplaceOrganization(ctx, toReplaceOrganizationParams(id, input))
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return apperror.Wrap(apperror.CodeConflict, "", fmt.Errorf("update organization: %w", err))
			}

			return fmt.Errorf("update organization: %w", err)
		}

		if err := q.DeleteOrganizationIdentifiers(ctx, id); err != nil {
			return fmt.Errorf("replace organization identifiers: %w", err)
		}

		if err := q.DeleteOrganizationParents(ctx, id); err != nil {
			return fmt.Errorf("replace organization parents: %w", err)
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
		return apperror.Default(apperror.CodeNotFound)
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
			OrganizationID: id,
		})
		if err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return apperror.Wrap(apperror.CodeConflict, "", fmt.Errorf("insert organization identifier: %w", err))
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
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && (pgErr.Code == "23514" || pgErr.Code == "P0001") {
				return apperror.Wrap(apperror.CodeValidationFailed, "", fmt.Errorf("link organization parent: %w", err))
			}
			return fmt.Errorf("link organization parent: %w", err)
		}
	}

	return nil
}

func organizationIDs(rows []sqlc.Organization) []uuid.UUID {
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids
}

func missingReferenceError(input, found []uuid.UUID, field, label string) error {
	foundIDs := make(map[uuid.UUID]struct{}, len(found))
	for _, id := range found {
		foundIDs[id] = struct{}{}
	}
	for i, id := range input {
		if _, ok := foundIDs[id]; !ok {
			err := apperror.New(apperror.CodeNotFound, label+" was not found.")
			err.Target = fmt.Sprintf("%s[%d]", field, i)
			return err
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
		return domain.Organization{}, apperror.Default(apperror.CodeNotFound)
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
	identifiersByOrganization := make(map[uuid.UUID][]domain.OrganizationIdentifier, len(rows))
	parentsByOrganization := make(map[uuid.UUID][]uuid.UUID, len(rows))
	for _, item := range identifiers {
		identifiersByOrganization[item.OrganizationID] = append(identifiersByOrganization[item.OrganizationID], toDomainOrganizationIdentifier(item))
	}
	for _, item := range parents {
		parentsByOrganization[item.ChildOrganizationID] = append(parentsByOrganization[item.ChildOrganizationID], item.ParentOrganizationID)
	}

	result := make([]domain.Organization, 0, len(rows))
	for _, row := range rows {
		organization, err := toDomainOrganization(
			row,
			identifiersByOrganization[row.ID],
			parentsByOrganization[row.ID],
		)
		if err != nil {
			return nil, fmt.Errorf("construct organization %s: %w", row.ID, err)
		}
		result = append(result, organization)
	}

	return result, nil
}
