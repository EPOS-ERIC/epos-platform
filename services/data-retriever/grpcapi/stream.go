package grpcapi

import (
	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
)

type streamSink struct {
	stream dataretrieverv1.DataRetrieverService_RetrieveStreamServer
}

func (s streamSink) SendDry(command string) error {
	return s.stream.Send(headersEvent(&dataretrieverv1.RetrieveStreamHeaders{CurlCommand: command}))
}

func (s streamSink) SendHeaders(record *domain.RetrievalRecord, head domain.ResponseHead) error {
	headers := &dataretrieverv1.RetrieveStreamHeaders{
		UpstreamStatus: int32(head.StatusCode),
		ContentType:    head.ContentType,
	}
	if record != nil {
		headers.Record = toProtoRetrievalRecord(*record)
	}
	return s.stream.Send(headersEvent(headers))
}

func (s streamSink) SendChunk(chunk []byte) error {
	return s.stream.Send(&dataretrieverv1.RetrieveStreamResponse{
		Event: &dataretrieverv1.RetrieveStreamResponse_Chunk{Chunk: chunk},
	})
}

func headersEvent(headers *dataretrieverv1.RetrieveStreamHeaders) *dataretrieverv1.RetrieveStreamResponse {
	return &dataretrieverv1.RetrieveStreamResponse{
		Event: &dataretrieverv1.RetrieveStreamResponse_Headers{Headers: headers},
	}
}
