// Package distributionaccess orchestrates metadata preparation and retrieval.
package distributionaccess

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

type metadataLifecycle interface {
	CheckDistributionRead(context.Context, string, uuid.UUID) error
	PrepareDistributionDownload(context.Context, string, uuid.UUID) (domain.DownloadTarget, error)
	PrepareDistributionAccess(context.Context, string, domain.AccessRequest) (domain.PreparedRequest, error)
}

type dataRetriever interface {
	RetrieveDownload(context.Context, domain.DownloadTarget) (domain.RetrievalResponse, error)
	RetrieveAccess(context.Context, domain.PreparedRequest) (domain.RetrievalResponse, error)
}

type converter interface {
	DistributionRelations(context.Context, string, uuid.UUID) ([]domain.DistributionPluginRelation, error)
	Conversion(context.Context, uuid.UUID) (domain.ConversionRelation, error)
	Execute(context.Context, uuid.UUID, uuid.UUID, string, io.Reader) (io.ReadCloser, error)
}

func (s *Service) DistributionRelations(ctx context.Context, actorID string, id uuid.UUID) ([]domain.DistributionPluginRelation, error) {
	if err := s.metadataLifecycle.CheckDistributionRead(ctx, actorID, id); err != nil {
		return nil, fmt.Errorf("authorize distribution read: %w", err)
	}

	return s.converter.DistributionRelations(ctx, actorID, id)
}

type Service struct {
	metadataLifecycle metadataLifecycle
	dataRetriever     dataRetriever
	converter         converter
}

func NewService(metadataLifecycle metadataLifecycle, dataRetriever dataRetriever, converter converter) *Service {
	return &Service{metadataLifecycle: metadataLifecycle, dataRetriever: dataRetriever, converter: converter}
}

func (s *Service) Prepare(ctx context.Context, actorID string, request domain.AccessRequest) (domain.PreparedRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.PreparedRequest{}, err
	}

	if request.OutputFormat != nil && request.ConversionID != nil {
		return domain.PreparedRequest{}, apperror.Field("conversion_id", "Format selectors are mutually exclusive.", nil)
	}

	if request.ConversionID != nil {
		return domain.PreparedRequest{}, apperror.Field("conversion_id", "Conversion is not supported when preparing a request.", nil)
	}

	target, err := s.metadataLifecycle.PrepareDistributionAccess(ctx, actorID, request)
	if err != nil {
		return domain.PreparedRequest{}, fmt.Errorf("prepare distribution access: %w", err)
	}

	return target, nil
}

func (s *Service) Access(ctx context.Context, actorID string, request domain.AccessRequest) (domain.RetrievalResponse, error) {
	if request.ConversionID != nil {
		if request.OutputFormat != nil {
			return domain.RetrievalResponse{}, apperror.Field("conversion_id", "Format selectors are mutually exclusive.", nil)
		}

		relation, err := s.converter.Conversion(ctx, *request.ConversionID)
		if err != nil {
			return domain.RetrievalResponse{}, fmt.Errorf("resolve conversion: %w", err)
		}

		if relation.DistributionID != request.DistributionID {
			return domain.RetrievalResponse{}, apperror.Default(apperror.CodeNotFound)
		}

		request.OutputFormat = &relation.InputFormat
		request.ConversionID = nil

		target, err := s.Prepare(ctx, actorID, request)
		if err != nil {
			return domain.RetrievalResponse{}, err
		}

		response, err := s.dataRetriever.RetrieveAccess(ctx, target)
		if err != nil {
			return domain.RetrievalResponse{}, err
		}

		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return response, nil
		}

		if response.Body == nil {
			return domain.RetrievalResponse{}, apperror.Default(apperror.CodeInternal)
		}

		output, conversionErr := s.converter.Execute(ctx, relation.ID, relation.DistributionID, target.URL, response.Body)
		closeErr := response.Body.Close()
		if conversionErr != nil {
			return domain.RetrievalResponse{}, fmt.Errorf("execute conversion: %w", errors.Join(conversionErr, closeErr))
		}

		if closeErr != nil {
			if err := output.Close(); err != nil {
				return domain.RetrievalResponse{}, fmt.Errorf("close retrieved and converted payloads: %w", errors.Join(closeErr, err))
			}

			return domain.RetrievalResponse{}, fmt.Errorf("close retrieved payload: %w", closeErr)
		}

		return domain.RetrievalResponse{StatusCode: 200, Body: output}, nil
	}

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
