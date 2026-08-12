package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/datasetlifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type fakeDatasetService struct {
	create func(context.Context, uuid.UUID, uuid.UUID, metadata.Dataset) (datasetlifecycle.Dataset, error)
	read   func(context.Context, *uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, bool, error)
}

func (s fakeDatasetService) CreateDraft(ctx context.Context, actor, group uuid.UUID, value metadata.Dataset) (datasetlifecycle.Dataset, error) {
	return s.create(ctx, actor, group, value)
}

func (s fakeDatasetService) Read(ctx context.Context, actor *uuid.UUID, datasetID, versionID uuid.UUID) (datasetlifecycle.Dataset, bool, error) {
	return s.read(ctx, actor, datasetID, versionID)
}

func (fakeDatasetService) ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int32, metadata.Dataset) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Submit(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Withdraw(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Reject(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Reopen(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Publish(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Unpublish(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}
func (fakeDatasetService) Discard(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error { return nil }
func (fakeDatasetService) DraftFromPublication(context.Context, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) DraftFromLatestArchive(context.Context, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) CopyPublished(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func TestDatasetCommandsRequireActor(t *testing.T) {
	router := chi.NewRouter()
	router.Post("/groups/{group_id}/datasets", NewDatasets(fakeDatasetService{}).Create)
	request := httptest.NewRequest(http.MethodPost, "/groups/"+uuid.NewString()+"/datasets", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestCreateDatasetPassesActorAndMetadata(t *testing.T) {
	actor := uuid.New()
	group := uuid.New()
	router := chi.NewRouter()
	datasets := NewDatasets(fakeDatasetService{create: func(_ context.Context, actualActor, actualGroup uuid.UUID, value metadata.Dataset) (datasetlifecycle.Dataset, error) {
		if actualActor != actor || actualGroup != group || value.SourceIRI != "https://example.test/dataset" {
			t.Fatal("unexpected create request")
		}
		return datasetlifecycle.Dataset{}, nil
	}})
	router.Post("/groups/{group_id}/datasets", datasets.Create)
	request := httptest.NewRequest(http.MethodPost, "/groups/"+group.String()+"/datasets", strings.NewReader(`{"source_iri":"https://example.test/dataset"}`))
	request.Header.Set(actorHeader, actor.String())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
}

func TestPublicDatasetOmitsContactPoints(t *testing.T) {
	id := uuid.New()
	router := chi.NewRouter()
	versionID := uuid.New()
	datasets := NewDatasets(fakeDatasetService{read: func(_ context.Context, actor *uuid.UUID, actualID, actualVersionID uuid.UUID) (datasetlifecycle.Dataset, bool, error) {
		if actor != nil || actualID != id || actualVersionID != versionID {
			t.Fatal("unexpected read request")
		}
		return datasetlifecycle.Dataset{Metadata: metadata.Dataset{ContactPoints: []metadata.ContactPoint{{Email: "private@example.test"}}, Distributions: []metadata.Distribution{{WebService: &metadata.WebService{ContactPoints: []metadata.ContactPoint{{Email: "private@example.test"}}}}}}}, false, nil
	}})
	router.Get("/datasets/{id}/{version_id}", datasets.Get)
	request := httptest.NewRequest(http.MethodGet, "/datasets/"+id.String()+"/"+versionID.String(), nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "private@example.test") {
		t.Fatalf("public response = %d %s", response.Code, response.Body.String())
	}
}

func TestDatasetReadOmitsContactPointsWithoutEligibility(t *testing.T) {
	id := uuid.New()
	actor := uuid.New()
	router := chi.NewRouter()
	versionID := uuid.New()
	datasets := NewDatasets(fakeDatasetService{read: func(_ context.Context, actualActor *uuid.UUID, actualID, actualVersionID uuid.UUID) (datasetlifecycle.Dataset, bool, error) {
		if actualActor == nil || *actualActor != actor || actualID != id || actualVersionID != versionID {
			t.Fatal("unexpected read request")
		}
		return datasetlifecycle.Dataset{Metadata: metadata.Dataset{ContactPoints: []metadata.ContactPoint{{Email: "private@example.test"}}}}, false, nil
	}})
	router.Get("/datasets/{id}/{version_id}", datasets.Get)
	request := httptest.NewRequest(http.MethodGet, "/datasets/"+id.String()+"/"+versionID.String(), nil)
	request.Header.Set(actorHeader, actor.String())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "private@example.test") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestCreateDatasetDoesNotExposeInternalErrors(t *testing.T) {
	actor := uuid.New()
	group := uuid.New()
	router := chi.NewRouter()
	datasets := NewDatasets(fakeDatasetService{create: func(context.Context, uuid.UUID, uuid.UUID, metadata.Dataset) (datasetlifecycle.Dataset, error) {
		return datasetlifecycle.Dataset{}, errors.New("postgres: password authentication failed")
	}})
	router.Post("/groups/{group_id}/datasets", datasets.Create)
	request := httptest.NewRequest(http.MethodPost, "/groups/"+group.String()+"/datasets", strings.NewReader(`{"source_iri":"https://example.test/dataset"}`))
	request.Header.Set(actorHeader, actor.String())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	var body struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode problem response: %v", err)
	}
	if body.Title != "internal_server_error" || body.Status != http.StatusInternalServerError || body.Detail != "dataset request failed" {
		t.Fatalf("problem = %#v", body)
	}
}
