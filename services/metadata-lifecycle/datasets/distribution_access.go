package datasets

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func (s *Service) PrepareAccess(ctx context.Context, actor *domain.Actor, distributionID uuid.UUID, input domain.DistributionAccessInput) (domain.DistributionAccess, error) {
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

	prepared, err := distribution.WebService.PrepareAccess(input)
	if err != nil {
		return domain.DistributionAccess{}, err
	}

	return domain.DistributionAccess{
		ResourceID:          datasetID,
		VersionID:           versionID,
		PreparedHTTPRequest: prepared,
	}, nil
}

func (s *Service) readDistribution(ctx context.Context, actor *domain.Actor, distributionID uuid.UUID) (uuid.UUID, uuid.UUID, domain.Distribution, error) {
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
