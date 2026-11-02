package grpcapi

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadataimport"
	"go.opentelemetry.io/otel/trace"
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
		return nil, apperror.New(apperror.CodeValidationFailed, "import request is required") //nolint:wrapcheck
	}

	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	batch, err := s.parser.Parse(bytes.NewReader(request.GetTurtle()))
	if err != nil {
		trace.SpanFromContext(ctx).RecordError(err)
		return nil, fmt.Errorf("parse Turtle: %w: %w", apperror.Default(apperror.CodeValidationFailed), err)
	}

	result, err := s.service.Import(ctx, actor, batch)
	if err != nil {
		return nil, err
	}

	response, err := toProtoImportResponse(result)
	if err != nil {
		return nil, err
	}
	return response, nil
}
