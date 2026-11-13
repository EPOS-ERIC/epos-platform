package softwareapplications

import (
	"context"
	"fmt"

	resourcedb "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/resource"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// ToCreateVersionParams builds SQLC parameters for a new application version.
func ToCreateVersionParams(applicationID uuid.UUID, actorID string, value domain.SoftwareApplicationInput) sqlc.CreateLifecycleSoftwareApplicationVersionParams {
	return sqlc.CreateLifecycleSoftwareApplicationVersionParams{
		SoftwareApplicationID: applicationID,
		Name:                  value.Name,
		Description:           value.Description,
		MainEntityOfPage:      value.MainEntityOfPage,
		License:               value.License,
		Version:               value.Version,
		Keywords:              value.Keywords,
		DownloadUrl:           value.DownloadURL,
		Requirements:          value.Requirements,
		OperatingSystem:       value.OperatingSystem,
		CreatedBy:             actorID,
	}
}

// ToReplaceDraftParams builds SQLC parameters for replacing an application draft.
func ToReplaceDraftParams(applicationID, versionID uuid.UUID, revision int32, value domain.SoftwareApplicationInput) sqlc.ReplaceLifecycleSoftwareApplicationDraftParams {
	return sqlc.ReplaceLifecycleSoftwareApplicationDraftParams{
		Name:                  value.Name,
		Description:           value.Description,
		MainEntityOfPage:      value.MainEntityOfPage,
		License:               value.License,
		Version:               value.Version,
		Keywords:              value.Keywords,
		DownloadUrl:           value.DownloadURL,
		Requirements:          value.Requirements,
		OperatingSystem:       value.OperatingSystem,
		ID:                    versionID,
		SoftwareApplicationID: applicationID,
		Revision:              revision,
	}
}

// PersistChildren persists application references for one version.
func PersistChildren(ctx context.Context, q *sqlc.Queries, versionID uuid.UUID, value domain.SoftwareApplicationInput) error {
	return persistChildren(ctx, q, versionID, value, nil)
}

func persistChildren(ctx context.Context, q *sqlc.Queries, versionID uuid.UUID, value domain.SoftwareApplicationInput, identities map[uuid.UUID]domain.Identifier) error {
	for _, identifier := range value.Identifiers {
		id, resourceID := resourcedb.IdentifierIDs(identifier, identities)
		if err := q.AddLifecycleSoftwareApplicationIdentifier(ctx, sqlc.AddLifecycleSoftwareApplicationIdentifierParams{
			ID:                           id,
			ResourceID:                   resourceID,
			Type:                         string(identifier.Type),
			Value:                        identifier.Value,
			SoftwareApplicationVersionID: &versionID,
		}); err != nil {
			return fmt.Errorf("add application identifier: %w", err)
		}
	}
	for _, creatorID := range value.CreatorIDs {
		if err := q.AddLifecycleSoftwareApplicationCreator(ctx, sqlc.AddLifecycleSoftwareApplicationCreatorParams{
			SoftwareApplicationVersionID: versionID,
			OrganizationID:               creatorID,
		}); err != nil {
			return fmt.Errorf("add application creator: %w", err)
		}
	}
	for _, contactPointID := range value.ContactPointIDs {
		if err := q.AddLifecycleSoftwareApplicationContactPoint(ctx, sqlc.AddLifecycleSoftwareApplicationContactPointParams{
			SoftwareApplicationVersionID: versionID,
			ContactPointID:               contactPointID,
		}); err != nil {
			return fmt.Errorf("add application contact point: %w", err)
		}
	}
	for _, categoryID := range value.CategoryIDs {
		if err := q.AddLifecycleSoftwareApplicationCategory(ctx, sqlc.AddLifecycleSoftwareApplicationCategoryParams{
			SoftwareApplicationVersionID: versionID,
			CategoryID:                   categoryID,
		}); err != nil {
			return fmt.Errorf("add application category: %w", err)
		}
	}
	return nil
}

func applicationResult(ctx context.Context, q *sqlc.Queries, applicationID, versionID uuid.UUID) (domain.SoftwareApplicationVersion, error) {
	row, err := q.GetLifecycleSoftwareApplicationVersion(ctx, sqlc.GetLifecycleSoftwareApplicationVersionParams{
		VersionID:             versionID,
		SoftwareApplicationID: applicationID,
	})
	if err != nil {
		return domain.SoftwareApplicationVersion{}, fmt.Errorf("get application version: %w", err)
	}
	metadata, err := applicationMetadata(ctx, q, row)
	if err != nil {
		return domain.SoftwareApplicationVersion{}, err
	}

	version := row.SoftwareApplicationVersion
	result, err := domain.NewSoftwareApplicationVersion(
		version.SoftwareApplicationID,
		version.ID,
		version.Revision,
		domain.Status(version.Status),
		domain.Actor(version.CreatedBy),
		version.SubmittedBy,
		resourcedb.ToGroup(row.Group),
		version.CreatedAt,
		version.UpdatedAt,
		version.RejectedBy,
		version.RejectedAt,
		version.RejectionNote,
		metadata,
	)
	if err != nil {
		return domain.SoftwareApplicationVersion{}, fmt.Errorf("validate application result: %w", err)
	}
	return result, nil
}

func applicationMetadata(ctx context.Context, q *sqlc.Queries, row sqlc.GetLifecycleSoftwareApplicationVersionRow) (domain.SoftwareApplicationMetadata, error) {
	version := row.SoftwareApplicationVersion
	identifiers, err := q.ListLifecycleSoftwareApplicationIdentifiers(ctx, &version.ID)
	if err != nil {
		return domain.SoftwareApplicationMetadata{}, fmt.Errorf("list application identifiers: %w", err)
	}
	creators, err := q.ListLifecycleSoftwareApplicationCreators(ctx, version.ID)
	if err != nil {
		return domain.SoftwareApplicationMetadata{}, fmt.Errorf("list application creators: %w", err)
	}
	contacts, err := q.ListLifecycleSoftwareApplicationContactPoints(ctx, version.ID)
	if err != nil {
		return domain.SoftwareApplicationMetadata{}, fmt.Errorf("list application contact points: %w", err)
	}
	categories, err := q.ListLifecycleSoftwareApplicationCategories(ctx, version.ID)
	if err != nil {
		return domain.SoftwareApplicationMetadata{}, fmt.Errorf("list application categories: %w", err)
	}

	result := domain.SoftwareApplicationMetadata{
		ID:               version.ID,
		Name:             version.Name,
		Description:      version.Description,
		MainEntityOfPage: version.MainEntityOfPage,
		License:          version.License,
		Version:          version.Version,
		Keywords:         version.Keywords,
		DownloadURL:      version.DownloadUrl,
		Requirements:     version.Requirements,
		OperatingSystem:  version.OperatingSystem,
		Creators:         nil,
		Identifiers:      nil,
		ContactPoints:    nil,
		Categories:       nil,
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

func toDomainSoftwareApplicationVersionSummary(version sqlc.ListLifecycleSoftwareApplicationVersionSummariesRow) domain.ResourceVersionSummary {
	return domain.ResourceVersionSummary{
		ResourceType:  domain.SoftwareApplication,
		ResourceID:    version.SoftwareApplicationID,
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
