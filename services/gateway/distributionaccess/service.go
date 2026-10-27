// Package distributionaccess orchestrates metadata preparation and retrieval.
package distributionaccess

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

type metadataLifecycle interface {
	PrepareDistributionDownload(context.Context, string, uuid.UUID) (domain.DownloadTarget, error)
	PrepareDistributionAccess(context.Context, string, domain.AccessRequest) (domain.PreparedRequest, error)
}

type dataRetriever interface {
	RetrieveDownload(context.Context, domain.DownloadTarget) (domain.RetrievalResponse, error)
	RetrieveAccess(context.Context, domain.PreparedRequest) (domain.RetrievalResponse, error)
}

type Service struct {
	metadataLifecycle metadataLifecycle
	dataRetriever     dataRetriever
}

func NewService(metadataLifecycle metadataLifecycle, dataRetriever dataRetriever) *Service {
	return &Service{metadataLifecycle: metadataLifecycle, dataRetriever: dataRetriever}
}

func (s *Service) Prepare(ctx context.Context, actorID string, request domain.AccessRequest) (domain.PreparedRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.PreparedRequest{}, err
	}

	if request.OutputFormat != nil && request.ConversionID != nil {
		return domain.PreparedRequest{}, apperror.Field("conversion_id", "Format selectors are mutually exclusive.", nil)
	}

	// TODO: Connect conversion_id when converter and conversion-relation support exist.
	if request.ConversionID != nil {
		return domain.PreparedRequest{}, apperror.Default(apperror.CodeInternal)
	}

	target, err := s.metadataLifecycle.PrepareDistributionAccess(ctx, actorID, request)
	if err != nil {
		return domain.PreparedRequest{}, fmt.Errorf("prepare distribution access: %w", err)
	}

	return target, nil
}

func (s *Service) Access(ctx context.Context, actorID string, request domain.AccessRequest) (domain.RetrievalResponse, error) {
	target, err := s.Prepare(ctx, actorID, request)
	if err != nil {
		return domain.RetrievalResponse{}, err
	}

	return s.dataRetriever.RetrieveAccess(ctx, target)
}

func (s *Service) Download(ctx context.Context, actorID string, distributionID uuid.UUID) (domain.RetrievalResponse, error) {
	target, err := s.metadataLifecycle.PrepareDistributionDownload(ctx, actorID, distributionID)
	if err != nil {
		return domain.RetrievalResponse{}, fmt.Errorf("prepare distribution download: %w", err)
	}

	return s.dataRetriever.RetrieveDownload(ctx, target)
}
