// Package metadataimport provides PostgreSQL persistence for metadata imports.
package metadataimport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	datasetdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dataset"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dbtx"
	softwaredb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/software"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadataimport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	db *pgxpool.Pool
}

// New creates a metadata import repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{db: pool}
}

// Import persists one metadata batch in a transaction.
func (r *Repo) Import(ctx context.Context, actorID string, batch metadataimport.Batch) (metadataimport.Result, error) {
	result := metadataimport.Result{
		Groups:               len(batch.CategorySchemes),
		Categories:           len(batch.Categories),
		Organizations:        len(batch.Organizations),
		ContactPoints:        len(batch.ContactPoints),
		Datasets:             len(batch.Datasets),
		SoftwareApplications: len(batch.SoftwareApplications),
		SoftwareSourceCodes:  len(batch.SoftwareSourceCodes),
		Warnings:             batch.Warnings,
	}

	err := dbtx.Run(ctx, r.db, sqlc.New(r.db), func(q *sqlc.Queries) error {
		groupIDs, err := importGroups(ctx, q, batch.CategorySchemes)
		if err != nil {
			return err
		}
		organizationIDs, err := importOrganizations(ctx, q, batch)
		if err != nil {
			return err
		}
		contactIDs, err := importContactPoints(ctx, q, batch.ContactPoints)
		if err != nil {
			return err
		}
		if err := importOrganizationContactPoints(ctx, q, batch.Organizations, organizationIDs, contactIDs); err != nil {
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
		for _, application := range batch.SoftwareApplications {
			if err := importSoftwareApplication(ctx, q, actorID, application, groupIDs, organizationIDs, contactIDs, categoryIDs); err != nil {
				return err
			}
		}
		for _, sourceCode := range batch.SoftwareSourceCodes {
			if err := importSoftwareSourceCode(ctx, q, actorID, sourceCode, groupIDs, organizationIDs, contactIDs, categoryIDs); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		return result, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (strings.HasPrefix(pgErr.Code, "23") || pgErr.Code == "P0001") {
		return metadataimport.Result{}, fmt.Errorf("database invariant: %s: %w", pgErr.ConstraintName, metadataimport.ErrConflict)
	}
	return metadataimport.Result{}, fmt.Errorf("metadata import transaction: %w", err)
}

//nolint:dupl // Typed application and source-code imports use distinct SQLC operations.
func importSoftwareApplication(ctx context.Context, q *sqlc.Queries, actorID string, value metadata.SoftwareApplication, groupIDs, organizationIDs, contactIDs map[string]uuid.UUID, categoryIDs map[uuid.UUID]uuid.UUID) error {
	groupID, err := softwareGroupID(value.Categories, groupIDs)
	if err != nil {
		return fmt.Errorf("software application %q: %w", value.SourceIRI, err)
	}
	remapSoftwareApplicationReferences(&value, organizationIDs, contactIDs, categoryIDs)
	existing, err := q.GetImportSoftwareApplication(ctx, value.SourceIRI)
	if errors.Is(err, pgx.ErrNoRows) {
		logical, err := q.CreateLifecycleSoftwareApplication(ctx, sqlc.CreateLifecycleSoftwareApplicationParams{SourceIri: value.SourceIRI, GroupID: groupID, CreatedBy: actorID})
		if err != nil {
			return fmt.Errorf("create imported application: %w", err)
		}
		version, err := q.CreateLifecycleSoftwareApplicationVersion(ctx, softwaredb.ApplicationVersionParams(logical.ID, actorID, value))
		if err != nil {
			return fmt.Errorf("create imported application version: %w", err)
		}
		if err := softwaredb.PersistApplicationChildren(ctx, q, version.ID, value); err != nil {
			return fmt.Errorf("persist imported application metadata: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("load imported application: %w", err)
	}
	if existing.GroupID != groupID {
		return fmt.Errorf("application %q belongs to another group: %w", value.SourceIRI, metadataimport.ErrConflict)
	}
	if existing.ActiveVersionID == nil {
		version, err := q.CreateLifecycleSoftwareApplicationVersion(ctx, softwaredb.ApplicationVersionParams(existing.ID, actorID, value))
		if err != nil {
			return fmt.Errorf("create imported application draft: %w", err)
		}
		if err := softwaredb.PersistApplicationChildren(ctx, q, version.ID, value); err != nil {
			return fmt.Errorf("persist imported application metadata: %w", err)
		}
		return nil
	}
	if *existing.ActiveStatus != sqlc.ResourceStatusDRAFT {
		return fmt.Errorf("application %q has non-draft active change: %w", value.SourceIRI, metadataimport.ErrConflict)
	}
	version, err := q.ReplaceLifecycleSoftwareApplicationDraft(ctx, softwaredb.ReplaceApplicationVersionParams(*existing.ActiveVersionID, existing.ID, *existing.ActiveRevision, value))
	if err != nil {
		return fmt.Errorf("replace imported application draft: %w", err)
	}
	if err := q.DeleteLifecycleSoftwareApplicationVersionChildren(ctx, &version.ID); err != nil {
		return fmt.Errorf("clear imported application draft: %w", err)
	}
	if err := softwaredb.PersistApplicationChildren(ctx, q, version.ID, value); err != nil {
		return fmt.Errorf("persist imported application metadata: %w", err)
	}
	return nil
}

//nolint:dupl // Typed application and source-code imports use distinct SQLC operations.
func importSoftwareSourceCode(ctx context.Context, q *sqlc.Queries, actorID string, value metadata.SoftwareSourceCode, groupIDs, organizationIDs, contactIDs map[string]uuid.UUID, categoryIDs map[uuid.UUID]uuid.UUID) error {
	groupID, err := softwareGroupID(value.Categories, groupIDs)
	if err != nil {
		return fmt.Errorf("software source code %q: %w", value.SourceIRI, err)
	}
	remapSoftwareSourceCodeReferences(&value, organizationIDs, contactIDs, categoryIDs)
	existing, err := q.GetImportSoftwareSourceCode(ctx, value.SourceIRI)
	if errors.Is(err, pgx.ErrNoRows) {
		logical, err := q.CreateLifecycleSoftwareSourceCode(ctx, sqlc.CreateLifecycleSoftwareSourceCodeParams{SourceIri: value.SourceIRI, GroupID: groupID, CreatedBy: actorID})
		if err != nil {
			return fmt.Errorf("create imported source code: %w", err)
		}
		version, err := q.CreateLifecycleSoftwareSourceCodeVersion(ctx, softwaredb.SourceCodeVersionParams(logical.ID, actorID, value))
		if err != nil {
			return fmt.Errorf("create imported source code version: %w", err)
		}
		if err := softwaredb.PersistSourceCodeChildren(ctx, q, version.ID, value); err != nil {
			return fmt.Errorf("persist imported source code metadata: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("load imported source code: %w", err)
	}
	if existing.GroupID != groupID {
		return fmt.Errorf("source code %q belongs to another group: %w", value.SourceIRI, metadataimport.ErrConflict)
	}
	if existing.ActiveVersionID == nil {
		version, err := q.CreateLifecycleSoftwareSourceCodeVersion(ctx, softwaredb.SourceCodeVersionParams(existing.ID, actorID, value))
		if err != nil {
			return fmt.Errorf("create imported source code draft: %w", err)
		}
		if err := softwaredb.PersistSourceCodeChildren(ctx, q, version.ID, value); err != nil {
			return fmt.Errorf("persist imported source code metadata: %w", err)
		}
		return nil
	}
	if *existing.ActiveStatus != sqlc.ResourceStatusDRAFT {
		return fmt.Errorf("source code %q has non-draft active change: %w", value.SourceIRI, metadataimport.ErrConflict)
	}
	version, err := q.ReplaceLifecycleSoftwareSourceCodeDraft(ctx, softwaredb.ReplaceSourceCodeVersionParams(*existing.ActiveVersionID, existing.ID, *existing.ActiveRevision, value))
	if err != nil {
		return fmt.Errorf("replace imported source code draft: %w", err)
	}
	if err := q.DeleteLifecycleSoftwareSourceCodeVersionChildren(ctx, &version.ID); err != nil {
		return fmt.Errorf("clear imported source code draft: %w", err)
	}
	if err := softwaredb.PersistSourceCodeChildren(ctx, q, version.ID, value); err != nil {
		return fmt.Errorf("persist imported source code metadata: %w", err)
	}
	return nil
}

func softwareGroupID(categories []metadata.Category, groupIDs map[string]uuid.UUID) (uuid.UUID, error) {
	if len(categories) == 0 {
		return uuid.Nil, fmt.Errorf("no category group: %w", metadataimport.ErrInvalid)
	}
	groupID, ok := groupIDs[categories[0].Scheme.SourceIRI]
	if !ok {
		return uuid.Nil, fmt.Errorf("unknown category scheme: %w", metadataimport.ErrInvalid)
	}
	return groupID, nil
}

func remapSoftwareApplicationReferences(value *metadata.SoftwareApplication, organizationIDs, contactIDs map[string]uuid.UUID, categoryIDs map[uuid.UUID]uuid.UUID) {
	for i := range value.Creators {
		value.Creators[i].ID = organizationIDs[value.Creators[i].SourceIRI]
	}
	for i := range value.ContactPoints {
		value.ContactPoints[i].ID = contactIDs[value.ContactPoints[i].SourceIRI]
	}
	for i := range value.Categories {
		value.Categories[i].ID = categoryIDs[value.Categories[i].ID]
	}
}

func remapSoftwareSourceCodeReferences(value *metadata.SoftwareSourceCode, organizationIDs, contactIDs map[string]uuid.UUID, categoryIDs map[uuid.UUID]uuid.UUID) {
	for i := range value.Creators {
		value.Creators[i].ID = organizationIDs[value.Creators[i].SourceIRI]
	}
	for i := range value.ContactPoints {
		value.ContactPoints[i].ID = contactIDs[value.ContactPoints[i].SourceIRI]
	}
	for i := range value.Categories {
		value.Categories[i].ID = categoryIDs[value.Categories[i].ID]
	}
}

func importGroups(ctx context.Context, q *sqlc.Queries, schemes []metadata.CategoryScheme) (map[string]uuid.UUID, error) {
	ids := make(map[string]uuid.UUID, len(schemes))
	for _, scheme := range schemes {
		id, err := q.UpsertImportGroup(ctx, sqlc.UpsertImportGroupParams{
			ID:          scheme.ID,
			Name:        scheme.Name,
			Description: scheme.Description,
			SourceIri:   scheme.SourceIRI,
			Concept:     sqlc.TopLevelConcept(scheme.Concept),
			Label:       scheme.Label,
			Logo:        scheme.Logo,
			Page:        scheme.Page,
			Color:       scheme.Color,
			Position:    scheme.Position,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert group %q: %w", scheme.SourceIRI, err)
		}
		ids[scheme.SourceIRI] = id
	}
	return ids, nil
}

func importOrganizations(ctx context.Context, q *sqlc.Queries, batch metadataimport.Batch) (map[string]uuid.UUID, error) {
	ids := make(map[string]uuid.UUID, len(batch.Organizations))
	stableIDs := make(map[uuid.UUID]uuid.UUID, len(batch.Organizations))
	for _, organization := range batch.Organizations {
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
		stableIDs[organization.ID] = id
	}
	for _, organization := range batch.Organizations {
		childID := stableIDs[organization.ID]
		if err := q.DeleteImportOrganizationMemberships(ctx, childID); err != nil {
			return nil, fmt.Errorf("replace organization hierarchy %q: %w", organization.SourceIRI, err)
		}
	}
	for _, edge := range batch.OrganizationEdges {
		parentID, parentOK := stableIDs[edge.ParentID]
		childID, childOK := stableIDs[edge.ChildID]
		if !parentOK || !childOK {
			return nil, fmt.Errorf("invalid organization hierarchy: %w", metadataimport.ErrInvalid)
		}
		if err := q.AddImportOrganizationMembership(ctx, sqlc.AddImportOrganizationMembershipParams{
			ChildOrganizationID:  childID,
			ParentOrganizationID: parentID,
		}); err != nil {
			return nil, fmt.Errorf("add organization hierarchy: %w", err)
		}
	}
	return ids, nil
}

func importOrganizationContactPoints(ctx context.Context, q *sqlc.Queries, organizations []metadata.Organization, organizationIDs, contactIDs map[string]uuid.UUID) error {
	for _, organization := range organizations {
		organizationID := organizationIDs[organization.SourceIRI]
		if err := q.DeleteImportOrganizationContactPoints(ctx, organizationID); err != nil {
			return fmt.Errorf("replace organization contact points %q: %w", organization.SourceIRI, err)
		}
		for _, contact := range organization.ContactPoints {
			contactID, ok := contactIDs[contact.SourceIRI]
			if !ok {
				return fmt.Errorf("organization %q references unknown contact point %q: %w", organization.SourceIRI, contact.SourceIRI, metadataimport.ErrInvalid)
			}
			if err := q.AddImportOrganizationContactPoint(ctx, sqlc.AddImportOrganizationContactPointParams{
				OrganizationID: organizationID,
				ContactPointID: contactID,
			}); err != nil {
				return fmt.Errorf("add organization contact point %q: %w", organization.SourceIRI, err)
			}
		}
	}
	return nil
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
	actorID string,
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
