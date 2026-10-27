package grpcapi

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retrieval"
)

type Server struct {
	dataretrieverv1.UnimplementedDataRetrieverServiceServer
	service *retrieval.Service
}

func New(service *retrieval.Service) *Server {
	return &Server{service: service}
}

func (s *Server) RetrieveStream(request *dataretrieverv1.RetrieveStreamRequest, stream dataretrieverv1.DataRetrieverService_RetrieveStreamServer) error {
	input, err := toDomainRetrievalInput(request)
	if err != nil {
		return err
	}
	return s.service.Retrieve(stream.Context(), input, streamSink{stream: stream})
}

func (s *Server) GetRetrieval(ctx context.Context, request *dataretrieverv1.GetRetrievalRequest) (*dataretrieverv1.GetRetrievalResponse, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	record, err := s.service.Get(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	return &dataretrieverv1.GetRetrievalResponse{Record: toProtoRetrievalRecord(record)}, nil
}
