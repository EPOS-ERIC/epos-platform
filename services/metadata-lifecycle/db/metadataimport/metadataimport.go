// Package metadataimport provides PostgreSQL persistence for metadata imports.
package metadataimport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	datasetdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dataset"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/dbtx"
	softwareapplicationsdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/softwareapplications"
	softwaresourcecodesdb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/softwaresourcecodes"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
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
func (r *Repo) Import(ctx context.Context, actorID string, batch domain.ImportBatch) (domain.ImportResult, error) {
	result := domain.ImportResult{
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
		return domain.ImportResult{}, fmt.Errorf("database invariant: %s: %w", pgErr.ConstraintName, domain.ErrImportConflict)
	}
	return domain.ImportResult{}, fmt.Errorf("metadata import transaction: %w", err)
}

//nolint:dupl // Typed application and source-code imports use distinct SQLC operations.
func importSoftwareApplication(ctx context.Context, q *sqlc.Queries, actorID string, value domain.SoftwareApplicationMetadata, groupIDs, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID) error {
	groupID, err := softwareGroupID(value.Categories, groupIDs)
	if err != nil {
		return fmt.Errorf("software application %q: %w", value.ID, err)
	}
	remapSoftwareApplicationReferences(&value, organizationIDs, contactIDs, categoryIDs)
	existing, err := q.GetImportSoftwareApplication(ctx, value.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		logical, err := q.CreateImportSoftwareApplication(ctx, sqlc.CreateImportSoftwareApplicationParams{
			ID:        value.ID,
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create imported application: %w", err)
		}
		version, err := q.CreateLifecycleSoftwareApplicationVersion(ctx, softwareapplicationsdb.ToCreateVersionParams(logical.ID, actorID, value.Input()))
		if err != nil {
			return fmt.Errorf("create imported application version: %w", err)
		}
		if err := softwareapplicationsdb.PersistChildren(ctx, q, version.ID, value.Input()); err != nil {
			return fmt.Errorf("persist imported application metadata: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("load imported application: %w", err)
	}
	if existing.GroupID != groupID {
		return fmt.Errorf("application %q belongs to another group: %w", value.ID, domain.ErrImportConflict)
	}
	if existing.ActiveVersionID == nil {
		version, err := q.CreateLifecycleSoftwareApplicationVersion(ctx, softwareapplicationsdb.ToCreateVersionParams(existing.ID, actorID, value.Input()))
		if err != nil {
			return fmt.Errorf("create imported application draft: %w", err)
		}
		if err := softwareapplicationsdb.PersistChildren(ctx, q, version.ID, value.Input()); err != nil {
			return fmt.Errorf("persist imported application metadata: %w", err)
		}
		return nil
	}
	if *existing.ActiveStatus != sqlc.ResourceStatusDRAFT {
		return fmt.Errorf("application %q has non-draft active change: %w", value.ID, domain.ErrImportConflict)
	}
	version, err := q.ReplaceLifecycleSoftwareApplicationDraft(ctx, softwareapplicationsdb.ToReplaceDraftParams(existing.ID, *existing.ActiveVersionID, *existing.ActiveRevision, value.Input()))
	if err != nil {
		return fmt.Errorf("replace imported application draft: %w", err)
	}
	if err := q.DeleteLifecycleSoftwareApplicationVersionChildren(ctx, &version.ID); err != nil {
		return fmt.Errorf("clear imported application draft: %w", err)
	}
	if err := softwareapplicationsdb.PersistChildren(ctx, q, version.ID, value.Input()); err != nil {
		return fmt.Errorf("persist imported application metadata: %w", err)
	}
	return nil
}

//nolint:dupl // Typed application and source-code imports use distinct SQLC operations.
func importSoftwareSourceCode(ctx context.Context, q *sqlc.Queries, actorID string, value domain.SoftwareSourceCodeMetadata, groupIDs, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID) error {
	groupID, err := softwareGroupID(value.Categories, groupIDs)
	if err != nil {
		return fmt.Errorf("software source code %q: %w", value.ID, err)
	}
	remapSoftwareSourceCodeReferences(&value, organizationIDs, contactIDs, categoryIDs)
	existing, err := q.GetImportSoftwareSourceCode(ctx, value.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		logical, err := q.CreateImportSoftwareSourceCode(ctx, sqlc.CreateImportSoftwareSourceCodeParams{
			ID:        value.ID,
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create imported source code: %w", err)
		}
		version, err := q.CreateLifecycleSoftwareSourceCodeVersion(ctx, softwaresourcecodesdb.ToCreateVersionParams(logical.ID, actorID, value.Input()))
		if err != nil {
			return fmt.Errorf("create imported source code version: %w", err)
		}
		if err := softwaresourcecodesdb.PersistChildren(ctx, q, version.ID, value.Input()); err != nil {
			return fmt.Errorf("persist imported source code metadata: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("load imported source code: %w", err)
	}
	if existing.GroupID != groupID {
		return fmt.Errorf("source code %q belongs to another group: %w", value.ID, domain.ErrImportConflict)
	}
	if existing.ActiveVersionID == nil {
		version, err := q.CreateLifecycleSoftwareSourceCodeVersion(ctx, softwaresourcecodesdb.ToCreateVersionParams(existing.ID, actorID, value.Input()))
		if err != nil {
			return fmt.Errorf("create imported source code draft: %w", err)
		}
		if err := softwaresourcecodesdb.PersistChildren(ctx, q, version.ID, value.Input()); err != nil {
			return fmt.Errorf("persist imported source code metadata: %w", err)
		}
		return nil
	}
	if *existing.ActiveStatus != sqlc.ResourceStatusDRAFT {
		return fmt.Errorf("source code %q has non-draft active change: %w", value.ID, domain.ErrImportConflict)
	}
	version, err := q.ReplaceLifecycleSoftwareSourceCodeDraft(ctx, softwaresourcecodesdb.ToReplaceDraftParams(existing.ID, *existing.ActiveVersionID, *existing.ActiveRevision, value.Input()))
	if err != nil {
		return fmt.Errorf("replace imported source code draft: %w", err)
	}
	if err := q.DeleteLifecycleSoftwareSourceCodeVersionChildren(ctx, &version.ID); err != nil {
		return fmt.Errorf("clear imported source code draft: %w", err)
	}
	if err := softwaresourcecodesdb.PersistChildren(ctx, q, version.ID, value.Input()); err != nil {
		return fmt.Errorf("persist imported source code metadata: %w", err)
	}
	return nil
}

func softwareGroupID(categories []domain.CategoryMetadata, groupIDs map[uuid.UUID]uuid.UUID) (uuid.UUID, error) {
	if len(categories) == 0 {
		return uuid.Nil, fmt.Errorf("no category group: %w", domain.ErrInvalidImport)
	}
	groupID, ok := groupIDs[categories[0].Scheme.ID]
	if !ok {
		return uuid.Nil, fmt.Errorf("unknown category scheme: %w", domain.ErrInvalidImport)
	}
	return groupID, nil
}

func remapSoftwareApplicationReferences(value *domain.SoftwareApplicationMetadata, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID) {
	for i := range value.Creators {
		value.Creators[i].ID = organizationIDs[value.Creators[i].ID]
	}
	for i := range value.ContactPoints {
		value.ContactPoints[i].ID = contactIDs[value.ContactPoints[i].ID]
	}
	for i := range value.Categories {
		value.Categories[i].ID = categoryIDs[value.Categories[i].ID]
	}
}

func remapSoftwareSourceCodeReferences(value *domain.SoftwareSourceCodeMetadata, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID) {
	for i := range value.Creators {
		value.Creators[i].ID = organizationIDs[value.Creators[i].ID]
	}
	for i := range value.ContactPoints {
		value.ContactPoints[i].ID = contactIDs[value.ContactPoints[i].ID]
	}
	for i := range value.Categories {
		value.Categories[i].ID = categoryIDs[value.Categories[i].ID]
	}
}

func importGroups(ctx context.Context, q *sqlc.Queries, schemes []domain.Group) (map[uuid.UUID]uuid.UUID, error) {
	ids := make(map[uuid.UUID]uuid.UUID, len(schemes))
	for _, scheme := range schemes {
		id, err := q.UpsertImportGroup(ctx, sqlc.UpsertImportGroupParams{
			ID:          scheme.ID,
			Name:        scheme.Name,
			Description: scheme.Description,
			Concept:     sqlc.TopLevelConcept(scheme.Concept),
			Label:       scheme.Label,
			Logo:        scheme.Logo,
			Page:        scheme.Page,
			Color:       scheme.Color,
			Position:    scheme.Position,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert group %q: %w", scheme.ID, err)
		}
		ids[scheme.ID] = id
	}
	return ids, nil
}

func importOrganizations(ctx context.Context, q *sqlc.Queries, batch domain.ImportBatch) (map[uuid.UUID]uuid.UUID, error) {
	ids := make(map[uuid.UUID]uuid.UUID, len(batch.Organizations))
	stableIDs := make(map[uuid.UUID]uuid.UUID, len(batch.Organizations))
	for _, organization := range batch.Organizations {
		id, err := q.UpsertImportOrganization(ctx, sqlc.UpsertImportOrganizationParams{
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
			return nil, fmt.Errorf("upsert organization %q: %w", organization.ID, err)
		}
		if err := q.DeleteImportOrganizationIdentifiers(ctx, &id); err != nil {
			return nil, fmt.Errorf("replace organization identifiers %q: %w", organization.ID, err)
		}
		for _, identifier := range organization.Identifiers {
			if err := q.AddImportOrganizationIdentifier(ctx, sqlc.AddImportOrganizationIdentifierParams{
				Type:           string(identifier.Type),
				Value:          identifier.Value,
				OrganizationID: &id,
			}); err != nil {
				return nil, fmt.Errorf("add organization identifier %q: %w", organization.ID, err)
			}
		}
		ids[organization.ID] = id
		stableIDs[organization.ID] = id
	}
	for _, organization := range batch.Organizations {
		childID := stableIDs[organization.ID]
		if err := q.DeleteImportOrganizationMemberships(ctx, childID); err != nil {
			return nil, fmt.Errorf("replace organization hierarchy %q: %w", organization.ID, err)
		}
	}
	for _, edge := range batch.OrganizationEdges {
		parentID, parentOK := stableIDs[edge.ParentID]
		childID, childOK := stableIDs[edge.ChildID]
		if !parentOK || !childOK {
			return nil, fmt.Errorf("invalid organization hierarchy: %w", domain.ErrInvalidImport)
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

func importOrganizationContactPoints(ctx context.Context, q *sqlc.Queries, organizations []domain.OrganizationMetadata, organizationIDs, contactIDs map[uuid.UUID]uuid.UUID) error {
	for _, organization := range organizations {
		organizationID := organizationIDs[organization.ID]
		if err := q.DeleteImportOrganizationContactPoints(ctx, organizationID); err != nil {
			return fmt.Errorf("replace organization contact points %q: %w", organization.ID, err)
		}
		for _, contact := range organization.ContactPoints {
			contactID, ok := contactIDs[contact.ID]
			if !ok {
				return fmt.Errorf("organization %q references unknown contact point %q: %w", organization.ID, contact.ID, domain.ErrInvalidImport)
			}
			if err := q.AddImportOrganizationContactPoint(ctx, sqlc.AddImportOrganizationContactPointParams{
				OrganizationID: organizationID,
				ContactPointID: contactID,
			}); err != nil {
				return fmt.Errorf("add organization contact point %q: %w", organization.ID, err)
			}
		}
	}
	return nil
}

func importContactPoints(ctx context.Context, q *sqlc.Queries, contacts []domain.ContactPoint) (map[uuid.UUID]uuid.UUID, error) {
	ids := make(map[uuid.UUID]uuid.UUID, len(contacts))
	for _, contact := range contacts {
		id, err := q.UpsertImportContactPoint(ctx, sqlc.UpsertImportContactPointParams{
			ID:    contact.ID,
			Email: contact.Email,
			Role:  string(contact.Role),
		})
		if err != nil {
			return nil, fmt.Errorf("upsert contact point %q: %w", contact.ID, err)
		}
		ids[contact.ID] = id
	}
	return ids, nil
}

func importCategories(ctx context.Context, q *sqlc.Queries, batch domain.ImportBatch, groupIDs map[uuid.UUID]uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	ids := make(map[uuid.UUID]uuid.UUID, len(batch.Categories))
	groups := make(map[uuid.UUID]uuid.UUID, len(batch.Categories))
	for _, category := range batch.Categories {
		groupID, ok := groupIDs[category.Scheme.ID]
		if !ok {
			return nil, fmt.Errorf("category %q references unknown scheme: %w", category.ID, domain.ErrInvalidImport)
		}
		row, err := q.UpsertImportCategory(ctx, sqlc.UpsertImportCategoryParams{
			ID:      category.ID,
			GroupID: groupID,
			Name:    category.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert category %q: %w", category.ID, err)
		}
		if row.GroupID != groupID {
			return nil, fmt.Errorf("category %q belongs to another group: %w", category.ID, domain.ErrImportConflict)
		}
		ids[category.ID], groups[category.ID] = row.ID, groupID
	}
	for _, edge := range batch.CategoryEdges {
		parentID, parentOK := ids[edge.ParentID]
		childID, childOK := ids[edge.ChildID]
		if !parentOK || !childOK || groups[edge.ParentID] != groups[edge.ChildID] {
			return nil, fmt.Errorf("invalid category hierarchy: %w", domain.ErrInvalidImport)
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
	value domain.DatasetMetadata,
	groupIDs map[uuid.UUID]uuid.UUID,
	organizationIDs, contactIDs map[uuid.UUID]uuid.UUID,
	categoryIDs map[uuid.UUID]uuid.UUID,
) error {
	if len(value.Categories) == 0 {
		return fmt.Errorf("dataset %q has no category group: %w", value.ID, domain.ErrInvalidImport)
	}
	groupID, ok := groupIDs[value.Categories[0].Scheme.ID]
	if !ok {
		return fmt.Errorf("dataset %q references unknown category scheme: %w", value.ID, domain.ErrInvalidImport)
	}
	valueInput := value.Input()
	remapDatasetReferences(&valueInput, organizationIDs, contactIDs, categoryIDs)
	valueInput, err := domain.NewDatasetInput(valueInput)
	if err != nil {
		return fmt.Errorf("validate imported dataset %q: %w", value.ID, err)
	}

	existing, err := q.GetImportDataset(ctx, value.ID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		logical, createErr := q.CreateImportDataset(ctx, sqlc.CreateImportDatasetParams{
			ID:        value.ID,
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if createErr != nil {
			return fmt.Errorf("create imported dataset %q: %w", value.ID, createErr)
		}
		params, buildErr := datasetdb.ToCreateVersionParams(logical.ID, actorID, valueInput)
		if buildErr != nil {
			return fmt.Errorf("build imported dataset version: %w", buildErr)
		}
		version, createErr := q.CreateLifecycleDatasetVersion(ctx, params)
		if createErr != nil {
			return fmt.Errorf("create imported draft %q: %w", value.ID, createErr)
		}
		if err := datasetdb.PersistChildren(ctx, q, version.ID, valueInput); err != nil {
			return fmt.Errorf("persist imported dataset metadata: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("load imported dataset %q: %w", value.ID, err)
	case existing.GroupID != groupID:
		return fmt.Errorf("dataset %q belongs to another group: %w", value.ID, domain.ErrImportConflict)
	case existing.ActiveVersionID == nil:
		params, buildErr := datasetdb.ToCreateVersionParams(existing.ID, actorID, valueInput)
		if buildErr != nil {
			return fmt.Errorf("build imported dataset version: %w", buildErr)
		}
		version, createErr := q.CreateLifecycleDatasetVersion(ctx, params)
		if createErr != nil {
			return fmt.Errorf("create imported draft %q: %w", value.ID, createErr)
		}
		if err := datasetdb.PersistChildren(ctx, q, version.ID, valueInput); err != nil {
			return fmt.Errorf("persist imported dataset metadata: %w", err)
		}
		return nil
	case *existing.ActiveStatus != sqlc.ResourceStatusDRAFT:
		return fmt.Errorf("dataset %q has non-draft active change: %w", value.ID, domain.ErrImportConflict)
	default:
		params, buildErr := datasetdb.ToReplaceDraftParams(existing.ID, *existing.ActiveVersionID, *existing.ActiveRevision, valueInput)
		if buildErr != nil {
			return fmt.Errorf("build imported draft replacement: %w", buildErr)
		}
		version, replaceErr := q.ReplaceLifecycleDatasetDraft(ctx, params)
		if replaceErr != nil {
			return fmt.Errorf("replace imported draft %q: %w", value.ID, replaceErr)
		}
		if err := q.DeleteLifecycleDatasetVersionChildren(ctx, &version.ID); err != nil {
			return fmt.Errorf("clear imported draft %q: %w", value.ID, err)
		}
		if err := datasetdb.PersistChildren(ctx, q, version.ID, valueInput); err != nil {
			return fmt.Errorf("persist imported dataset metadata: %w", err)
		}
		return nil
	}
}

func remapDatasetReferences(value *domain.DatasetInput, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID) {
	for i := range value.ProviderIDs {
		value.ProviderIDs[i] = organizationIDs[value.ProviderIDs[i]]
	}
	for i := range value.ContactPointIDs {
		value.ContactPointIDs[i] = contactIDs[value.ContactPointIDs[i]]
	}
	for i := range value.CategoryIDs {
		value.CategoryIDs[i] = categoryIDs[value.CategoryIDs[i]]
	}
	for i := range value.Distributions {
		service := value.Distributions[i].WebService
		if service == nil {
			continue
		}
		if service.ProviderID != nil {
			*service.ProviderID = organizationIDs[*service.ProviderID]
		}
		for j := range service.ContactPointIDs {
			service.ContactPointIDs[j] = contactIDs[service.ContactPointIDs[j]]
		}
	}
}
