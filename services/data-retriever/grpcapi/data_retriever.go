package grpcapi

import (
	"context"
	"database/sql"
	"errors"

	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type retrievalClient interface {
	Execute(context.Context, domain.ValidatedRequest) (retriever.Result, error)
}

type Server struct {
	dataretrieverv1.UnimplementedDataRetrieverServiceServer
	repo   repository.RetrievalRepository
	client retrievalClient
}

func New(repo repository.RetrievalRepository, client retrievalClient) *Server {
	return &Server{repo: repo, client: client}
}

func (s *Server) Retrieve(ctx context.Context, request *dataretrieverv1.RetrieveRequest) (*dataretrieverv1.RetrieveResponse, error) {
	retrieveRequest, err := toDomainRetrieveRequest(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	validated, err := retrieveRequest.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if validated.Mode == domain.RetrieveModeDry {
		return &dataretrieverv1.RetrieveResponse{
			Result: &dataretrieverv1.RetrieveResponse_CurlCommand{CurlCommand: validated.CurlCommand()},
		}, nil
	}

	result, err := s.client.Execute(ctx, validated)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "retrieve upstream resource")
	}

	record, err := s.repo.Create(ctx, validated, result)
	if err != nil {
		return nil, status.Error(codes.Internal, "store retrieval record")
	}

	response := &dataretrieverv1.RetrieveResponse{
		Record:         toProtoRetrievalRecord(record),
		UpstreamStatus: int32(result.StatusCode),
		ContentType:    result.ContentType,
	}
	if result.IsBinary {
		response.Result = &dataretrieverv1.RetrieveResponse_RedirectUrl{RedirectUrl: validated.GeneratedURL}
	} else {
		response.Result = &dataretrieverv1.RetrieveResponse_InlinePayload{InlinePayload: result.Payload}
	}
	return response, nil
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
