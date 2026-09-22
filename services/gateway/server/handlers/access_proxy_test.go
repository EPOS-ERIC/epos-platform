package handlers

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"google.golang.org/grpc"
)

type stubAccessStream struct {
	grpc.ClientStream
	events []*dataretrieverv1.RetrieveStreamResponse
	err    error
}

func (f *stubAccessStream) Recv() (*dataretrieverv1.RetrieveStreamResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.events) == 0 {
		return nil, io.EOF
	}
	event := f.events[0]
	f.events = f.events[1:]
	return event, nil
}

type stubAccessRetriever struct {
	stream dataretrieverv1.DataRetrieverService_RetrieveStreamClient
	err    error
	calls  int
}

func (f *stubAccessRetriever) RetrieveStream(_ context.Context, _ *dataretrieverv1.RetrieveStreamRequest) (dataretrieverv1.DataRetrieverService_RetrieveStreamClient, error) {
	f.calls++
	return f.stream, f.err
}

func testAccessLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func executeBody() []byte {
	return []byte(`{"resource_id":"2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192","version_id":"67f6f345-5a27-467b-967b-0aa9e7f85bb1","mode":"EXECUTE","method":"GET","url_template":"https://example.org/files"}`)
}

func streamHeadersEvent(headers *dataretrieverv1.RetrieveStreamHeaders) *dataretrieverv1.RetrieveStreamResponse {
	return &dataretrieverv1.RetrieveStreamResponse{
		Event: &dataretrieverv1.RetrieveStreamResponse_Headers{Headers: headers},
	}
}

func streamChunkEvent(chunk string) *dataretrieverv1.RetrieveStreamResponse {
	return &dataretrieverv1.RetrieveStreamResponse{
		Event: &dataretrieverv1.RetrieveStreamResponse_Chunk{Chunk: []byte(chunk)},
	}
}

func TestAccessProxyStreamsUpstreamBodyAndStatus(t *testing.T) {
	stub := &stubAccessRetriever{stream: &stubAccessStream{events: []*dataretrieverv1.RetrieveStreamResponse{
		streamHeadersEvent(&dataretrieverv1.RetrieveStreamHeaders{
			Record:         &dataretrieverv1.RetrievalRecord{Id: 7},
			UpstreamStatus: 500,
			ContentType:    "application/json; charset=utf-8",
		}),
		streamChunkEvent(`{"error":`),
		streamChunkEvent(`true}`),
	}}}
	handler := AccessProxyMiddleware(stub, testAccessLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("DRY/fallback handler must not be called for EXECUTE")
	}))

	request := httptest.NewRequest(http.MethodPost, "/v1/access", bytes.NewReader(executeBody()))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	result := recorder.Result()
	defer func() { _ = result.Body.Close() }()
	if result.StatusCode != 500 {
		t.Fatalf("status = %d, want 500", result.StatusCode)
	}
	if result.Header.Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", result.Header.Get("Content-Type"))
	}
	if result.Header.Get("X-Retrieval-Id") != "7" {
		t.Fatalf("retrieval id header = %q", result.Header.Get("X-Retrieval-Id"))
	}
	body, _ := io.ReadAll(result.Body)
	if string(body) != `{"error":true}` {
		t.Fatalf("body = %q", body)
	}
	if !recorder.Flushed {
		t.Fatal("streamed response was never flushed")
	}
	if stub.calls != 1 {
		t.Fatalf("retrieve calls = %d, want 1", stub.calls)
	}
}

func TestAccessProxyRelaysRedirectWithLocation(t *testing.T) {
	stub := &stubAccessRetriever{stream: &stubAccessStream{events: []*dataretrieverv1.RetrieveStreamResponse{
		streamHeadersEvent(&dataretrieverv1.RetrieveStreamHeaders{
			Record:             &dataretrieverv1.RetrievalRecord{Id: 9},
			UpstreamStatus:     302,
			ContentType:        "text/html",
			ContentDisposition: `attachment; filename="original.tif"`,
			RedirectUrl:        "https://example.org/final",
		}),
	}}}
	handler := AccessProxyMiddleware(stub, testAccessLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("fallback handler must not be called for EXECUTE")
	}))

	request := httptest.NewRequest(http.MethodPost, "/v1/access", bytes.NewReader(executeBody()))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	result := recorder.Result()
	defer func() { _ = result.Body.Close() }()
	if result.StatusCode != 302 {
		t.Fatalf("status = %d, want 302", result.StatusCode)
	}
	if result.Header.Get("Location") != "https://example.org/final" {
		t.Fatalf("location = %q", result.Header.Get("Location"))
	}
	if result.Header.Get("Content-Disposition") != `attachment; filename="original.tif"` {
		t.Fatalf("content disposition = %q", result.Header.Get("Content-Disposition"))
	}
}

func TestAccessProxyMapsStreamFailureToBadGateway(t *testing.T) {
	stub := &stubAccessRetriever{err: context.DeadlineExceeded}
	handler := AccessProxyMiddleware(stub, testAccessLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("fallback handler must not be called for EXECUTE")
	}))

	request := httptest.NewRequest(http.MethodPost, "/v1/access", bytes.NewReader(executeBody()))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", recorder.Code)
	}
}

func TestAccessProxyFallsThroughForDry(t *testing.T) {
	stub := &stubAccessRetriever{}
	fallbackCalled := false
	handler := AccessProxyMiddleware(stub, testAccessLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fallbackCalled = true
		w.WriteHeader(http.StatusTeapot)
	}))

	body := []byte(`{"resource_id":"2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192","version_id":"67f6f345-5a27-467b-967b-0aa9e7f85bb1","mode":"DRY","method":"GET","url_template":"https://example.org/files"}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/access", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if !fallbackCalled {
		t.Fatal("DRY request must fall through to the generated server")
	}
	if stub.calls != 0 {
		t.Fatalf("retrieve calls = %d, want 0", stub.calls)
	}
}
