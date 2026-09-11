package dataset

import (
	"fmt"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// ToCreateVersionParams builds SQLC parameters for a new dataset version.
func ToCreateVersionParams(datasetID uuid.UUID, actorID string, value domain.DatasetInput) (sqlc.CreateLifecycleDatasetVersionParams, error) {
	var start, end *time.Time
	if value.TemporalCoverage != nil {
		start, end = value.TemporalCoverage.Start, value.TemporalCoverage.End
	}
	return sqlc.CreateLifecycleDatasetVersionParams{
		DatasetID:          datasetID,
		Title:              value.Title,
		Description:        value.Description,
		Keywords:           strings.Join(value.Keywords, ","),
		AccrualPeriodicity: value.AccrualPeriodicity,
		QualityAnnotation:  value.QualityAnnotation,
		AccessRight:        value.AccessRight,
		SourceCreatedAt:    value.SourceCreated,
		SourceUpdatedAt:    value.SourceUpdated,
		SpatialCoverage:    value.SpatialCoverage,
		TemporalStart:      start,
		TemporalEnd:        end,
		CreatedBy:          actorID,
	}, nil
}

// ToReplaceDraftParams builds SQLC parameters for replacing a dataset draft.
func ToReplaceDraftParams(datasetID, id uuid.UUID, revision int32, value domain.DatasetInput) (sqlc.ReplaceLifecycleDatasetDraftParams, error) {
	p, err := ToCreateVersionParams(uuid.Nil, "", value)
	if err != nil {
		return sqlc.ReplaceLifecycleDatasetDraftParams{}, err
	}
	return sqlc.ReplaceLifecycleDatasetDraftParams{
		Title:              p.Title,
		Description:        p.Description,
		Keywords:           p.Keywords,
		AccrualPeriodicity: p.AccrualPeriodicity,
		QualityAnnotation:  p.QualityAnnotation,
		AccessRight:        p.AccessRight,
		SourceCreatedAt:    p.SourceCreatedAt,
		SourceUpdatedAt:    p.SourceUpdatedAt,
		SpatialCoverage:    p.SpatialCoverage,
		TemporalStart:      p.TemporalStart,
		TemporalEnd:        p.TemporalEnd,
		DatasetID:          datasetID,
		ID:                 id,
		Revision:           revision,
	}, nil
}

func toDomainDatasetVersion(version sqlc.GetLifecycleDatasetVersionRow, metadata domain.DatasetMetadata) (domain.DatasetVersion, error) {
	result, err := domain.NewDatasetVersion(
		version.DatasetID,
		version.ID,
		version.Revision,
		domain.Status(version.Status),
		domain.Actor(version.CreatedBy),
		toDomainGroup(version.Group),
		version.CreatedAt,
		version.UpdatedAt,
		version.RejectionNote,
		metadata,
	)
	if err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("validate dataset result: %w", err)
	}
	return result, nil
}

func temporal(start, end *time.Time) *domain.Temporal {
	if start == nil && end == nil {
		return nil
	}
	return &domain.Temporal{Start: start, End: end}
}

func toDomainWebService(value sqlc.WebService) (domain.WebService, error) {
	var documentation *domain.Documentation
	if value.DocumentationUri != nil {
		documentation = &domain.Documentation{
			Title:       value.DocumentationTitle,
			Description: value.DocumentationDescription,
			URI:         *value.DocumentationUri,
		}
	}
	return domain.WebService{
		ID:               value.ID,
		Name:             value.Name,
		Description:      value.Description,
		TemplateURL:      value.TemplateUrl,
		Method:           domain.HTTPMethod(value.Method),
		Type:             domain.WebServiceType(value.Type),
		VisibleOn:        domainVisibleOnValues(value.VisibleOn),
		Body:             nil,
		Parameters:       nil,
		Format:           value.Format,
		Provider:         nil,
		Documentation:    documentation,
		SpatialCoverage:  value.SpatialCoverage,
		TemporalCoverage: temporal(value.TemporalStart, value.TemporalEnd),
		ContactPoints:    nil,
	}, nil
}

func domainVisibleOnValues(values []sqlc.WebServiceVisibleOn) []domain.WebServiceVisibleOn {
	if values == nil {
		return nil
	}
	result := make([]domain.WebServiceVisibleOn, len(values))
	for i, value := range values {
		result[i] = domain.WebServiceVisibleOn(value)
	}
	return result
}

func toDomainDistribution(value sqlc.Distribution, services map[uuid.UUID]domain.WebService) domain.Distribution {
	result := domain.Distribution{
		ID:               value.ID,
		Title:            value.Title,
		Description:      value.Description,
		License:          value.License,
		Type:             domain.DistributionType(value.Type),
		Issued:           value.SourceIssuedAt,
		Modified:         value.SourceModifiedAt,
		ByteSize:         nil,
		WebService:       nil,
		DownloadableFile: nil,
	}
	if value.ByteSize != nil {
		byteSize := int64(*value.ByteSize)
		result.ByteSize = &byteSize
	}
	if value.WebServiceID != nil {
		service := services[*value.WebServiceID]
		result.WebService = &service
		return result
	}
	result.DownloadableFile = &domain.DownloadableFile{
		URL:    *value.DownloadableFileUrl,
		Format: *value.DownloadableFileFormat,
	}
	return result
}

func toDomainCategoryMetadata(value sqlc.ListLifecycleDatasetCategoriesRow) domain.CategoryMetadata {
	return domain.CategoryMetadata{
		ID:   value.ID,
		Name: value.Name,
		Scheme: domain.Group{
			ID:          uuid.Nil,
			Name:        "",
			Description: nil,
			Label:       nil,
			Logo:        nil,
			Page:        nil,
			Color:       nil,
			Position:    0,
			Concept:     "",
		},
		Children: nil,
	}
}

func toDomainOrganizationMetadata(value sqlc.Organization) domain.OrganizationMetadata {
	return domain.OrganizationMetadata{
		ID:            value.ID,
		Name:          value.Name,
		Street:        value.Street,
		Locality:      value.Locality,
		PostalCode:    value.PostalCode,
		Country:       value.Country,
		LogoURL:       value.LogoUrl,
		PageURL:       value.PageUrl,
		Email:         value.Email,
		LEICode:       value.LeiCode,
		ContactPoints: nil,
		Identifiers:   nil,
	}
}

func toDomainContactPoint(value sqlc.ContactPoint) domain.ContactPoint {
	return domain.ContactPoint{
		ID:    value.ID,
		Email: value.Email,
		Role:  value.Role,
	}
}

func toDomainIdentifier(value sqlc.Identifier) domain.Identifier {
	return domain.Identifier{
		ID:    value.ID,
		Type:  domain.IdentifierType(value.Type),
		Value: value.Value,
	}
}

func toDomainGroup(value sqlc.Group) domain.Group {
	return domain.Group{
		ID:          value.ID,
		Name:        value.Name,
		Description: value.Description,
		Label:       value.Label,
		Logo:        value.Logo,
		Page:        value.Page,
		Color:       value.Color,
		Position:    value.Position,
		Concept:     string(value.Concept),
	}
}
