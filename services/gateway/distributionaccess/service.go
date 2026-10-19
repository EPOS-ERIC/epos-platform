// Package distributionaccess orchestrates metadata preparation and retrieval.
package distributionaccess

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

type metadataLifecycle interface {
	PrepareDistributionDownload(context.Context, string, uuid.UUID) (domain.DownloadTarget, error)
}

type dataRetriever interface {
	RetrieveDownload(context.Context, domain.DownloadTarget) (domain.DownloadResponse, error)
}

type Service struct {
	metadataLifecycle metadataLifecycle
	dataRetriever     dataRetriever
}

func NewService(metadataLifecycle metadataLifecycle, dataRetriever dataRetriever) *Service {
	return &Service{metadataLifecycle: metadataLifecycle, dataRetriever: dataRetriever}
}

func (s *Service) Download(ctx context.Context, actorID string, distributionID uuid.UUID) (domain.DownloadResponse, error) {
	target, err := s.metadataLifecycle.PrepareDistributionDownload(ctx, actorID, distributionID)
	if err != nil {
		return domain.DownloadResponse{}, fmt.Errorf("prepare distribution download: %w", err)
	}

	return s.dataRetriever.RetrieveDownload(ctx, target)
}
