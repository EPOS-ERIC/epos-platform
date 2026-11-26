package dataset

import (
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// ToCreateVersionParams builds SQLC parameters for a new dataset version.
func ToCreateVersionParams(datasetID uuid.UUID, actorID string, value domain.DatasetInput) sqlc.CreateLifecycleDatasetVersionParams {
	var start, end *time.Time
	if value.TemporalCoverage != nil {
		start, end = value.TemporalCoverage.Start, value.TemporalCoverage.End
	}
	var geologicalStart, geologicalEnd *float64
	if value.GeologicalTemporalCoverage != nil {
		geologicalStart, geologicalEnd = value.GeologicalTemporalCoverage.StartMYA, value.GeologicalTemporalCoverage.EndMYA
	}
	return sqlc.CreateLifecycleDatasetVersionParams{
		DatasetID:                  datasetID,
		Title:                      value.Title,
		Description:                value.Description,
		Keywords:                   value.Keywords,
		AccrualPeriodicity:         (*sqlc.DatasetAccrualPeriodicity)(value.AccrualPeriodicity),
		QualityAnnotation:          value.QualityAnnotation,
		ConformsTo:                 value.ConformsTo,
		Provenance:                 value.Provenance,
		Type:                       (*sqlc.DatasetType)(value.Type),
		Version:                    value.Version,
		SpatialResolutionInMeters:  value.SpatialResolutionInMeters,
		TemporalResolution:         value.TemporalResolution,
		AccessRight:                (*sqlc.DatasetAccessRight)(value.AccessRight),
		SourceCreatedAt:            value.SourceCreated,
		SourceUpdatedAt:            value.SourceUpdated,
		SpatialCoverage:            value.SpatialCoverage,
		TemporalStart:              start,
		TemporalEnd:                end,
		GeologicalTemporalStartMya: geologicalStart,
		GeologicalTemporalEndMya:   geologicalEnd,
		CreatedBy:                  actorID,
	}
}

// ToReplaceDraftParams builds SQLC parameters for replacing a dataset draft.
func ToReplaceDraftParams(datasetID, id uuid.UUID, revision int32, value domain.DatasetInput) sqlc.ReplaceLifecycleDatasetDraftParams {
	p := ToCreateVersionParams(uuid.Nil, "", value)
	return sqlc.ReplaceLifecycleDatasetDraftParams{
		Title:                      p.Title,
		Description:                p.Description,
		Keywords:                   p.Keywords,
		AccrualPeriodicity:         p.AccrualPeriodicity,
		QualityAnnotation:          p.QualityAnnotation,
		ConformsTo:                 p.ConformsTo,
		Provenance:                 p.Provenance,
		Type:                       p.Type,
		Version:                    p.Version,
		SpatialResolutionInMeters:  p.SpatialResolutionInMeters,
		TemporalResolution:         p.TemporalResolution,
		AccessRight:                p.AccessRight,
		SourceCreatedAt:            p.SourceCreatedAt,
		SourceUpdatedAt:            p.SourceUpdatedAt,
		SpatialCoverage:            p.SpatialCoverage,
		TemporalStart:              p.TemporalStart,
		TemporalEnd:                p.TemporalEnd,
		GeologicalTemporalStartMya: p.GeologicalTemporalStartMya,
		GeologicalTemporalEndMya:   p.GeologicalTemporalEndMya,
		DatasetID:                  datasetID,
		ID:                         id,
		Revision:                   revision,
	}
}

func toDomainDatasetVersion(version sqlc.GetLifecycleDatasetVersionRow, metadata domain.DatasetMetadata) (domain.DatasetVersion, error) {
	result := domain.DatasetVersion{
		DatasetID:     version.DatasetID,
		VersionID:     version.ID,
		Revision:      version.Revision,
		Status:        domain.Status(version.Status),
		CreatedBy:     version.CreatedBy,
		SubmittedBy:   version.SubmittedBy,
		Group:         toDomainGroup(version.Group),
		CreatedAt:     version.CreatedAt,
		UpdatedAt:     version.UpdatedAt,
		RejectedBy:    version.RejectedBy,
		RejectedAt:    version.RejectedAt,
		RejectionNote: version.RejectionNote,
		Metadata:      metadata,
	}
	if err := result.Normalize(); err != nil {
		return domain.DatasetVersion{}, fmt.Errorf("validate dataset result: %w", err)
	}
	return result, nil
}

func toDomainDatasetVersionSummary(version sqlc.ListLifecycleDatasetVersionSummariesRow) domain.ResourceVersionSummary {
	return domain.ResourceVersionSummary{
		ResourceType:  domain.Dataset,
		ResourceID:    version.DatasetID,
		VersionID:     version.ID,
		Revision:      version.Revision,
		Status:        domain.Status(version.Status),
		Name:          version.Title,
		CreatedBy:     version.CreatedBy,
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

func temporal(start, end *time.Time) *domain.Temporal {
	if start == nil && end == nil {
		return nil
	}
	return &domain.Temporal{Start: start, End: end}
}

func geologicalTemporal(start, end *float64) *domain.GeologicalTemporal {
	if start == nil && end == nil {
		return nil
	}
	return &domain.GeologicalTemporal{StartMYA: start, EndMYA: end}
}

func toDomainWebService(value sqlc.WebService) (domain.WebService, error) {
	common := domain.WebServiceCommon{
		ID:               value.ID,
		ResourceID:       value.ResourceID,
		VisibleOn:        domainVisibleOnValues(value.VisibleOn),
		DocumentationURI: value.DocumentationUri,
		Keywords:         value.Keywords,
	}
	switch domain.WebServiceType(value.Type) {
	case domain.WebServiceTypeHTTP:
		return &domain.HTTPWebService{WebServiceCommon: common}, nil
	case domain.WebServiceTypeDynamic:
		return &domain.DynamicWebService{WebServiceCommon: common}, nil
	case domain.WebServiceTypeWFS:
		return &domain.WFSWebService{WebServiceCommon: common}, nil
	case domain.WebServiceTypeWMS:
		return &domain.WMSWebService{WebServiceCommon: common}, nil
	case domain.WebServiceTypeWMTS:
		return &domain.WMTSWebService{WebServiceCommon: common}, nil
	default:
		return nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("unsupported web service type %q", value.Type))
	}
}

func genericWebService(value domain.WebService) *domain.GenericWebService {
	switch service := value.(type) {
	case *domain.HTTPWebService:
		return &service.GenericWebService
	case *domain.DynamicWebService:
		return &service.GenericWebService
	default:
		return nil
	}
}

func genericWebServiceInput(value domain.WebServiceInput) *domain.GenericWebServiceInput {
	switch service := value.(type) {
	case *domain.HTTPWebServiceInput:
		return &service.GenericWebServiceInput
	case *domain.DynamicWebServiceInput:
		return &service.GenericWebServiceInput
	default:
		return nil
	}
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
		ResourceID:       value.ResourceID,
		Title:            value.Title,
		Description:      value.Description,
		License:          value.License,
		Type:             domain.DistributionType(value.Type),
		ByteSize:         value.ByteSize,
		WebService:       nil,
		DownloadableFile: nil,
	}
	if value.WebServiceID != nil {
		result.WebService = services[*value.WebServiceID]
		return result
	}
	result.DownloadableFile = &domain.DownloadableFile{
		URL:    *value.DownloadableFileUrl,
		Format: *value.DownloadableFileFormat,
	}
	return result
}

func toDomainOrganizationMetadata(value sqlc.Organization) domain.OrganizationMetadata {
	return domain.OrganizationMetadata{
		ID:          value.ID,
		Name:        value.Name,
		Street:      value.Street,
		Locality:    value.Locality,
		PostalCode:  value.PostalCode,
		Country:     value.Country,
		LogoURL:     value.LogoUrl,
		PageURL:     value.PageUrl,
		Email:       value.Email,
		Identifiers: nil,
	}
}

func toDomainContactPoint(value sqlc.ContactPoint) domain.ContactPoint {
	return domain.ContactPoint{
		ID:    value.ID,
		Email: value.Email,
	}
}

func toDomainIdentifier(value sqlc.Identifier) domain.Identifier {
	return domain.Identifier{
		ID:         value.ID,
		ResourceID: value.ResourceID,
		Type:       domain.IdentifierType(value.Type),
		Value:      value.Value,
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
		ConceptID:   value.ConceptID,
	}
}
