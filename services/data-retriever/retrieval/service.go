// Package retrieval coordinates retrieval execution, persistence, and streaming.
package retrieval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/schema"
	"github.com/google/uuid"
)

const streamChunkSize = 64 * 1024

type Repository interface {
	CreatePlaceholder(context.Context, domain.PreparedRetrieval, domain.ResponseHead) (domain.RetrievalRecord, error)
	FinalizeRetrieval(context.Context, int64, string, int64) error
	ReplaceOutputParameters(context.Context, uuid.UUID, uuid.UUID, []domain.OutputParameter) error
	GetByID(context.Context, int64) (domain.RetrievalRecord, error)
}

type UpstreamClient interface {
	ExecuteStream(context.Context, domain.PreparedRetrieval) (*UpstreamResponse, error)
}

type StreamSink interface {
	SendDry(string) error
	SendHeaders(domain.RetrievalRecord, domain.ResponseHead) error
	SendChunk([]byte) error
}

type UpstreamResponse struct {
	Head domain.ResponseHead
	Body io.ReadCloser
}

type Service struct {
	repo         Repository
	client       UpstreamClient
	log          *slog.Logger
	schemaBudget uint64
}

func New(repo Repository, client UpstreamClient, log *slog.Logger, schemaBudget uint64) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, client: client, log: log, schemaBudget: schemaBudget}
}

func (s *Service) Get(ctx context.Context, id int64) (domain.RetrievalRecord, error) {
	if id <= 0 {
		return domain.RetrievalRecord{}, apperror.Field("id", "ID must be a positive integer.", nil)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Retrieve(ctx context.Context, input domain.RetrievalInput, sink StreamSink) error {
	prepared, err := domain.NewPreparedRetrieval(input)
	if err != nil {
		return err
	}
	if prepared.Mode == domain.RetrieveModeDry {
		if err := sink.SendDry(prepared.CurlCommand()); err != nil {
			return fmt.Errorf("send dry-run response: %w", err)
		}
		return nil
	}

	response, err := s.client.ExecuteStream(ctx, prepared)
	if err != nil {
		return upstreamError(ctx, err)
	}
	defer func() { _ = response.Body.Close() }()

	record, err := s.repo.CreatePlaceholder(ctx, prepared, response.Head)
	if err != nil {
		return apperror.Wrap(apperror.CodeInternal, "", err)
	}
	if isRedirect(response.Head) {
		if err := sink.SendHeaders(record, response.Head); err != nil {
			return s.streamError(ctx, record.ID)
		}
		size := response.Head.ContentLength
		if size < 0 {
			size = -1
		}
		return s.finalize(ctx, record.ID, "", size)
	}
	if err := sink.SendHeaders(record, response.Head); err != nil {
		return s.streamError(ctx, record.ID)
	}

	hash := sha256.New()
	var size int64
	var sample []byte
	overBudget := s.schemaBudget == 0
	buffer := make([]byte, streamChunkSize)
	for {
		count, readErr := response.Body.Read(buffer)
		if count > 0 {
			_, _ = hash.Write(buffer[:count])
			size += int64(count)
			if !overBudget {
				if uint64(size) > s.schemaBudget {
					overBudget = true
					sample = nil
				} else {
					sample = append(sample, buffer[:count]...)
				}
			}
			if err := sink.SendChunk(append([]byte(nil), buffer[:count]...)); err != nil {
				return s.streamError(ctx, record.ID)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return s.readError(ctx, record.ID, readErr)
		}
	}
	if err := s.finalize(ctx, record.ID, hex.EncodeToString(hash.Sum(nil)), size); err != nil {
		return err
	}
	s.discoverOutputSchema(ctx, prepared, response.Head.ContentType, sample, overBudget)
	return nil
}

func isRedirect(head domain.ResponseHead) bool {
	return head.StatusCode >= 300 && head.StatusCode < 400 && head.Location != ""
}

func (s *Service) finalize(ctx context.Context, id int64, payloadHash string, payloadSize int64) error {
	if err := s.repo.FinalizeRetrieval(ctx, id, payloadHash, payloadSize); err != nil {
		return apperror.Wrap(apperror.CodeInternal, "", err)
	}
	return nil
}

func (s *Service) streamError(ctx context.Context, id int64) error {
	s.markIncomplete(ctx, id)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return apperror.Default(apperror.CodeUpstreamUnavailable)
}

func (s *Service) readError(ctx context.Context, id int64, cause error) error {
	s.markIncomplete(ctx, id)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	var networkError net.Error
	if errors.As(cause, &networkError) && networkError.Timeout() {
		return apperror.Default(apperror.CodeUpstreamTimeout)
	}
	return apperror.Default(apperror.CodeUpstreamUnavailable)
}

func (s *Service) markIncomplete(ctx context.Context, id int64) {
	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.repo.FinalizeRetrieval(finalizeCtx, id, "", -1); err != nil {
		s.log.ErrorContext(finalizeCtx, "mark incomplete retrieval", "retrieval_id", id, "error", err)
	}
}

func upstreamError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return apperror.Default(apperror.CodeUpstreamTimeout)
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return apperror.Default(apperror.CodeUpstreamTimeout)
	}
	return apperror.Default(apperror.CodeUpstreamUnavailable)
}

func (s *Service) discoverOutputSchema(ctx context.Context, request domain.PreparedRetrieval, contentType string, payload []byte, overBudget bool) {
	if overBudget || len(payload) == 0 {
		return
	}
	parameters, parsed := schema.Extract(contentType, payload)
	if !parsed {
		return
	}
	if err := s.repo.ReplaceOutputParameters(ctx, request.ResourceID, request.VersionID, parameters); err != nil {
		s.log.ErrorContext(ctx, "discover output schema: replace output parameters", "error", err)
	}
}
