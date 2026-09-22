package grpcapi

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"

	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/schema"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// streamChunkSize bounds memory per streamed chunk regardless of payload size.
const streamChunkSize = 64 * 1024

type retrievalClient interface {
	ExecuteStream(context.Context, domain.ValidatedRequest) (*retriever.UpstreamStream, error)
}

type Server struct {
	dataretrieverv1.UnimplementedDataRetrieverServiceServer
	repo   repository.RetrievalRepository
	client retrievalClient
	log    *slog.Logger
	// schemaBudget bounds how many leading payload bytes are buffered for
	// output schema extraction. Larger payloads stream normally but their
	// schema is not extracted. Zero disables extraction.
	schemaBudget uint64
}

func New(repo repository.RetrievalRepository, client retrievalClient, log *slog.Logger, schemaBudget uint64) *Server {
	return &Server{repo: repo, client: client, log: log, schemaBudget: schemaBudget}
}

func (s *Server) RetrieveStream(request *dataretrieverv1.RetrieveStreamRequest, stream dataretrieverv1.DataRetrieverService_RetrieveStreamServer) error {
	retrieveRequest, err := toDomainRetrieveRequest(request)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	validated, err := retrieveRequest.Validate()
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if validated.Mode == domain.RetrieveModeDry {
		return stream.Send(&dataretrieverv1.RetrieveStreamResponse{
			Event: &dataretrieverv1.RetrieveStreamResponse_Headers{
				Headers: &dataretrieverv1.RetrieveStreamHeaders{
					CurlCommand: validated.CurlCommand(),
				},
			},
		})
	}

	upstream, err := s.client.ExecuteStream(stream.Context(), validated)
	if err != nil {
		return status.Error(codes.Unavailable, "retrieve upstream resource")
	}
	defer func() { _ = upstream.Body.Close() }()

	record, err := s.repo.CreatePlaceholder(stream.Context(), validated, upstream.Head)
	if err != nil {
		return status.Error(codes.Internal, "store retrieval record")
	}

	headers := &dataretrieverv1.RetrieveStreamHeaders{
		Record:             toProtoRetrievalRecord(record),
		UpstreamStatus:     int32(upstream.Head.StatusCode),
		ContentType:        upstream.Head.ContentType,
		ContentDisposition: upstream.Head.ContentDisposition,
	}
	if upstream.Head.StatusCode >= 300 && upstream.Head.StatusCode < 400 && upstream.Head.Location != "" {
		headers.RedirectUrl = upstream.Head.Location
		if err := stream.Send(headersEvent(headers)); err != nil {
			return s.streamContextError(stream, record.ID)
		}
		size := upstream.Head.ContentLength
		if size < 0 {
			size = -1
		}
		return s.finalize(stream.Context(), record.ID, "", size)
	}
	if err := stream.Send(headersEvent(headers)); err != nil {
		return s.streamContextError(stream, record.ID)
	}

	hash := sha256.New()
	var size int64
	var sample []byte
	overBudget := s.schemaBudget == 0
	buffer := make([]byte, streamChunkSize)
	for {
		n, readErr := upstream.Body.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
			size += int64(n)
			if !overBudget {
				if uint64(size) > s.schemaBudget {
					overBudget = true
					sample = nil
				} else {
					sample = append(sample, buffer[:n]...)
				}
			}
			if err := stream.Send(&dataretrieverv1.RetrieveStreamResponse{
				Event: &dataretrieverv1.RetrieveStreamResponse_Chunk{Chunk: append([]byte(nil), buffer[:n]...)},
			}); err != nil {
				return s.streamContextError(stream, record.ID)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return s.streamContextError(stream, record.ID)
		}
	}
	if err := s.finalize(stream.Context(), record.ID, hex.EncodeToString(hash.Sum(nil)), size); err != nil {
		return err
	}
	s.discoverOutputSchema(stream.Context(), validated, upstream.Head.ContentType, sample, overBudget)
	return nil
}

// discoverOutputSchema extracts the payload schema and replaces the stored
// output parameters for the resource version. A stale schema is preferable to
// a broken download, so replacement failures are logged without failing the
// already streamed response.
func (s *Server) discoverOutputSchema(ctx context.Context, validated domain.ValidatedRequest, contentType string, sample []byte, overBudget bool) {
	if overBudget || len(sample) == 0 {
		return
	}
	parameters := schema.Extract(contentType, sample)
	resourceID, err := uuid.Parse(validated.ResourceID)
	if err != nil {
		s.log.ErrorContext(ctx, "discover output schema: invalid resource ID", "error", err)
		return
	}
	versionID, err := uuid.Parse(validated.VersionID)
	if err != nil {
		s.log.ErrorContext(ctx, "discover output schema: invalid version ID", "error", err)
		return
	}
	if err := s.repo.ReplaceOutputParameters(ctx, resourceID, versionID, parameters); err != nil {
		s.log.ErrorContext(ctx, "discover output schema: replace output parameters", "error", err)
	}
}

func headersEvent(headers *dataretrieverv1.RetrieveStreamHeaders) *dataretrieverv1.RetrieveStreamResponse {
	return &dataretrieverv1.RetrieveStreamResponse{
		Event: &dataretrieverv1.RetrieveStreamResponse_Headers{Headers: headers},
	}
}

// finalize persists the exact streamed payload hash and size. Incomplete
// streams keep an empty hash and -1 size instead of misleading values.
func (s *Server) finalize(ctx context.Context, id int64, payloadHash string, payloadSize int64) error {
	if err := s.repo.FinalizeRetrieval(ctx, id, payloadHash, payloadSize); err != nil {
		return status.Error(codes.Internal, "finalize retrieval record")
	}
	return nil
}

// streamContextError marks the record incomplete and reports the stream
// failure. A canceled client context is normal (client went away) and is not
// rewritten as a server error.
func (s *Server) streamContextError(stream dataretrieverv1.DataRetrieverService_RetrieveStreamServer, id int64) error {
	_ = s.repo.FinalizeRetrieval(stream.Context(), id, "", -1)
	if ctxErr := stream.Context().Err(); ctxErr != nil {
		return ctxErr
	}
	return status.Error(codes.Unavailable, "retrieve upstream resource")
}

func (s *Server) GetRetrieval(ctx context.Context, request *dataretrieverv1.GetRetrievalRequest) (*dataretrieverv1.GetRetrievalResponse, error) {
	if request == nil || request.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}

	record, err := s.repo.GetByID(ctx, request.GetId())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "retrieval record not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "get retrieval record")
	}
	return &dataretrieverv1.GetRetrievalResponse{Record: toProtoRetrievalRecord(record)}, nil
}
