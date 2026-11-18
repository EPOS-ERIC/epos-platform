package softwaresourcecodes

import (
	"context"
	"fmt"

	resourcedb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/resource"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// ToCreateVersionParams builds SQLC parameters for a new source-code version.
func ToCreateVersionParams(sourceCodeID uuid.UUID, actorID string, value domain.SoftwareSourceCodeInput) sqlc.CreateLifecycleSoftwareSourceCodeVersionParams {
	return sqlc.CreateLifecycleSoftwareSourceCodeVersionParams{
		SoftwareSourceCodeID: sourceCodeID,
		Name:                 value.Name,
		Description:          value.Description,
		ProgrammingLanguage:  value.ProgrammingLanguage,
		Keywords:             value.Keywords,
		Repository:           value.Repository,
		License:              value.License,
		SoftwareRequirements: value.SoftwareRequirements,
		RuntimePlatform:      value.RuntimePlatform,
		MainEntityOfPage:     value.MainEntityOfPage,
		Version:              value.Version,
		CreatedBy:            actorID,
	}
}

// ToReplaceDraftParams builds SQLC parameters for replacing a source-code draft.
func ToReplaceDraftParams(sourceCodeID, versionID uuid.UUID, revision int32, value domain.SoftwareSourceCodeInput) sqlc.ReplaceLifecycleSoftwareSourceCodeDraftParams {
	return sqlc.ReplaceLifecycleSoftwareSourceCodeDraftParams{
		Name:                 value.Name,
		Description:          value.Description,
		ProgrammingLanguage:  value.ProgrammingLanguage,
		Keywords:             value.Keywords,
		Repository:           value.Repository,
		License:              value.License,
		SoftwareRequirements: value.SoftwareRequirements,
		RuntimePlatform:      value.RuntimePlatform,
		MainEntityOfPage:     value.MainEntityOfPage,
		Version:              value.Version,
		ID:                   versionID,
		SoftwareSourceCodeID: sourceCodeID,
		Revision:             revision,
	}
}

// PersistChildren persists source-code references for one version.
func PersistChildren(ctx context.Context, q *sqlc.Queries, versionID uuid.UUID, value domain.SoftwareSourceCodeInput) error {
	return persistChildren(ctx, q, versionID, value, nil)
}

func persistChildren(ctx context.Context, q *sqlc.Queries, versionID uuid.UUID, value domain.SoftwareSourceCodeInput, identities map[uuid.UUID]domain.Identifier) error {
	for _, identifier := range value.Identifiers {
		id, resourceID := resourcedb.IdentifierIDs(identifier, identities)
		if err := q.AddLifecycleSoftwareSourceCodeIdentifier(ctx, sqlc.AddLifecycleSoftwareSourceCodeIdentifierParams{
			ID:                          id,
			ResourceID:                  resourceID,
			Type:                        string(identifier.Type),
			Value:                       identifier.Value,
			SoftwareSourceCodeVersionID: &versionID,
		}); err != nil {
			return fmt.Errorf("add source code identifier: %w", err)
		}
	}
	for _, creatorID := range value.CreatorIDs {
		if err := q.AddLifecycleSoftwareSourceCodeCreator(ctx, sqlc.AddLifecycleSoftwareSourceCodeCreatorParams{
			SoftwareSourceCodeVersionID: versionID,
			OrganizationID:              creatorID,
		}); err != nil {
			return fmt.Errorf("add source code creator: %w", err)
		}
	}
	for _, contactPointID := range value.ContactPointIDs {
		if err := q.AddLifecycleSoftwareSourceCodeContactPoint(ctx, sqlc.AddLifecycleSoftwareSourceCodeContactPointParams{
			SoftwareSourceCodeVersionID: versionID,
			ContactPointID:              contactPointID,
		}); err != nil {
			return fmt.Errorf("add source code contact point: %w", err)
		}
	}
	for _, categoryID := range value.CategoryIDs {
		if err := q.AddLifecycleSoftwareSourceCodeCategory(ctx, sqlc.AddLifecycleSoftwareSourceCodeCategoryParams{
			SoftwareSourceCodeVersionID: versionID,
			CategoryID:                  categoryID,
		}); err != nil {
			return fmt.Errorf("add source code category: %w", err)
		}
	}
	return nil
}

func sourceCodeResult(ctx context.Context, q *sqlc.Queries, sourceCodeID, versionID uuid.UUID) (domain.SoftwareSourceCodeVersion, error) {
	row, err := q.GetLifecycleSoftwareSourceCodeVersion(ctx, sqlc.GetLifecycleSoftwareSourceCodeVersionParams{
		VersionID:            versionID,
		SoftwareSourceCodeID: sourceCodeID,
	})
	if err != nil {
		return domain.SoftwareSourceCodeVersion{}, fmt.Errorf("get source code version: %w", err)
	}
	metadata, err := sourceCodeMetadata(ctx, q, row)
	if err != nil {
		return domain.SoftwareSourceCodeVersion{}, err
	}

	version := row.SoftwareSourceCodeVersion
	result := domain.SoftwareSourceCodeVersion{
		SoftwareSourceCodeID: version.SoftwareSourceCodeID,
		VersionID:            version.ID,
		Revision:             version.Revision,
		Status:               domain.Status(version.Status),
		CreatedBy:            domain.Actor(version.CreatedBy),
		SubmittedBy:          (*domain.Actor)(version.SubmittedBy),
		Group:                resourcedb.ToGroup(row.Group),
		CreatedAt:            version.CreatedAt,
		UpdatedAt:            version.UpdatedAt,
		RejectedBy:           version.RejectedBy,
		RejectedAt:           version.RejectedAt,
		RejectionNote:        version.RejectionNote,
		Metadata:             metadata,
	}
	if err := result.Normalize(); err != nil {
		return domain.SoftwareSourceCodeVersion{}, fmt.Errorf("validate source code result: %w", err)
	}
	return result, nil
}

func sourceCodeMetadata(ctx context.Context, q *sqlc.Queries, row sqlc.GetLifecycleSoftwareSourceCodeVersionRow) (domain.SoftwareSourceCodeMetadata, error) {
	version := row.SoftwareSourceCodeVersion
	identifiers, err := q.ListLifecycleSoftwareSourceCodeIdentifiers(ctx, &version.ID)
	if err != nil {
		return domain.SoftwareSourceCodeMetadata{}, fmt.Errorf("list source code identifiers: %w", err)
	}
	creators, err := q.ListLifecycleSoftwareSourceCodeCreators(ctx, version.ID)
	if err != nil {
		return domain.SoftwareSourceCodeMetadata{}, fmt.Errorf("list source code creators: %w", err)
	}
	contacts, err := q.ListLifecycleSoftwareSourceCodeContactPoints(ctx, version.ID)
	if err != nil {
		return domain.SoftwareSourceCodeMetadata{}, fmt.Errorf("list source code contact points: %w", err)
	}
	categories, err := q.ListLifecycleSoftwareSourceCodeCategories(ctx, version.ID)
	if err != nil {
		return domain.SoftwareSourceCodeMetadata{}, fmt.Errorf("list source code categories: %w", err)
	}

	result := domain.SoftwareSourceCodeMetadata{
		ID:                   version.ID,
		Name:                 version.Name,
		Description:          version.Description,
		ProgrammingLanguage:  version.ProgrammingLanguage,
		Keywords:             version.Keywords,
		Repository:           version.Repository,
		License:              version.License,
		SoftwareRequirements: version.SoftwareRequirements,
		RuntimePlatform:      version.RuntimePlatform,
		MainEntityOfPage:     version.MainEntityOfPage,
		Version:              version.Version,
		Creators:             nil,
		Identifiers:          nil,
		ContactPoints:        nil,
		Categories:           nil,
	}
	for _, item := range identifiers {
		result.Identifiers = append(result.Identifiers, resourcedb.ToIdentifier(item.Identifier))
	}
	for _, item := range creators {
		result.Creators = append(result.Creators, resourcedb.ToOrganizationMetadata(item))
	}
	for _, item := range contacts {
		result.ContactPoints = append(result.ContactPoints, resourcedb.ToContactPoint(item.ContactPoint))
	}
	for _, item := range categories {
		result.Categories = append(result.Categories, resourcedb.ToCategoryMetadata(item.Category.ID, item.Category.Name, item.Group))
	}
	return result, nil
}

func toDomainSoftwareSourceCodeVersionSummary(version sqlc.ListLifecycleSoftwareSourceCodeVersionSummariesRow) domain.ResourceVersionSummary {
	return domain.ResourceVersionSummary{
		ResourceType:  domain.SoftwareSourceCode,
		ResourceID:    version.SoftwareSourceCodeID,
		VersionID:     version.ID,
		Revision:      version.Revision,
		Status:        domain.Status(version.Status),
		Name:          version.Name,
		CreatedBy:     domain.Actor(version.CreatedBy),
		CreatedAt:     version.CreatedAt,
		UpdatedAt:     version.UpdatedAt,
		SubmittedBy:   version.SubmittedBy,
		SubmittedAt:   version.SubmittedAt,
		PublishedBy:   version.PublishedBy,
		PublishedAt:   version.PublishedAt,
		ArchivedBy:    version.ArchivedBy,
		ArchivedAt:    version.ArchivedAt,
		RejectedBy:    version.RejectedBy,
		RejectedAt:    version.RejectedAt,
		RejectionNote: version.RejectionNote,
	}
}
