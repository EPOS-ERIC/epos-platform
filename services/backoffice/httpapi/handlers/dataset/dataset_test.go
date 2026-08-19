package dataset

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/datasetlifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/httpapi/transport"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type fakeDatasetService struct {
	create func(context.Context, string, uuid.UUID, metadata.Dataset) (datasetlifecycle.Dataset, error)
	read   func(context.Context, *string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, bool, error)
}

func (s fakeDatasetService) CreateDraft(ctx context.Context, actor string, group uuid.UUID, value metadata.Dataset) (datasetlifecycle.Dataset, error) {
	return s.create(ctx, actor, group, value)
}

func (s fakeDatasetService) Read(ctx context.Context, actor *string, datasetID, versionID uuid.UUID) (datasetlifecycle.Dataset, bool, error) {
	return s.read(ctx, actor, datasetID, versionID)
}

func (fakeDatasetService) ReplaceDraft(context.Context, string, uuid.UUID, uuid.UUID, int32, metadata.Dataset) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Submit(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Withdraw(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Reject(context.Context, string, uuid.UUID, uuid.UUID, string) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Reopen(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Publish(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}
func (fakeDatasetService) Discard(context.Context, string, uuid.UUID, uuid.UUID) error { return nil }
func (fakeDatasetService) DraftFromPublication(context.Context, string, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) DraftFromLatestArchive(context.Context, string, uuid.UUID) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func (fakeDatasetService) CopyPublished(context.Context, string, uuid.UUID, uuid.UUID, string) (datasetlifecycle.Dataset, error) {
	return datasetlifecycle.Dataset{}, nil
}

func TestDatasetCommandsRequireActor(t *testing.T) {
	router := chi.NewRouter()
	router.Post("/resources/dataset", New(fakeDatasetService{}).Create)
	request := httptest.NewRequest(http.MethodPost, "/resources/dataset", strings.NewReader(`{}`))
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
	datasets := New(fakeDatasetService{create: func(_ context.Context, actualActor string, actualGroup uuid.UUID, value metadata.Dataset) (datasetlifecycle.Dataset, error) {
		if actualActor != actor.String() || actualGroup != group || value.SourceIRI != "https://example.test/dataset" {
			t.Fatal("unexpected create request")
		}
		return datasetlifecycle.Dataset{}, nil
	}})
	router.Post("/resources/dataset", datasets.Create)
	request := httptest.NewRequest(http.MethodPost, "/resources/dataset", strings.NewReader(`{"group_id":"`+group.String()+`","metadata":{"source_iri":"https://example.test/dataset"}}`))
	request.Header.Set(transport.ActorHeader, actor.String())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
}

func TestCreateDatasetRejectsResponseFields(t *testing.T) {
	actor := uuid.New()
	group := uuid.New()
	router := chi.NewRouter()
	router.Post("/resources/dataset", New(fakeDatasetService{}).Create)
	request := httptest.NewRequest(http.MethodPost, "/resources/dataset", strings.NewReader(`{"group_id":"`+group.String()+`","metadata":{"source_iri":"dataset:test","status":"DRAFT"}}`))
	request.Header.Set(transport.ActorHeader, actor.String())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestCreateDatasetDecodesTypedParameter(t *testing.T) {
	actor := uuid.New()
	group := uuid.New()
	parameterID := uuid.New()
	router := chi.NewRouter()
	datasets := New(fakeDatasetService{create: func(_ context.Context, _ string, _ uuid.UUID, value metadata.Dataset) (datasetlifecycle.Dataset, error) {
		parameter, ok := value.Distributions[0].WebService.Parameters[0].(metadata.StringParameter)
		if !ok || parameter.Metadata.ID != parameterID || parameter.Kind() != metadata.ParameterKindString {
			t.Fatalf("parameter = %#v", value.Distributions[0].WebService.Parameters[0])
		}
		return datasetlifecycle.Dataset{}, nil
	}})
	router.Post("/resources/dataset", datasets.Create)
	body := `{"group_id":"` + group.String() + `","metadata":{"source_iri":"dataset:test","distributions":[{"title":"Test","type":"http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE","web_service":{"name":"Test","template_url":"https://example.test/{value}","format":"application/json","parameters":[{"metadata":{"id":"` + parameterID.String() + `","kind":"STRING","variable":"value","property":"schema:value","label":"Value","required":true,"multiple_values":false,"read_only":false},"default":null,"allowed_values":[],"pattern":""}]}}]}}`
	request := httptest.NewRequest(http.MethodPost, "/resources/dataset", strings.NewReader(body))
	request.Header.Set(transport.ActorHeader, actor.String())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestPublicDatasetOmitsContactPoints(t *testing.T) {
	id := uuid.New()
	router := chi.NewRouter()
	versionID := uuid.New()
	datasets := New(fakeDatasetService{read: func(_ context.Context, actor *string, actualID, actualVersionID uuid.UUID) (datasetlifecycle.Dataset, bool, error) {
		if actor != nil || actualID != id || actualVersionID != versionID {
			t.Fatal("unexpected read request")
		}
		return datasetlifecycle.Dataset{Metadata: metadata.Dataset{ContactPoints: []metadata.ContactPoint{{Email: "private@example.test"}}, Distributions: []metadata.Distribution{{WebService: &metadata.WebService{ContactPoints: []metadata.ContactPoint{{Email: "private@example.test"}}}}}}}, false, nil
	}})
	router.Get("/resources/dataset/{resource_id}/versions/{version_id}", datasets.Get)
	request := httptest.NewRequest(http.MethodGet, "/resources/dataset/"+id.String()+"/versions/"+versionID.String(), nil)
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
	datasets := New(fakeDatasetService{read: func(_ context.Context, actualActor *string, actualID, actualVersionID uuid.UUID) (datasetlifecycle.Dataset, bool, error) {
		if actualActor == nil || *actualActor != actor.String() || actualID != id || actualVersionID != versionID {
			t.Fatal("unexpected read request")
		}
		return datasetlifecycle.Dataset{Metadata: metadata.Dataset{ContactPoints: []metadata.ContactPoint{{Email: "private@example.test"}}}}, false, nil
	}})
	router.Get("/resources/dataset/{resource_id}/versions/{version_id}", datasets.Get)
	request := httptest.NewRequest(http.MethodGet, "/resources/dataset/"+id.String()+"/versions/"+versionID.String(), nil)
	request.Header.Set(transport.ActorHeader, actor.String())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "private@example.test") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestDatasetReadExposesGroupOnce(t *testing.T) {
	id := uuid.New()
	versionID := uuid.New()
	groupID := uuid.New()
	router := chi.NewRouter()
	datasets := New(fakeDatasetService{read: func(context.Context, *string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, bool, error) {
		return datasetlifecycle.Dataset{
			Group:    metadata.CategoryScheme{ID: groupID, SourceIRI: "scheme:test", Name: "Test", Concept: metadata.TopLevelConceptData},
			Metadata: metadata.Dataset{Categories: []metadata.Category{{ID: uuid.New(), SourceIRI: "category:test", Name: "Category", Scheme: metadata.CategoryScheme{ID: groupID}}}},
		}, true, nil
	}})
	router.Get("/resources/dataset/{resource_id}/versions/{version_id}", datasets.Get)
	request := httptest.NewRequest(http.MethodGet, "/resources/dataset/"+id.String()+"/versions/"+versionID.String(), nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"group":{"id":"`+groupID.String()) || strings.Contains(body, `"scheme"`) || strings.Contains(body, `"group":null`) {
		t.Fatalf("response shape = %s", body)
	}
}

func TestCreateDatasetDoesNotExposeInternalErrors(t *testing.T) {
	actor := uuid.New()
	group := uuid.New()
	router := chi.NewRouter()
	datasets := New(fakeDatasetService{create: func(context.Context, string, uuid.UUID, metadata.Dataset) (datasetlifecycle.Dataset, error) {
		return datasetlifecycle.Dataset{}, errors.New("postgres: password authentication failed")
	}})
	router.Post("/resources/dataset", datasets.Create)
	request := httptest.NewRequest(http.MethodPost, "/resources/dataset", strings.NewReader(`{"group_id":"`+group.String()+`","metadata":{"source_iri":"https://example.test/dataset"}}`))
	request.Header.Set(transport.ActorHeader, actor.String())
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
