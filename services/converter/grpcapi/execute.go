package grpcapi

import (
	"bytes"
	"errors"
	"io"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	converterv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/converter/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func (s *Server) Execute(stream grpc.BidiStreamingServer[converterv1.ExecuteRequest, converterv1.ExecuteResponse]) (resultErr error) {
	first, err := stream.Recv()
	if err != nil {
		return apperror.Field("header", "Execution header is required.", err)
	}
	header := first.GetHeader()
	if header == nil {
		return apperror.Field("header", "Execution header must be first.", nil)
	}
	defer func() {
		if resultErr != nil {
			span := trace.SpanFromContext(stream.Context())
			span.SetAttributes(attribute.String("converter.source_url", header.GetSourceUrl()))
			span.RecordError(resultErr)
			span.SetStatus(codes.Error, "conversion failed")
		}
	}()
	conversionID, err := parseID("conversion_id", header.GetConversionId())
	if err != nil {
		return err
	}
	distributionID, err := parseID("distribution_id", header.GetDistributionId())
	if err != nil {
		return err
	}
	record, err := s.service.Executable(stream.Context(), conversionID, distributionID)
	if err != nil {
		return err
	}
	// input is buffered in memory; use disk-backed buffering if payloads or concurrency grow.
	var input []byte
	for {
		request, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		chunk, ok := request.Event.(*converterv1.ExecuteRequest_Chunk)
		if !ok || len(chunk.Chunk) == 0 {
			return apperror.Field("chunk", "A non-empty input chunk is required.", nil)
		}
		if int64(len(chunk.Chunk)) > s.max-int64(len(input)) {
			return apperror.New(apperror.CodeRequestTooLarge, "Input exceeds the configured limit.")
		}
		input = append(input, chunk.Chunk...)
	}
	if len(input) == 0 {
		return apperror.New(apperror.CodeValidationFailed, "Conversion input is empty.")
	}
	output, err := s.manager.Execute(stream.Context(), record.ID, record.Arguments, bytes.NewReader(input))
	if err != nil {
		return err
	}
	for len(output) > 0 {
		n := min(len(output), 64*1024)
		if err := stream.Send(&converterv1.ExecuteResponse{Chunk: output[:n]}); err != nil {
			return err
		}
		output = output[n:]
	}
	return nil
}
