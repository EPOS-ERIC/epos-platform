package metadataimport

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authorization"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/google/uuid"
)

type fakeParser struct {
	batch Batch
	err   error
}

func (p fakeParser) Parse(io.Reader, Options) (Batch, error) { return p.batch, p.err }

type fakeAuthorizer struct {
	result authorization.Result
	err    error
}

func (a fakeAuthorizer) AuthorizeAdmin(context.Context, uuid.UUID) (authorization.Result, error) {
	return a.result, a.err
}

type fakeRepository struct {
	called bool
}

func (r *fakeRepository) Import(_ context.Context, _ uuid.UUID, batch Batch) (Result, error) {
	r.called = true
	return Result{Datasets: len(batch.Datasets)}, nil
}

func TestImportDeniesNonAdminBeforeParsing(t *testing.T) {
	repository := &fakeRepository{}
	service := New(fakeParser{err: errors.New("must not parse")}, fakeAuthorizer{result: authorization.Result{Allowed: false, Reason: "global admin required"}}, repository)

	_, err := service.Import(context.Background(), uuid.New(), strings.NewReader(""))
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("Import() error = %v, want ErrDenied", err)
	}
	if repository.called {
		t.Fatal("repository called for denied import")
	}
}

func TestImportRequiresExactlyOneDatasetScheme(t *testing.T) {
	schemeA := metadata.CategoryScheme{ID: uuid.New(), SourceIRI: "scheme:a", Name: "A", Concept: metadata.TopLevelConceptData}
	schemeB := metadata.CategoryScheme{ID: uuid.New(), SourceIRI: "scheme:b", Name: "B", Concept: metadata.TopLevelConceptData}
	tests := []struct {
		name       string
		categories []metadata.Category
	}{
		{name: "no category"},
		{name: "multiple schemes", categories: []metadata.Category{
			{ID: uuid.New(), SourceIRI: "category:a", Name: "A", Scheme: schemeA},
			{ID: uuid.New(), SourceIRI: "category:b", Name: "B", Scheme: schemeB},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := New(fakeParser{batch: Batch{Datasets: []metadata.Dataset{{SourceIRI: "dataset:1", Categories: test.categories}}}}, fakeAuthorizer{result: authorization.Result{Allowed: true}}, repository)
			_, err := service.Import(context.Background(), uuid.New(), strings.NewReader(""))
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("Import() error = %v, want ErrInvalid", err)
			}
			if repository.called {
				t.Fatal("repository called for invalid ownership")
			}
		})
	}
}

func TestImportPersistsSingleSchemeDataset(t *testing.T) {
	scheme := metadata.CategoryScheme{ID: uuid.New(), SourceIRI: "scheme:a", Name: "A", Concept: metadata.TopLevelConceptData}
	batch := Batch{Datasets: []metadata.Dataset{{
		SourceIRI:  "dataset:1",
		Categories: []metadata.Category{{ID: uuid.New(), SourceIRI: "category:a", Name: "A", Scheme: scheme}},
	}}}
	repository := &fakeRepository{}
	service := New(fakeParser{batch: batch}, fakeAuthorizer{result: authorization.Result{Allowed: true}}, repository)

	result, err := service.Import(context.Background(), uuid.New(), strings.NewReader(""))
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if !repository.called || result.Datasets != 1 {
		t.Fatalf("repository called = %v, datasets = %d", repository.called, result.Datasets)
	}
}

func TestImportRejectsSoftwareUntilLifecycleExists(t *testing.T) {
	repository := &fakeRepository{}
	service := New(fakeParser{batch: Batch{SoftwareApplications: []metadata.SoftwareApplication{{SourceIRI: "app:1"}}}}, fakeAuthorizer{result: authorization.Result{Allowed: true}}, repository)

	_, err := service.Import(context.Background(), uuid.New(), strings.NewReader(""))
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Import() error = %v, want ErrInvalid", err)
	}
}
