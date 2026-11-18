package datasets

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// RequestConversion builds notifications for an editable draft web service.
func (s *Service) RequestConversion(ctx context.Context, actor domain.Actor, distributionID uuid.UUID, input domain.ConversionRequestInput) ([]domain.Notification, error) {
	if distributionID == uuid.Nil {
		return nil, apperror.Field("distribution_id", "Distribution ID must be a UUID.", nil)
	}

	datasetID, versionID, err := s.store.GetDistributionDatasetVersion(ctx, distributionID)
	if err != nil {
		return nil, operationError("resolve distribution", err)
	}

	if err := s.authorizeResource(ctx, actor, datasetID, domain.GroupPermissionResourceEdit); err != nil {
		return nil, operationError("authorize conversion request", err)
	}

	dataset, err := s.store.Get(ctx, datasetID, versionID)
	if err != nil {
		return nil, operationError("get distribution metadata", err)
	}

	if dataset.Status != domain.Draft {
		return nil, apperror.Default(apperror.CodeConflict)
	}

	for _, distribution := range dataset.Metadata.Distributions {
		if distribution.ID != distributionID {
			continue
		}

		if distribution.Type != domain.DistributionTypeWebService || distribution.WebService == nil {
			return nil, apperror.Field("distribution_id", "Distribution is not a web service.", nil)
		}

		webservice, ok := distribution.WebService.(domain.ExecutableWebService)
		if !ok {
			return nil, apperror.Field("web_service", "Web Service is not formattable", nil)
		}

		valid := false
		for _, format := range webservice.NativeFormats() {
			if format.OutputFormat == input.InputFormat {
				valid = true
				break
			}
		}
		if !valid {
			return nil, apperror.Field("input_format", "Select a native format for this distribution.", nil)
		}

		defaultURL := ""
		if distribution.WebService.ServiceType() == domain.WebServiceTypeHTTP {
			prepared, err := webservice.PrepareAccess(domain.DistributionAccessInput{
				Input: domain.HTTPAccessInput{}, OutputFormat: &input.InputFormat,
			})
			if err == nil {
				defaultURL = prepared.URL
			}
		}
		return s.notifier.Build(ctx, domain.ConversionRequested{
			ActorID: actor,
			Resource: domain.NotificationResource{
				Type:       domain.Dataset,
				ResourceID: datasetID,
				VersionID:  versionID,
				Name:       dataset.Metadata.Title,
				Status:     dataset.Status,
				Revision:   dataset.Revision,
			},
			DistributionID:    distributionID,
			DistributionTitle: distribution.Title,
			WebServiceType:    distribution.WebService.ServiceType(),
			Input:             input,
			DefaultURL:        defaultURL,
		})
	}
	return nil, apperror.Default(apperror.CodeNotFound)
}
