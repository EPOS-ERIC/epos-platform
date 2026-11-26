package datasets

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

// CheckDistributionRead verifies access without preparing a distribution request.
func (s *Service) CheckDistributionRead(ctx context.Context, actor string, distributionID uuid.UUID) error {
	_, _, _, err := s.readDistribution(ctx, &actor, distributionID)
	return err
}

// DistributionFormats returns native formats to authorized converter managers.
func (s *Service) DistributionFormats(ctx context.Context, actor string, distributionID uuid.UUID) ([]string, error) {
	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actor, domain.GlobalPermissionConverterManage)
	if err != nil {
		return nil, operationError("authorize converter management", err)
	}

	if !allowed.Allowed {
		return nil, apperror.Default(apperror.CodePermissionDenied)
	}

	datasetID, versionID, err := s.store.GetDistributionDatasetVersion(ctx, distributionID)
	if err != nil {
		return nil, operationError("resolve distribution", err)
	}

	dataset, err := s.store.Get(ctx, datasetID, versionID)
	if err != nil {
		return nil, operationError("get distribution metadata", err)
	}

	for _, distribution := range dataset.Metadata.Distributions {
		if distribution.ID != distributionID {
			continue
		}

		if distribution.Type != domain.DistributionTypeWebService || distribution.WebService == nil {
			return nil, apperror.Field("distribution_id", "Distribution is not a web service.", nil)
		}

		executable, ok := distribution.WebService.(domain.ExecutableWebService)
		if !ok {
			return []string{}, nil
		}
		formats := executable.NativeFormats()

		result := make([]string, len(formats))
		for i, format := range formats {
			result[i] = format.OutputFormat
		}

		return result, nil
	}

	return nil, apperror.Default(apperror.CodeNotFound)
}

func (s *Service) PrepareAccess(ctx context.Context, actor *string, distributionID uuid.UUID, input domain.DistributionAccessInput) (domain.DistributionAccess, error) {
	if err := ctx.Err(); err != nil {
		return domain.DistributionAccess{}, err
	}

	datasetID, versionID, distribution, err := s.readDistribution(ctx, actor, distributionID)
	if err != nil {
		return domain.DistributionAccess{}, err
	}

	if distribution.Type != domain.DistributionTypeWebService || distribution.WebService == nil {
		return domain.DistributionAccess{}, apperror.Field("distribution_id", "Distribution is not a web service.", nil)
	}

	executable, ok := distribution.WebService.(domain.ExecutableWebService)
	if !ok {
		return domain.DistributionAccess{}, apperror.Field("type", "Prepared HTTP access is not supported for this service configuration.", nil)
	}

	prepared, err := executable.PrepareAccess(input)
	if err != nil {
		return domain.DistributionAccess{}, err
	}

	return domain.DistributionAccess{
		ResourceID:          datasetID,
		VersionID:           versionID,
		PreparedHTTPRequest: prepared,
	}, nil
}

func (s *Service) readDistribution(ctx context.Context, actor *string, distributionID uuid.UUID) (uuid.UUID, uuid.UUID, domain.Distribution, error) {
	if distributionID == uuid.Nil {
		return uuid.Nil, uuid.Nil, domain.Distribution{}, apperror.Field("distribution_id", "Distribution ID must be a UUID.", nil)
	}

	datasetID, versionID, err := s.store.GetDistributionDatasetVersion(ctx, distributionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, domain.Distribution{}, operationError("resolve distribution", err)
	}

	dataset, err := s.Read(ctx, actor, datasetID, versionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, domain.Distribution{}, err
	}

	for _, distribution := range dataset.Metadata.Distributions {
		if distribution.ID == distributionID {
			return datasetID, versionID, distribution, nil
		}
	}

	return uuid.Nil, uuid.Nil, domain.Distribution{}, apperror.Default(apperror.CodeNotFound)
}
