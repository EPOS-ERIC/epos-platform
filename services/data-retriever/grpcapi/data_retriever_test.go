package grpcapi

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/retriever"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeRepository struct {
	getErr error
}

func (f fakeRepository) Create(_ context.Context, request domain.ValidatedRequest, _ retriever.Result) (repository.RetrievalRecord, error) {
	return repository.RetrievalRecord{
		ID:               1,
		ResourceID:       request.ResourceID,
		VersionID:        request.VersionID,
		Method:           "GET",
		CreatedAt:        time.Now().UTC(),
		OutputParameters: request.OutputParameters,
	}, nil
}

func (f fakeRepository) GetByID(_ context.Context, _ int64) (repository.RetrievalRecord, error) {
	if f.getErr != nil {
		return repository.RetrievalRecord{}, f.getErr
	}
	return repository.RetrievalRecord{ID: 1}, nil
}

type fakeClient struct {
	result retriever.Result
}

func (f fakeClient) Execute(_ context.Context, _ domain.ValidatedRequest) (retriever.Result, error) {
	return f.result, nil
}

type countingRepository struct {
	createCalls int
}

func (f *countingRepository) Create(_ context.Context, request domain.ValidatedRequest, _ retriever.Result) (repository.RetrievalRecord, error) {
	f.createCalls++
	return repository.RetrievalRecord{ID: 1, ResourceID: request.ResourceID, VersionID: request.VersionID}, nil
}

func (f *countingRepository) GetByID(_ context.Context, _ int64) (repository.RetrievalRecord, error) {
	return repository.RetrievalRecord{}, sql.ErrNoRows
}

type countingClient struct {
	executeCalls int
}

func (f *countingClient) Execute(_ context.Context, _ domain.ValidatedRequest) (retriever.Result, error) {
	f.executeCalls++
	return retriever.Result{}, nil
}

func retrieveRequest() *dataretrieverv1.RetrieveRequest {
	return &dataretrieverv1.RetrieveRequest{
		ResourceId:  "2f1d18c5-bb8a-4f2d-84f2-0a1af16dc192",
		VersionId:   "67f6f345-5a27-467b-967b-0aa9e7f85bb1",
		Method:      dataretrieverv1.HTTPMethod_HTTP_METHOD_GET,
		UrlTemplate: "https://example.org/files/{id}",
		PathParams:  map[string]string{"id": "1"},
		OutputParameters: []*dataretrieverv1.OutputParameter{{
			Name: "temperature",
			Kind: dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_FLOAT,
		}},
	}
}

func TestRetrieveReturnsInlinePayload(t *testing.T) {
	service := New(fakeRepository{}, fakeClient{result: retriever.Result{
		StatusCode:   200,
		ContentType:  "application/json",
		Payload:      []byte(`{"ok":true}`),
		PayloadHash:  "hash",
		ResponseDate: time.Now().UTC(),
	}})

	response, err := service.Retrieve(context.Background(), retrieveRequest())
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if string(response.GetInlinePayload()) != `{"ok":true}` {
		t.Fatalf("inline payload = %q", response.GetInlinePayload())
	}
	if response.GetRedirectUrl() != "" {
		t.Fatalf("unexpected redirect URL %q", response.GetRedirectUrl())
	}
	if len(response.GetRecord().GetOutputParameters()) != 1 {
		t.Fatalf("output parameter count = %d", len(response.GetRecord().GetOutputParameters()))
	}
}

func TestRetrieveReturnsRedirectForLargeBinaryPayload(t *testing.T) {
	service := New(fakeRepository{}, fakeClient{result: retriever.Result{
		StatusCode:   200,
		ContentType:  "application/octet-stream",
		PayloadHash:  "hash",
		PayloadSize:  1 << 30,
		IsBinary:     true,
		ResponseDate: time.Now().UTC(),
	}})

	response, err := service.Retrieve(context.Background(), retrieveRequest())
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if response.GetRedirectUrl() != "https://example.org/files/1" {
		t.Fatalf("redirect URL = %q", response.GetRedirectUrl())
	}
	if len(response.GetInlinePayload()) != 0 {
		t.Fatal("binary response must not include an inline payload")
	}
}

func TestRetrieveDryReturnsCurlWithoutExecutingOrPersisting(t *testing.T) {
	repo := &countingRepository{}
	client := &countingClient{}
	service := New(repo, client)
	request := retrieveRequest()
	request.Mode = dataretrieverv1.RetrieveMode_RETRIEVE_MODE_DRY
	request.BearerToken = "secret"

	response, err := service.Retrieve(context.Background(), request)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if response.GetCurlCommand() == "" {
		t.Fatal("dry response does not contain a curl command")
	}
	if response.GetRecord() != nil || response.GetUpstreamStatus() != 0 || response.GetContentType() != "" {
		t.Fatal("dry response must not contain execution data")
	}
	if client.executeCalls != 0 {
		t.Fatalf("Execute() calls = %d, want 0", client.executeCalls)
	}
	if repo.createCalls != 0 {
		t.Fatalf("Create() calls = %d, want 0", repo.createCalls)
	}
}

func TestGetRetrievalMapsMissingRecordToNotFound(t *testing.T) {
	service := New(fakeRepository{getErr: sql.ErrNoRows}, fakeClient{})

	_, err := service.GetRetrieval(context.Background(), &dataretrieverv1.GetRetrievalRequest{Id: 1})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.NotFound)
	}
}
