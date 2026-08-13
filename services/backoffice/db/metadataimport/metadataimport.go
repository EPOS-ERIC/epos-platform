// Package metadataimport provides PostgreSQL persistence for metadata imports.
package metadataimport

import (
	"context"
	"errors"
	"fmt"

	datasetdb "github.com/EPOS-ERIC/epos-platform/services/backoffice/db/dataset"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadataimport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{db: pool}
}

func (r *Repository) withTx(ctx context.Context, fn func(*sqlc.Queries) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin import transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(sqlc.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit import transaction: %w", err)
	}
	return nil
}

func (r *Repository) Import(ctx context.Context, actorID uuid.UUID, batch metadataimport.Batch) (metadataimport.Result, error) {
	result := metadataimport.Result{
		Groups:        len(batch.CategorySchemes),
		Categories:    len(batch.Categories),
		Organizations: len(batch.Organizations),
		ContactPoints: len(batch.ContactPoints),
		Datasets:      len(batch.Datasets),
		Warnings:      batch.Warnings,
	}

	err := r.withTx(ctx, func(q *sqlc.Queries) error {
		groupIDs, err := importGroups(ctx, q, batch.CategorySchemes)
		if err != nil {
			return err
		}
		organizationIDs, err := importOrganizations(ctx, q, batch.Organizations)
		if err != nil {
			return err
		}
		contactIDs, err := importContactPoints(ctx, q, batch.ContactPoints)
		if err != nil {
			return err
		}
		categoryIDs, err := importCategories(ctx, q, batch, groupIDs)
		if err != nil {
			return err
		}
		for _, dataset := range batch.Datasets {
			err := importDataset(ctx, q, actorID, dataset, groupIDs, organizationIDs, contactIDs, categoryIDs)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		return result, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23505" || pgErr.Code == "23503" || pgErr.Code == "23514") {
		return metadataimport.Result{}, fmt.Errorf("database invariant: %s: %w", pgErr.ConstraintName, metadataimport.ErrConflict)
	}
	return metadataimport.Result{}, err
}

func importGroups(ctx context.Context, q *sqlc.Queries, schemes []metadata.CategoryScheme) (map[string]uuid.UUID, error) {
	ids := make(map[string]uuid.UUID, len(schemes))
	for _, scheme := range schemes {
		id, err := q.UpsertImportGroup(ctx, sqlc.UpsertImportGroupParams{
			ID:        scheme.ID,
			Name:      scheme.Name,
			SourceIri: scheme.SourceIRI,
			Concept:   sqlc.TopLevelConcept(scheme.Concept),
			Label:     scheme.Label,
			Logo:      scheme.Logo,
			Page:      scheme.Page,
			Color:     scheme.Color,
			Position:  scheme.Position,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert group %q: %w", scheme.SourceIRI, err)
		}
		ids[scheme.SourceIRI] = id
	}
	return ids, nil
}

func importOrganizations(ctx context.Context, q *sqlc.Queries, organizations []metadata.Organization) (map[string]uuid.UUID, error) {
	ids := make(map[string]uuid.UUID, len(organizations))
	for _, organization := range organizations {
		id, err := q.UpsertImportOrganization(ctx, sqlc.UpsertImportOrganizationParams{
			ID:         organization.ID,
			SourceIri:  organization.SourceIRI,
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
			return nil, fmt.Errorf("upsert organization %q: %w", organization.SourceIRI, err)
		}
		if err := q.DeleteImportOrganizationIdentifiers(ctx, &id); err != nil {
			return nil, fmt.Errorf("replace organization identifiers %q: %w", organization.SourceIRI, err)
		}
		for _, identifier := range organization.Identifiers {
			if err := q.AddImportOrganizationIdentifier(ctx, sqlc.AddImportOrganizationIdentifierParams{
				SourceIri:      identifier.SourceIRI,
				Type:           string(identifier.Type),
				Value:          identifier.Value,
				OrganizationID: &id,
			}); err != nil {
				return nil, fmt.Errorf("add organization identifier %q: %w", organization.SourceIRI, err)
			}
		}
		ids[organization.SourceIRI] = id
	}
	return ids, nil
}

func importContactPoints(ctx context.Context, q *sqlc.Queries, contacts []metadata.ContactPoint) (map[string]uuid.UUID, error) {
	ids := make(map[string]uuid.UUID, len(contacts))
	for _, contact := range contacts {
		id, err := q.UpsertImportContactPoint(ctx, sqlc.UpsertImportContactPointParams{
			ID:        contact.ID,
			SourceIri: contact.SourceIRI,
			Email:     contact.Email,
			Role:      string(contact.Role),
		})
		if err != nil {
			return nil, fmt.Errorf("upsert contact point %q: %w", contact.SourceIRI, err)
		}
		ids[contact.SourceIRI] = id
	}
	return ids, nil
}

func importCategories(ctx context.Context, q *sqlc.Queries, batch metadataimport.Batch, groupIDs map[string]uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	ids := make(map[uuid.UUID]uuid.UUID, len(batch.Categories))
	groups := make(map[uuid.UUID]uuid.UUID, len(batch.Categories))
	for _, category := range batch.Categories {
		groupID, ok := groupIDs[category.Scheme.SourceIRI]
		if !ok {
			return nil, fmt.Errorf("category %q references unknown scheme: %w", category.SourceIRI, metadataimport.ErrInvalid)
		}
		row, err := q.UpsertImportCategory(ctx, sqlc.UpsertImportCategoryParams{
			ID:        category.ID,
			GroupID:   groupID,
			SourceIri: category.SourceIRI,
			Name:      category.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert category %q: %w", category.SourceIRI, err)
		}
		if row.GroupID != groupID {
			return nil, fmt.Errorf("category %q belongs to another group: %w", category.SourceIRI, metadataimport.ErrConflict)
		}
		ids[category.ID], groups[category.ID] = row.ID, groupID
	}
	for _, edge := range batch.CategoryEdges {
		parentID, parentOK := ids[edge.ParentID]
		childID, childOK := ids[edge.ChildID]
		if !parentOK || !childOK || groups[edge.ParentID] != groups[edge.ChildID] {
			return nil, fmt.Errorf("invalid category hierarchy: %w", metadataimport.ErrInvalid)
		}
		if err := q.SetImportCategoryParent(ctx, sqlc.SetImportCategoryParentParams{
			ParentID: &parentID,
			ChildID:  childID,
			GroupID:  groups[edge.ChildID],
		}); err != nil {
			return nil, fmt.Errorf("set category parent: %w", err)
		}
	}
	return ids, nil
}

func importDataset(
	ctx context.Context,
	q *sqlc.Queries,
	actorID uuid.UUID,
	value metadata.Dataset,
	groupIDs map[string]uuid.UUID,
	organizationIDs, contactIDs map[string]uuid.UUID,
	categoryIDs map[uuid.UUID]uuid.UUID,
) error {
	groupID := groupIDs[value.Categories[0].Scheme.SourceIRI]
	remapDatasetReferences(&value, organizationIDs, contactIDs, categoryIDs)

	existing, err := q.GetImportDataset(ctx, value.SourceIRI)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		logical, createErr := q.CreateLifecycleDataset(ctx, sqlc.CreateLifecycleDatasetParams{
			SourceIri: value.SourceIRI,
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if createErr != nil {
			return fmt.Errorf("create imported dataset %q: %w", value.SourceIRI, createErr)
		}
		params, buildErr := datasetdb.VersionParams(logical.ID, actorID, value)
		if buildErr != nil {
			return fmt.Errorf("build imported dataset version: %w", buildErr)
		}
		version, createErr := q.CreateLifecycleDatasetVersion(ctx, params)
		if createErr != nil {
			return fmt.Errorf("create imported draft %q: %w", value.SourceIRI, createErr)
		}
		if err := datasetdb.PersistDatasetChildren(ctx, q, version.ID, value); err != nil {
			return fmt.Errorf("persist imported dataset metadata: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("load imported dataset %q: %w", value.SourceIRI, err)
	case existing.GroupID != groupID:
		return fmt.Errorf("dataset %q belongs to another group: %w", value.SourceIRI, metadataimport.ErrConflict)
	case existing.ActiveVersionID == nil:
		params, buildErr := datasetdb.VersionParams(existing.ID, actorID, value)
		if buildErr != nil {
			return fmt.Errorf("build imported dataset version: %w", buildErr)
		}
		version, createErr := q.CreateLifecycleDatasetVersion(ctx, params)
		if createErr != nil {
			return fmt.Errorf("create imported draft %q: %w", value.SourceIRI, createErr)
		}
		if err := datasetdb.PersistDatasetChildren(ctx, q, version.ID, value); err != nil {
			return fmt.Errorf("persist imported dataset metadata: %w", err)
		}
		return nil
	case *existing.ActiveStatus != sqlc.ResourceStatusDRAFT:
		return fmt.Errorf("dataset %q has non-draft active change: %w", value.SourceIRI, metadataimport.ErrConflict)
	default:
		params, buildErr := datasetdb.ReplaceParams(existing.ID, *existing.ActiveVersionID, *existing.ActiveRevision, value)
		if buildErr != nil {
			return fmt.Errorf("build imported draft replacement: %w", buildErr)
		}
		version, replaceErr := q.ReplaceLifecycleDatasetDraft(ctx, params)
		if replaceErr != nil {
			return fmt.Errorf("replace imported draft %q: %w", value.SourceIRI, replaceErr)
		}
		if err := q.DeleteLifecycleDatasetVersionChildren(ctx, &version.ID); err != nil {
			return fmt.Errorf("clear imported draft %q: %w", value.SourceIRI, err)
		}
		if err := datasetdb.PersistDatasetChildren(ctx, q, version.ID, value); err != nil {
			return fmt.Errorf("persist imported dataset metadata: %w", err)
		}
		return nil
	}
}

func remapDatasetReferences(value *metadata.Dataset, organizationIDs, contactIDs map[string]uuid.UUID, categoryIDs map[uuid.UUID]uuid.UUID) {
	for i := range value.Providers {
		value.Providers[i].ID = organizationIDs[value.Providers[i].SourceIRI]
	}
	for i := range value.ContactPoints {
		value.ContactPoints[i].ID = contactIDs[value.ContactPoints[i].SourceIRI]
	}
	for i := range value.Categories {
		value.Categories[i].ID = categoryIDs[value.Categories[i].ID]
	}
	for i := range value.Distributions {
		service := value.Distributions[i].WebService
		if service == nil {
			continue
		}
		if service.Provider != nil {
			service.Provider.ID = organizationIDs[service.Provider.SourceIRI]
		}
		for j := range service.ContactPoints {
			service.ContactPoints[j].ID = contactIDs[service.ContactPoints[j].SourceIRI]
		}
	}
}
