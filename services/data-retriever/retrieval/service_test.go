package retrieval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/google/uuid"
)

type stubUpstreamClient struct {
	response *UpstreamResponse
	err      error
	calls    int
}

func (s *stubUpstreamClient) ExecuteStream(context.Context, domain.PreparedRetrieval) (*UpstreamResponse, error) {
	s.calls++
	return s.response, s.err
}

type stubRepository struct {
	record     domain.RetrievalRecord
	hash       string
	size       int64
	parameters []domain.OutputParameter
	creates    int
}

func (s *stubRepository) CreatePlaceholder(_ context.Context, request domain.PreparedRetrieval, head domain.ResponseHead) (domain.RetrievalRecord, error) {
	s.creates++
	s.record = domain.RetrievalRecord{
		ID:          42,
		ResourceID:  request.ResourceID,
		VersionID:   request.VersionID,
		HTTPStatus:  head.StatusCode,
		ContentType: head.ContentType,
	}
	return s.record, nil
}

func (s *stubRepository) FinalizeRetrieval(_ context.Context, _ int64, hash string, size int64) error {
	s.hash = hash
	s.size = size
	return nil
}

func (s *stubRepository) ReplaceOutputParameters(_ context.Context, _, _ uuid.UUID, parameters []domain.OutputParameter) error {
	s.parameters = parameters
	return nil
}

func (s *stubRepository) GetByID(context.Context, int64) (domain.RetrievalRecord, error) {
	return s.record, nil
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("upstream read failed") }
func (failingBody) Close() error             { return nil }

type recordingSink struct {
	events []string
	record *domain.RetrievalRecord
	head   domain.ResponseHead
	chunks [][]byte
}

func (s *recordingSink) SendHeaders(record *domain.RetrievalRecord, head domain.ResponseHead) error {
	s.events = append(s.events, "headers")
	s.record = record
	s.head = head
	return nil
}

func (s *recordingSink) SendChunk(chunk []byte) error {
	s.events = append(s.events, "chunk")
	s.chunks = append(s.chunks, chunk)
	return nil
}

func TestServiceStreamsAndFinalizesCompletePayload(t *testing.T) {
	const payload = `{"name":"Etna","active":true}`
	resourceID := uuid.MustParse("2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192")
	versionID := uuid.MustParse("67f6f345-5a27-467b-967b-0aa9e7f85bb1")
	repo := &stubRepository{}
	service := New(repo, &stubUpstreamClient{response: &UpstreamResponse{
		Head: domain.ResponseHead{StatusCode: 200, ContentType: "application/json", ContentLength: int64(len(payload))},
		Body: io.NopCloser(strings.NewReader(payload)),
	}}, slog.Default(), 1024, 1024)
	sink := &recordingSink{}
	input := domain.RetrievalInput{
		ResourceID: resourceID,
		VersionID:  versionID,
		Method:     "GET",
		URL:        "https://example.test/data",
	}

	if err := service.Retrieve(t.Context(), input, sink); err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(sink.events) != 2 || sink.events[0] != "headers" || sink.events[1] != "chunk" || sink.record.ID != 42 || string(sink.chunks[0]) != payload {
		t.Fatalf("stream events/record/chunks = %+v / %+v / %q", sink.events, sink.record, sink.chunks)
	}
	wantHash := sha256.Sum256([]byte(payload))
	if repo.hash != hex.EncodeToString(wantHash[:]) || repo.size != int64(len(payload)) {
		t.Fatalf("finalized hash/size = %q/%d", repo.hash, repo.size)
	}
	if len(repo.parameters) != 2 || repo.parameters[0] != (domain.OutputParameter{Name: "active", Kind: "BOOLEAN"}) || repo.parameters[1] != (domain.OutputParameter{Name: "name", Kind: "STRING"}) {
		t.Fatalf("discovered parameters = %+v", repo.parameters)
	}
}

func TestServiceStreamsResponseWhenRedirectHasNoLocation(t *testing.T) {
	const body = "not redirected"
	client := &stubUpstreamClient{response: &UpstreamResponse{
		Head: domain.ResponseHead{StatusCode: 302, ContentLength: int64(len(body))},
		Body: io.NopCloser(strings.NewReader(body)),
	}}
	repo := &stubRepository{}
	service := New(repo, client, slog.Default(), 1024, 1024)
	sink := &recordingSink{}
	input := domain.RetrievalInput{
		ResourceID: uuid.New(),
		VersionID:  uuid.New(),
		Method:     "GET",
		URL:        "https://example.test/data",
	}

	if err := service.Retrieve(t.Context(), input, sink); err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(sink.events) != 2 || sink.head.StatusCode != 302 || string(sink.chunks[0]) != body || repo.size != int64(len(body)) {
		t.Fatalf("response handling = events %v, status %d, body %q, size %d", sink.events, sink.head.StatusCode, sink.chunks, repo.size)
	}
}

func TestServiceReadFailureMarksRetrievalIncomplete(t *testing.T) {
	client := &stubUpstreamClient{response: &UpstreamResponse{
		Head: domain.ResponseHead{StatusCode: 200, ContentLength: -1},
		Body: failingBody{},
	}}
	repo := &stubRepository{}
	service := New(repo, client, slog.Default(), 1024, 1024)
	input := domain.RetrievalInput{
		ResourceID: uuid.New(),
		VersionID:  uuid.New(),
		Method:     "GET",
		URL:        "https://example.test/data",
	}

	err := service.Retrieve(t.Context(), input, &recordingSink{})
	if !apperror.IsCode(err, apperror.CodeInternal) {
		t.Fatalf("Retrieve() error = %v, want internal error", err)
	}
	if repo.hash != "" || repo.size != -1 {
		t.Fatalf("incomplete retrieval finalized as %q/%d", repo.hash, repo.size)
	}
}
