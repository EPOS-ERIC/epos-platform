// Package metadataimport provides PostgreSQL persistence for metadata imports.
package metadataimport

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
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
	q  *sqlc.Queries
}

// New creates a metadata import repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{db: pool, q: sqlc.New(pool)}
}

// Import persists one metadata batch in a transaction.
func (r *Repo) Import(ctx context.Context, actor string, batch domain.ImportBatch) (domain.ImportResult, error) {
	result := domain.ImportResult{Warnings: batch.Warnings}

	err := dbtx.RunWithTx(ctx, r.db, r.q, func(q *sqlc.Queries, tx pgx.Tx) error {
		groupIDs, err := importGroups(ctx, tx, q, batch.Groups, &result)
		if err != nil {
			return err
		}
		var organizationIDs map[uuid.UUID]uuid.UUID
		if batch.ResolveSyncedOrganizations {
			organizationIDs, err = resolveSyncedOrganizations(ctx, q, batch.Organizations, &result)
		} else {
			organizationIDs, err = importOrganizations(ctx, tx, q, batch.Organizations, &result)
		}
		if err != nil {
			return err
		}
		contactIDs, err := importContactPoints(ctx, tx, q, batch.ContactPoints, &result)
		if err != nil {
			return err
		}
		if !batch.ResolveSyncedOrganizations {
			if err := importOrganizationMemberships(ctx, tx, q, batch, organizationIDs, &result); err != nil {
				return err
			}
		}
		categoryIDs, err := importCategories(ctx, tx, q, batch, groupIDs, &result)
		if err != nil {
			return err
		}
		for _, dataset := range batch.Datasets {
			value, ok := prepareDataset(dataset, organizationIDs, contactIDs, categoryIDs, &result)
			if !ok {
				continue
			}
			imported, err := importEntity(ctx, tx, "dataset", dataset.ID.String(), &result, func() error {
				return importDataset(ctx, q, actor, value, groupIDs, organizationIDs, contactIDs, categoryIDs)
			})
			if err != nil {
				return err
			}
			if imported {
				result.Datasets++
			}
		}
		for _, application := range batch.SoftwareApplications {
			value, ok := prepareSoftwareApplication(application, organizationIDs, contactIDs, categoryIDs, &result)
			if !ok {
				continue
			}
			imported, err := importEntity(ctx, tx, "software application", application.ID.String(), &result, func() error {
				return importSoftwareApplication(ctx, q, actor, value, groupIDs, organizationIDs, contactIDs, categoryIDs)
			})
			if err != nil {
				return err
			}
			if imported {
				result.SoftwareApplications++
			}
		}
		for _, sourceCode := range batch.SoftwareSourceCodes {
			value, ok := prepareSoftwareSourceCode(sourceCode, organizationIDs, contactIDs, categoryIDs, &result)
			if !ok {
				continue
			}
			imported, err := importEntity(ctx, tx, "software source code", sourceCode.ID.String(), &result, func() error {
				return importSoftwareSourceCode(ctx, q, actor, value, groupIDs, organizationIDs, contactIDs, categoryIDs)
			})
			if err != nil {
				return err
			}
			if imported {
				result.SoftwareSourceCodes++
			}
		}
		return nil
	})
	if err == nil {
		return result, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (strings.HasPrefix(pgErr.Code, "23") || pgErr.Code == "P0001") {
		return domain.ImportResult{}, fmt.Errorf("database invariant: %s: %w", pgErr.ConstraintName, errors.Join(apperror.Default(apperror.CodeConflict), pgErr))
	}
	return domain.ImportResult{}, fmt.Errorf("metadata import transaction: %w", err)
}

func resolveSyncedOrganizations(ctx context.Context, q *sqlc.Queries, organizations []domain.OrganizationMetadata, result *domain.ImportResult) (map[uuid.UUID]uuid.UUID, error) {
	ids := make(map[uuid.UUID]uuid.UUID, len(organizations))
	for _, organization := range organizations {
		matches, err := q.ListSyncedOrganizationsByName(ctx, organization.Name)
		if err != nil {
			return nil, fmt.Errorf("find synced organization %q: %w", organization.Name, err)
		}
		if len(matches) != 1 {
			addImportDiagnostic(result, domain.ImportDiagnostic{
				Severity: domain.ImportSeverityWarning,
				Subject:  organization.ID.String(),
				Message:  fmt.Sprintf("organization %q has %d synced matches; relationships omitted", organization.Name, len(matches)),
			})
			continue
		}
		ids[organization.ID] = matches[0]
	}
	return ids, nil
}

func importEntity(ctx context.Context, tx pgx.Tx, kind, subject string, result *domain.ImportResult, fn func() error) (bool, error) {
	err := withSavepoint(ctx, tx, savepointName(kind, subject), fn)
	if err == nil {
		return true, nil
	}
	if !recoverableImportError(err) {
		return false, err
	}
	addImportDiagnostic(result, domain.ImportDiagnostic{
		Severity: domain.ImportSeverityError,
		Subject:  subject,
		Message:  fmt.Sprintf("%s skipped: %s", kind, err),
	})
	return false, nil
}

func addImportDiagnostic(result *domain.ImportResult, diagnostic domain.ImportDiagnostic) {
	result.Warnings = append(result.Warnings, diagnostic)
}

func withSavepoint(ctx context.Context, tx pgx.Tx, name string, fn func() error) error {
	if _, err := tx.Exec(ctx, "SAVEPOINT "+name); err != nil {
		return fmt.Errorf("create savepoint %q: %w", name, err)
	}
	err := fn()
	if err == nil {
		if _, releaseErr := tx.Exec(ctx, "RELEASE SAVEPOINT "+name); releaseErr != nil {
			return fmt.Errorf("release savepoint %q: %w", name, releaseErr)
		}
		return nil
	}
	if _, rollbackErr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+name); rollbackErr != nil {
		return errors.Join(err, fmt.Errorf("rollback savepoint %q: %w", name, rollbackErr))
	}
	if _, releaseErr := tx.Exec(ctx, "RELEASE SAVEPOINT "+name); releaseErr != nil {
		return errors.Join(err, fmt.Errorf("release savepoint %q: %w", name, releaseErr))
	}
	return err
}

func savepointName(kind, subject string) string {
	return "import_" + strings.ReplaceAll(uuid.NewSHA1(uuid.NameSpaceURL, []byte(kind+"\x00"+subject)).String(), "-", "_")
}

func recoverableImportError(err error) bool {
	if apperror.IsCode(err, apperror.CodeValidationFailed) || apperror.IsCode(err, apperror.CodeConflict) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (strings.HasPrefix(pgErr.Code, "23") || pgErr.Code == "P0001")
}

//nolint:dupl // Typed application and source-code imports use distinct SQLC operations.
func importSoftwareApplication(ctx context.Context, q *sqlc.Queries, actorID string, value domain.SoftwareApplicationMetadata, groupIDs, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID) error {
	groupID, err := softwareGroupID(value.Categories, groupIDs)
	if err != nil {
		return fmt.Errorf("software application %q: %w", value.ID, err)
	}
	remapSoftwareApplicationReferences(&value, organizationIDs, contactIDs, categoryIDs)
	_, err = q.GetImportSoftwareApplication(ctx, value.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		logical, err := q.CreateImportSoftwareApplication(ctx, sqlc.CreateImportSoftwareApplicationParams{
			ID:        value.ID,
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create imported application: %w", err)
		}
		input := toApplicationInput(value)
		version, err := q.CreateLifecycleSoftwareApplicationVersion(ctx, softwareapplicationsdb.ToCreateVersionParams(logical.ID, actorID, input))
		if err != nil {
			return fmt.Errorf("create imported application version: %w", err)
		}
		if err := softwareapplicationsdb.PersistChildren(ctx, q, version.ID, input); err != nil {
			return fmt.Errorf("persist imported application metadata: %w", err)
		}
		if _, err := q.SubmitLifecycleSoftwareApplicationVersion(ctx, sqlc.SubmitLifecycleSoftwareApplicationVersionParams{ID: version.ID, ActorID: &actorID}); err != nil {
			return fmt.Errorf("submit imported application: %w", err)
		}
		if _, err := q.PublishLifecycleSoftwareApplicationVersion(ctx, sqlc.PublishLifecycleSoftwareApplicationVersionParams{VersionID: version.ID, ActorID: &actorID}); err != nil {
			return fmt.Errorf("publish imported application: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("load imported application: %w", err)
	}
	return fmt.Errorf("application %q already exists: %w", value.ID, apperror.Default(apperror.CodeConflict))
}

//nolint:dupl // Typed application and source-code imports use distinct SQLC operations.
func importSoftwareSourceCode(ctx context.Context, q *sqlc.Queries, actorID string, value domain.SoftwareSourceCodeMetadata, groupIDs, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID) error {
	groupID, err := softwareGroupID(value.Categories, groupIDs)
	if err != nil {
		return fmt.Errorf("software source code %q: %w", value.ID, err)
	}
	remapSoftwareSourceCodeReferences(&value, organizationIDs, contactIDs, categoryIDs)
	_, err = q.GetImportSoftwareSourceCode(ctx, value.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		logical, err := q.CreateImportSoftwareSourceCode(ctx, sqlc.CreateImportSoftwareSourceCodeParams{
			ID:        value.ID,
			GroupID:   groupID,
			CreatedBy: actorID,
		})
		if err != nil {
			return fmt.Errorf("create imported source code: %w", err)
		}
		input := toSourceCodeInput(value)
		version, err := q.CreateLifecycleSoftwareSourceCodeVersion(ctx, softwaresourcecodesdb.ToCreateVersionParams(logical.ID, actorID, input))
		if err != nil {
			return fmt.Errorf("create imported source code version: %w", err)
		}
		if err := softwaresourcecodesdb.PersistChildren(ctx, q, version.ID, input); err != nil {
			return fmt.Errorf("persist imported source code metadata: %w", err)
		}
		if _, err := q.SubmitLifecycleSoftwareSourceCodeVersion(ctx, sqlc.SubmitLifecycleSoftwareSourceCodeVersionParams{ID: version.ID, ActorID: &actorID}); err != nil {
			return fmt.Errorf("submit imported source code: %w", err)
		}
		if _, err := q.PublishLifecycleSoftwareSourceCodeVersion(ctx, sqlc.PublishLifecycleSoftwareSourceCodeVersionParams{VersionID: version.ID, ActorID: &actorID}); err != nil {
			return fmt.Errorf("publish imported source code: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("load imported source code: %w", err)
	}
	return fmt.Errorf("source code %q already exists: %w", value.ID, apperror.Default(apperror.CodeConflict))
}

func softwareGroupID(categories []domain.CategoryMetadata, groupIDs map[uuid.UUID]uuid.UUID) (uuid.UUID, error) {
	if len(categories) == 0 {
		return uuid.Nil, fmt.Errorf("no category group: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	groupID, ok := groupIDs[categories[0].GroupID]
	if !ok {
		return uuid.Nil, fmt.Errorf("unknown category group: %w", apperror.Default(apperror.CodeValidationFailed))
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

func importGroups(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, groups []domain.ImportGroup, result *domain.ImportResult) (map[uuid.UUID]uuid.UUID, error) {
	ids := make(map[uuid.UUID]uuid.UUID, len(groups))
	for _, group := range groups {
		var id uuid.UUID
		imported, err := importEntity(ctx, tx, "group", group.ID.String(), result, func() error {
			var err error
			id, err = q.UpsertImportGroup(ctx, sqlc.UpsertImportGroupParams{
				ID:          group.ID,
				Name:        group.Name,
				Description: group.Description,
				Concept:     string(group.Concept),
				Label:       group.Label,
				Logo:        group.Logo,
				Page:        group.Page,
				Color:       group.Color,
				Position:    group.Position,
			})
			if err != nil {
				return fmt.Errorf("upsert group %q: %w", group.ID, err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if !imported {
			continue
		}
		ids[group.ID] = id
		result.Groups++
	}
	return ids, nil
}

func importOrganizations(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, organizations []domain.OrganizationMetadata, result *domain.ImportResult) (map[uuid.UUID]uuid.UUID, error) {
	ids := make(map[uuid.UUID]uuid.UUID, len(organizations))
	ordered := append([]domain.OrganizationMetadata(nil), organizations...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Name != ordered[j].Name {
			return ordered[i].Name < ordered[j].Name
		}
		return ordered[i].ID.String() < ordered[j].ID.String()
	})
	canonical := make([]domain.OrganizationMetadata, 0, len(ordered))
	sourceIDsByName := make(map[string][]uuid.UUID, len(ordered))
	canonicalIndexByName := make(map[string]int, len(ordered))
	for _, organization := range ordered {
		index, ok := canonicalIndexByName[organization.Name]
		if !ok {
			index = len(canonical)
			canonical = append(canonical, organization)
			canonicalIndexByName[organization.Name] = index
		} else {
			for _, identifier := range organization.Identifiers {
				canonical[index].Identifiers = appendUniqueIdentifier(canonical[index].Identifiers, identifier)
			}
		}
		sourceIDsByName[organization.Name] = append(sourceIDsByName[organization.Name], organization.ID)
	}

	for _, organization := range canonical {
		var id uuid.UUID
		imported, err := importEntity(ctx, tx, "organization", organization.ID.String(), result, func() error {
			var err error
			id, err = q.GetImportOrganizationByName(ctx, organization.Name)
			if errors.Is(err, pgx.ErrNoRows) {
				id = organization.ID
			} else if err != nil {
				return fmt.Errorf("find organization %q: %w", organization.ID, err)
			}
			id, err = q.UpsertImportOrganization(ctx, sqlc.UpsertImportOrganizationParams{
				ID:         id,
				Name:       organization.Name,
				Street:     organization.Street,
				Locality:   organization.Locality,
				PostalCode: organization.PostalCode,
				Country:    organization.Country,
				LogoUrl:    organization.LogoURL,
				PageUrl:    organization.PageURL,
				Email:      organization.Email,
			})
			if err != nil {
				return fmt.Errorf("upsert organization %q: %w", organization.ID, err)
			}
			if err := q.DeleteImportOrganizationIdentifiers(ctx, id); err != nil {
				return fmt.Errorf("replace organization identifiers %q: %w", organization.ID, err)
			}
			for _, identifier := range organization.Identifiers {
				if err := q.AddImportOrganizationIdentifier(ctx, sqlc.AddImportOrganizationIdentifierParams{
					Type:           string(identifier.Type),
					Value:          identifier.Value,
					OrganizationID: id,
				}); err != nil {
					return fmt.Errorf("add organization identifier %q: %w", organization.ID, err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if imported {
			for _, sourceID := range sourceIDsByName[organization.Name] {
				ids[sourceID] = id
			}
			result.Organizations++
		}
	}
	return ids, nil
}

func appendUniqueIdentifier(values []domain.Identifier, value domain.Identifier) []domain.Identifier {
	for _, existing := range values {
		if existing.Type == value.Type && existing.Value == value.Value {
			return values
		}
	}
	return append(values, value)
}

func importOrganizationMemberships(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, batch domain.ImportBatch, organizationIDs map[uuid.UUID]uuid.UUID, result *domain.ImportResult) error {
	for _, organization := range batch.Organizations {
		organizationID, ok := organizationIDs[organization.ID]
		if !ok {
			continue
		}
		if _, err := importEntity(ctx, tx, "organization hierarchy", organization.ID.String(), result, func() error {
			if err := q.DeleteImportOrganizationMemberships(ctx, organizationID); err != nil {
				return fmt.Errorf("replace organization hierarchy %q: %w", organization.ID, err)
			}
			return nil
		}); err != nil {
			return err
		}
	}

	for _, edge := range batch.OrganizationEdges {
		parentID, parentOK := organizationIDs[edge.ParentID]
		childID, childOK := organizationIDs[edge.ChildID]
		if !parentOK || !childOK {
			addImportDiagnostic(result, domain.ImportDiagnostic{
				Severity: domain.ImportSeverityWarning,
				Subject:  edge.ChildID.String(),
				Message:  fmt.Sprintf("organization hierarchy %q -> %q omitted: organization is invalid", edge.ChildID, edge.ParentID),
			})
			continue
		}
		if _, err := importEntity(ctx, tx, "organization hierarchy", edge.ChildID.String(), result, func() error {
			if err := q.AddImportOrganizationMembership(ctx, sqlc.AddImportOrganizationMembershipParams{
				ChildOrganizationID:  childID,
				ParentOrganizationID: parentID,
			}); err != nil {
				return fmt.Errorf("add organization hierarchy: %w", err)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func importContactPoints(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, contacts []domain.ContactPoint, result *domain.ImportResult) (map[uuid.UUID]uuid.UUID, error) {
	ids := make(map[uuid.UUID]uuid.UUID, len(contacts))
	importedIDs := make(map[uuid.UUID]struct{}, len(contacts))
	for _, contact := range contacts {
		var id uuid.UUID
		imported, err := importEntity(ctx, tx, "contact point", contact.ID.String(), result, func() error {
			var err error
			id, err = q.UpsertImportContactPoint(ctx, sqlc.UpsertImportContactPointParams{
				ID:    contact.ID,
				Email: contact.Email,
			})
			if err != nil {
				return fmt.Errorf("upsert contact point %q: %w", contact.ID, err)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if imported {
			ids[contact.ID] = id
			if _, ok := importedIDs[id]; !ok {
				importedIDs[id] = struct{}{}
				result.ContactPoints++
			}
		}
	}
	return ids, nil
}

func importCategories(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, batch domain.ImportBatch, groupIDs map[uuid.UUID]uuid.UUID, result *domain.ImportResult) (map[uuid.UUID]uuid.UUID, error) {
	ids := make(map[uuid.UUID]uuid.UUID, len(batch.Categories))
	groups := make(map[uuid.UUID]uuid.UUID, len(batch.Categories))
	for _, category := range batch.Categories {
		groupID, ok := groupIDs[category.GroupID]
		if !ok {
			addImportDiagnostic(result, domain.ImportDiagnostic{
				Severity: domain.ImportSeverityError,
				Subject:  category.ID.String(),
				Message:  fmt.Sprintf("category skipped: group %q is unavailable", category.GroupID),
			})
			continue
		}
		var row sqlc.UpsertImportCategoryRow
		imported, err := importEntity(ctx, tx, "category", category.ID.String(), result, func() error {
			var err error
			row, err = q.UpsertImportCategory(ctx, sqlc.UpsertImportCategoryParams{
				ID:      category.ID,
				GroupID: groupID,
				Name:    category.Name,
			})
			if err != nil {
				return fmt.Errorf("upsert category %q: %w", category.ID, err)
			}
			if row.GroupID != groupID {
				return fmt.Errorf("category %q belongs to another group: %w", category.ID, apperror.Default(apperror.CodeConflict))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if imported {
			ids[category.ID], groups[category.ID] = row.ID, groupID
			result.Categories++
		}
	}
	for _, edge := range batch.CategoryEdges {
		parentID, parentOK := ids[edge.ParentID]
		childID, childOK := ids[edge.ChildID]
		if !parentOK || !childOK || groups[edge.ParentID] != groups[edge.ChildID] {
			addImportDiagnostic(result, domain.ImportDiagnostic{
				Severity: domain.ImportSeverityWarning,
				Subject:  edge.ChildID.String(),
				Message:  fmt.Sprintf("category parent %q omitted: category is unavailable", edge.ParentID),
			})
			continue
		}
		if _, err := importEntity(ctx, tx, "category hierarchy", edge.ChildID.String(), result, func() error {
			if err := q.SetImportCategoryParent(ctx, sqlc.SetImportCategoryParentParams{
				ParentID: &parentID,
				ChildID:  childID,
				GroupID:  groups[edge.ChildID],
			}); err != nil {
				return fmt.Errorf("set category parent: %w", err)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

func prepareDataset(value domain.DatasetMetadata, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID, result *domain.ImportResult) (domain.DatasetMetadata, bool) {
	providers := make([]domain.OrganizationMetadata, 0, len(value.Providers))
	for _, provider := range value.Providers {
		if _, ok := organizationIDs[provider.ID]; !ok {
			addMissingReferenceWarning(result, value.ID, "provider", "organization", provider.ID)
			continue
		}
		providers = append(providers, provider)
	}
	value.Providers = providers

	contacts := make([]domain.ContactPoint, 0, len(value.ContactPoints))
	for _, contact := range value.ContactPoints {
		if _, ok := contactIDs[contact.ID]; !ok {
			addMissingReferenceWarning(result, value.ID, "contactPoint", "contact point", contact.ID)
			continue
		}
		contacts = append(contacts, contact)
	}
	value.ContactPoints = contacts

	categories := make([]domain.CategoryMetadata, 0, len(value.Categories))
	for _, category := range value.Categories {
		if _, ok := categoryIDs[category.ID]; !ok {
			addMissingReferenceWarning(result, value.ID, "theme", "category", category.ID)
			continue
		}
		categories = append(categories, category)
	}
	value.Categories = categories

	for i := range value.Distributions {
		service := value.Distributions[i].WebService
		if service == nil {
			continue
		}

		common := webServiceCommon(service)
		if common == nil {
			addImportDiagnostic(result, domain.ImportDiagnostic{
				Severity: domain.ImportSeverityError,
				Subject:  value.ID.String(),
				Message:  fmt.Sprintf("dataset skipped: %s", apperror.Default(apperror.CodeInternal)),
			})
			return domain.DatasetMetadata{}, false
		}

		if common.Provider != nil {
			if _, ok := organizationIDs[common.Provider.ID]; !ok {
				addMissingReferenceWarning(result, value.ID, "provider", "organization", common.Provider.ID)
				common.Provider = nil
			}
		}
		serviceContacts := make([]domain.ContactPoint, 0, len(common.ContactPoints))
		for _, contact := range common.ContactPoints {
			if _, ok := contactIDs[contact.ID]; !ok {
				addMissingReferenceWarning(result, value.ID, "contactPoint", "contact point", contact.ID)
				continue
			}
			serviceContacts = append(serviceContacts, contact)
		}
		common.ContactPoints = serviceContacts
	}

	if err := value.Normalize(); err != nil {
		addImportDiagnostic(result, domain.ImportDiagnostic{
			Severity: domain.ImportSeverityError,
			Subject:  value.ID.String(),
			Message:  fmt.Sprintf("dataset skipped: %s", err),
		})
		return domain.DatasetMetadata{}, false
	}
	batch := domain.ImportBatch{
		Datasets: []domain.DatasetMetadata{value},
	}
	if err := batch.Normalize(); err != nil {
		addImportDiagnostic(result, domain.ImportDiagnostic{
			Severity: domain.ImportSeverityError,
			Subject:  value.ID.String(),
			Message:  fmt.Sprintf("dataset skipped: %s", err),
		})
		return domain.DatasetMetadata{}, false
	}
	return value, true
}

//nolint:dupl // Typed application and source-code preparation use distinct domain types.
func prepareSoftwareApplication(value domain.SoftwareApplicationMetadata, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID, result *domain.ImportResult) (domain.SoftwareApplicationMetadata, bool) {
	creators := make([]domain.OrganizationMetadata, 0, len(value.Creators))
	for _, creator := range value.Creators {
		if _, ok := organizationIDs[creator.ID]; !ok {
			addMissingReferenceWarning(result, value.ID, "creator", "organization", creator.ID)
			continue
		}
		creators = append(creators, creator)
	}
	value.Creators = creators
	value.ContactPoints = filterContactPoints(value.ID, "contactPoint", value.ContactPoints, contactIDs, result)
	value.Categories = filterCategories(value.ID, value.Categories, categoryIDs, result)

	if err := value.Normalize(); err != nil {
		addImportDiagnostic(result, domain.ImportDiagnostic{
			Severity: domain.ImportSeverityError,
			Subject:  value.ID.String(),
			Message:  fmt.Sprintf("software application skipped: %s", err),
		})
		return domain.SoftwareApplicationMetadata{}, false
	}
	batch := domain.ImportBatch{SoftwareApplications: []domain.SoftwareApplicationMetadata{value}}
	if err := batch.Normalize(); err != nil {
		addImportDiagnostic(result, domain.ImportDiagnostic{
			Severity: domain.ImportSeverityError,
			Subject:  value.ID.String(),
			Message:  fmt.Sprintf("software application skipped: %s", err),
		})
		return domain.SoftwareApplicationMetadata{}, false
	}
	return value, true
}

//nolint:dupl // Typed application and source-code preparation use distinct domain types.
func prepareSoftwareSourceCode(value domain.SoftwareSourceCodeMetadata, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID, result *domain.ImportResult) (domain.SoftwareSourceCodeMetadata, bool) {
	creators := make([]domain.OrganizationMetadata, 0, len(value.Creators))
	for _, creator := range value.Creators {
		if _, ok := organizationIDs[creator.ID]; !ok {
			addMissingReferenceWarning(result, value.ID, "creator", "organization", creator.ID)
			continue
		}
		creators = append(creators, creator)
	}
	value.Creators = creators
	value.ContactPoints = filterContactPoints(value.ID, "contactPoint", value.ContactPoints, contactIDs, result)
	value.Categories = filterCategories(value.ID, value.Categories, categoryIDs, result)

	if err := value.Normalize(); err != nil {
		addImportDiagnostic(result, domain.ImportDiagnostic{
			Severity: domain.ImportSeverityError,
			Subject:  value.ID.String(),
			Message:  fmt.Sprintf("software source code skipped: %s", err),
		})
		return domain.SoftwareSourceCodeMetadata{}, false
	}
	batch := domain.ImportBatch{SoftwareSourceCodes: []domain.SoftwareSourceCodeMetadata{value}}
	if err := batch.Normalize(); err != nil {
		addImportDiagnostic(result, domain.ImportDiagnostic{
			Severity: domain.ImportSeverityError,
			Subject:  value.ID.String(),
			Message:  fmt.Sprintf("software source code skipped: %s", err),
		})
		return domain.SoftwareSourceCodeMetadata{}, false
	}
	return value, true
}

func filterContactPoints(owner uuid.UUID, predicate string, values []domain.ContactPoint, contactIDs map[uuid.UUID]uuid.UUID, result *domain.ImportResult) []domain.ContactPoint {
	filtered := make([]domain.ContactPoint, 0, len(values))
	for _, contact := range values {
		if _, ok := contactIDs[contact.ID]; !ok {
			addMissingReferenceWarning(result, owner, predicate, "contact point", contact.ID)
			continue
		}
		filtered = append(filtered, contact)
	}
	return filtered
}

func filterCategories(owner uuid.UUID, values []domain.CategoryMetadata, categoryIDs map[uuid.UUID]uuid.UUID, result *domain.ImportResult) []domain.CategoryMetadata {
	filtered := make([]domain.CategoryMetadata, 0, len(values))
	for _, category := range values {
		if _, ok := categoryIDs[category.ID]; !ok {
			addMissingReferenceWarning(result, owner, "theme", "category", category.ID)
			continue
		}
		filtered = append(filtered, category)
	}
	return filtered
}

func addMissingReferenceWarning(result *domain.ImportResult, owner uuid.UUID, predicate, kind string, id uuid.UUID) {
	addImportDiagnostic(result, domain.ImportDiagnostic{
		Severity:  domain.ImportSeverityWarning,
		Subject:   owner.String(),
		Predicate: predicate,
		Message:   fmt.Sprintf("%s %q omitted: %s is unavailable", kind, id, kind),
	})
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
		return fmt.Errorf("dataset %q has no category group: %w", value.ID, apperror.Default(apperror.CodeValidationFailed))
	}
	groupID, ok := groupIDs[value.Categories[0].GroupID]
	if !ok {
		return fmt.Errorf("dataset %q references unknown group: %w", value.ID, apperror.Default(apperror.CodeValidationFailed))
	}
	valueInput, err := toDatasetInput(value)
	if err != nil {
		return fmt.Errorf("map imported dataset %q: %w", value.ID, err)
	}
	remapDatasetReferences(&valueInput, organizationIDs, contactIDs, categoryIDs)
	if err := valueInput.Normalize(); err != nil {
		return fmt.Errorf("validate imported dataset %q: %w", value.ID, err)
	}

	_, err = q.GetImportDataset(ctx, value.ID)
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
		version, createErr := q.CreateLifecycleDatasetVersion(ctx, datasetdb.ToCreateVersionParams(logical.ID, actorID, valueInput))
		if createErr != nil {
			return fmt.Errorf("create imported dataset version %q: %w", value.ID, createErr)
		}
		if err := datasetdb.PersistChildren(ctx, q, version.ID, valueInput); err != nil {
			return fmt.Errorf("persist imported dataset metadata: %w", err)
		}
		if _, err := q.SubmitLifecycleDatasetVersion(ctx, sqlc.SubmitLifecycleDatasetVersionParams{ID: version.ID, ActorID: &actorID}); err != nil {
			return fmt.Errorf("submit imported dataset: %w", err)
		}
		if _, err := q.PublishLifecycleDatasetVersion(ctx, sqlc.PublishLifecycleDatasetVersionParams{VersionID: version.ID, ActorID: &actorID}); err != nil {
			return fmt.Errorf("publish imported dataset: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("load imported dataset %q: %w", value.ID, err)
	default:
		return fmt.Errorf("dataset %q already exists: %w", value.ID, apperror.Default(apperror.CodeConflict))
	}
}

func remapDatasetReferences(value *domain.DatasetInput, organizationIDs, contactIDs, categoryIDs map[uuid.UUID]uuid.UUID) {
	providerIDs := make([]uuid.UUID, 0, len(value.ProviderIDs))
	seenProviders := make(map[uuid.UUID]struct{}, len(value.ProviderIDs))
	for _, providerID := range value.ProviderIDs {
		providerID = organizationIDs[providerID]
		if _, ok := seenProviders[providerID]; ok {
			continue
		}
		seenProviders[providerID] = struct{}{}
		providerIDs = append(providerIDs, providerID)
	}
	value.ProviderIDs = providerIDs
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
		common := service.CommonInput()
		if common.ProviderID != nil {
			*common.ProviderID = organizationIDs[*common.ProviderID]
		}
		for j := range common.ContactPointIDs {
			common.ContactPointIDs[j] = contactIDs[common.ContactPointIDs[j]]
		}
	}
}
