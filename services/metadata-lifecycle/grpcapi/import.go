package grpcapi

import (
	"bytes"
	"context"
	"fmt"
	"io"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadataimport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type importParser interface {
	Parse(io.Reader) (domain.ImportBatch, error)
}

type ImportServer struct {
	metadatalifecyclev1.UnimplementedImportServiceServer
	service *metadataimport.Service
	parser  importParser
}

func NewImport(service *metadataimport.Service, parser importParser) *ImportServer {
	return &ImportServer{service: service, parser: parser}
}

func (s *ImportServer) ImportTurtle(ctx context.Context, request *metadatalifecyclev1.ImportTurtleRequest) (*metadatalifecyclev1.ImportTurtleResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "import request is required") //nolint:wrapcheck
	}

	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	batch, err := s.parser.Parse(bytes.NewReader(request.GetTurtle()))
	if err != nil {
		return nil, fmt.Errorf("parse Turtle: %w: %w", domain.ErrInvalidImport, err)
	}

	result, err := s.service.Import(ctx, actor, batch)
	if err != nil {
		return nil, wrapHandlerError(err)
	}

	response, err := toProtoImportResponse(result)
	if err != nil {
		return nil, err
	}
	return response, nil
}
