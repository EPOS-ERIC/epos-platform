package grpcapi

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type finalizeCall struct {
	id   int64
	hash string
	size int64
}

type fakeRepository struct {
	placeholderCalls int
	finalized        []finalizeCall
	finalizeErr      error
	replaced         [][]domain.OutputParameter
	replaceErr       error
	getErr           error
}

func (f *fakeRepository) CreatePlaceholder(_ context.Context, request domain.ValidatedRequest, head retriever.StreamHead) (repository.RetrievalRecord, error) {
	f.placeholderCalls++
	return repository.RetrievalRecord{
		ID:          1,
		ResourceID:  request.ResourceID,
		VersionID:   request.VersionID,
		Method:      "GET",
		HTTPStatus:  head.StatusCode,
		ContentType: head.ContentType,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

func (f *fakeRepository) FinalizeRetrieval(_ context.Context, id int64, payloadHash string, payloadSize int64) error {
	if f.finalizeErr != nil {
		return f.finalizeErr
	}
	f.finalized = append(f.finalized, finalizeCall{id: id, hash: payloadHash, size: payloadSize})
	return nil
}

func (f *fakeRepository) ReplaceOutputParameters(_ context.Context, _ uuid.UUID, _ uuid.UUID, parameters []domain.OutputParameter) error {
	f.replaced = append(f.replaced, parameters)
	return f.replaceErr
}

func (f *fakeRepository) GetByID(_ context.Context, _ int64) (repository.RetrievalRecord, error) {
	if f.getErr != nil {
		return repository.RetrievalRecord{}, f.getErr
	}
	return repository.RetrievalRecord{ID: 1}, nil
}

type fakeClient struct {
	stream *retriever.UpstreamStream
	err    error
	calls  int
}

func (f *fakeClient) ExecuteStream(_ context.Context, _ domain.ValidatedRequest) (*retriever.UpstreamStream, error) {
	f.calls++
	return f.stream, f.err
}

type fakeStream struct {
	grpc.ServerStream
	ctx     context.Context
	sent    []*dataretrieverv1.RetrieveStreamResponse
	sendErr error
}

func (f *fakeStream) Context() context.Context { return f.ctx }

func (f *fakeStream) Send(response *dataretrieverv1.RetrieveStreamResponse) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sent = append(f.sent, response)
	return nil
}

func retrieveRequest() *dataretrieverv1.RetrieveStreamRequest {
	return &dataretrieverv1.RetrieveStreamRequest{
		ResourceId:  "2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192",
		VersionId:   "67f6f345-5a27-467b-967b-0aa9e7f85bb1",
		Method:      dataretrieverv1.HTTPMethod_HTTP_METHOD_GET,
		UrlTemplate: "https://example.org/files/{id}",
		PathParams:  map[string]string{"id": "1"},
	}
}

func inlineUpstream(payload string) *retriever.UpstreamStream {
	return &retriever.UpstreamStream{
		Head: retriever.StreamHead{
			StatusCode:    200,
			ContentType:   "application/json",
			ContentLength: int64(len(payload)),
		},
		Body: io.NopCloser(strings.NewReader(payload)),
	}
}

func headersOf(event *dataretrieverv1.RetrieveStreamResponse) *dataretrieverv1.RetrieveStreamHeaders {
	return event.GetHeaders()
}

func TestRetrieveStreamDrySendsCurlWithoutExecutingOrPersisting(t *testing.T) {
	repo := &fakeRepository{}
	client := &fakeClient{}
	service := New(repo, client, testLogger(), 2<<20)
	request := retrieveRequest()
	request.Mode = dataretrieverv1.RetrieveMode_RETRIEVE_MODE_DRY
	request.BearerToken = "secret"
	stream := &fakeStream{ctx: context.Background()}

	if err := service.RetrieveStream(request, stream); err != nil {
		t.Fatalf("RetrieveStream() error = %v", err)
	}
	if len(stream.sent) != 1 {
		t.Fatalf("events = %d, want 1", len(stream.sent))
	}
	if headersOf(stream.sent[0]).GetCurlCommand() == "" {
		t.Fatal("dry event does not contain a curl command")
	}
	if client.calls != 0 {
		t.Fatalf("ExecuteStream() calls = %d, want 0", client.calls)
	}
	if repo.placeholderCalls != 0 {
		t.Fatalf("CreatePlaceholder() calls = %d, want 0", repo.placeholderCalls)
	}
}

func TestRetrieveStreamRelaysBodyInChunksAndFinalizes(t *testing.T) {
	payload := `{"ok":true}`
	repo := &fakeRepository{}
	client := &fakeClient{stream: inlineUpstream(payload)}
	service := New(repo, client, testLogger(), 2<<20)
	stream := &fakeStream{ctx: context.Background()}

	if err := service.RetrieveStream(retrieveRequest(), stream); err != nil {
		t.Fatalf("RetrieveStream() error = %v", err)
	}
	if len(stream.sent) != 2 {
		t.Fatalf("events = %d, want headers + 1 chunk", len(stream.sent))
	}
	headers := headersOf(stream.sent[0])
	if headers.GetUpstreamStatus() != 200 || headers.GetContentType() != "application/json" {
		t.Fatalf("headers = %+v", headers)
	}
	if headers.GetRecord().GetId() != 1 {
		t.Fatalf("record id = %d, want 1", headers.GetRecord().GetId())
	}
	if string(stream.sent[1].GetChunk()) != payload {
		t.Fatalf("chunk = %q", stream.sent[1].GetChunk())
	}
	if len(headersOf(stream.sent[0]).GetCurlCommand()) != 0 {
		t.Fatal("execute event must not contain a curl command")
	}
	if len(repo.finalized) != 1 {
		t.Fatalf("finalize calls = %d, want 1", len(repo.finalized))
	}
	wantHash := sha256.Sum256([]byte(payload))
	if repo.finalized[0].hash != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("finalized hash = %q", repo.finalized[0].hash)
	}
	if repo.finalized[0].size != int64(len(payload)) {
		t.Fatalf("finalized size = %d, want %d", repo.finalized[0].size, len(payload))
	}
	if len(repo.replaced) != 1 {
		t.Fatalf("replace output parameters calls = %d, want 1", len(repo.replaced))
	}
	parameters := make(map[string]string, len(repo.replaced[0]))
	for _, parameter := range repo.replaced[0] {
		parameters[parameter.Name] = parameter.Kind
	}
	if parameters["ok"] != "BOOLEAN" {
		t.Fatalf("discovered parameters = %v, want ok=BOOLEAN", parameters)
	}
}

func TestRetrieveStreamSkipsSchemaExtractionOverBudget(t *testing.T) {
	payload := `{"field":"value"}`
	repo := &fakeRepository{}
	client := &fakeClient{stream: inlineUpstream(payload)}
	service := New(repo, client, testLogger(), 1)
	stream := &fakeStream{ctx: context.Background()}

	if err := service.RetrieveStream(retrieveRequest(), stream); err != nil {
		t.Fatalf("RetrieveStream() error = %v", err)
	}
	if len(repo.replaced) != 0 {
		t.Fatalf("replace output parameters calls = %d, want 0", len(repo.replaced))
	}
}

func TestRetrieveStreamRelaysRedirectLocation(t *testing.T) {
	repo := &fakeRepository{}
	client := &fakeClient{stream: &retriever.UpstreamStream{
		Head: retriever.StreamHead{
			StatusCode:         302,
			ContentType:        "text/html",
			ContentLength:      -1,
			Location:           "https://example.org/files/1/download",
			ContentDisposition: `attachment; filename="original.tif"`,
		},
		Body: io.NopCloser(strings.NewReader("redirect")),
	}}
	service := New(repo, client, testLogger(), 2<<20)
	stream := &fakeStream{ctx: context.Background()}

	if err := service.RetrieveStream(retrieveRequest(), stream); err != nil {
		t.Fatalf("RetrieveStream() error = %v", err)
	}
	if len(stream.sent) != 1 {
		t.Fatalf("events = %d, want headers only", len(stream.sent))
	}
	headers := headersOf(stream.sent[0])
	if headers.GetUpstreamStatus() != 302 {
		t.Fatalf("upstream status = %d, want 302", headers.GetUpstreamStatus())
	}
	if headers.GetRedirectUrl() != "https://example.org/files/1/download" {
		t.Fatalf("redirect URL = %q", headers.GetRedirectUrl())
	}
	if headers.GetContentDisposition() != `attachment; filename="original.tif"` {
		t.Fatalf("content disposition = %q", headers.GetContentDisposition())
	}
}

func TestRetrieveStreamMapsUpstreamFailureToUnavailable(t *testing.T) {
	repo := &fakeRepository{}
	client := &fakeClient{err: errors.New("dial")}
	service := New(repo, client, testLogger(), 2<<20)
	stream := &fakeStream{ctx: context.Background()}

	if err := service.RetrieveStream(retrieveRequest(), stream); status.Code(err) != codes.Unavailable {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.Unavailable)
	}
}

func TestRetrieveStreamRejectsInvalidRequest(t *testing.T) {
	service := New(&fakeRepository{}, &fakeClient{}, testLogger(), 2<<20)
	stream := &fakeStream{ctx: context.Background()}
	request := retrieveRequest()
	request.ResourceId = "not-a-uuid"

	if err := service.RetrieveStream(request, stream); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
}

func TestGetRetrievalMapsMissingRecordToNotFound(t *testing.T) {
	service := New(&fakeRepository{getErr: sql.ErrNoRows}, &fakeClient{}, testLogger(), 2<<20)

	_, err := service.GetRetrieval(context.Background(), &dataretrieverv1.GetRetrievalRequest{Id: 1})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.NotFound)
	}
}
