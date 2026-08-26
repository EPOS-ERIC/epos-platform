package datasetlifecycle

import (
	"context"
	"errors"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/authz"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/google/uuid"
)

type testAuthorizer struct {
	allowed    bool
	permission string
}

func (a *testAuthorizer) AuthorizeGroup(_ context.Context, _ string, _ uuid.UUID, permission string) (authz.Result, error) {
	a.permission = permission
	return authz.Result{Allowed: a.allowed, Reason: "denied"}, nil
}

func (a *testAuthorizer) AuthorizeResource(_ context.Context, _ string, _ authz.ResourceType, _ uuid.UUID, permission string) (authz.Result, error) {
	a.permission = permission
	return authz.Result{Allowed: a.allowed, Reason: "denied"}, nil
}

func (a *testAuthorizer) CanReadResource(_ context.Context, _ *string, _ authz.ResourceType, _ uuid.UUID) (authz.Result, error) {
	return authz.Result{Allowed: a.allowed, Reason: "denied"}, nil
}

func (a *testAuthorizer) CanReadDatasetVersion(context.Context, *string, uuid.UUID, uuid.UUID) (authz.Result, error) {
	return authz.Result{Allowed: a.allowed, Reason: "denied"}, nil
}

func (a *testAuthorizer) CanReadContactPoint(context.Context, string) (authz.Result, error) {
	return authz.Result{Allowed: a.allowed, Reason: "denied"}, nil
}

type testRepository struct {
	create func(metadata.Dataset) (Dataset, error)
	note   string
}

func (*testRepository) Get(context.Context, uuid.UUID, uuid.UUID) (Dataset, error) {
	return Dataset{}, nil
}

func (*testRepository) GetPublished(context.Context, uuid.UUID) (Dataset, error) {
	return Dataset{}, nil
}

func (*testRepository) GetActive(context.Context, uuid.UUID) (Dataset, error) {
	return Dataset{}, ErrNotFound
}

func (r *testRepository) CreateDraft(_ context.Context, _ string, _ uuid.UUID, v metadata.Dataset) (Dataset, error) {
	return r.create(v)
}

func (*testRepository) ReplaceDraft(context.Context, uuid.UUID, uuid.UUID, int32, metadata.Dataset) (Dataset, error) {
	return Dataset{}, nil
}

func (*testRepository) Submit(context.Context, string, uuid.UUID, uuid.UUID) (Dataset, error) {
	return Dataset{}, nil
}

func (*testRepository) Withdraw(context.Context, uuid.UUID, uuid.UUID) (Dataset, error) {
	return Dataset{}, nil
}

func (r *testRepository) Reject(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID, note string) (Dataset, error) {
	r.note = note
	return Dataset{}, nil
}

func (*testRepository) Reopen(context.Context, uuid.UUID, uuid.UUID) (Dataset, error) {
	return Dataset{}, nil
}

func (*testRepository) Publish(context.Context, string, uuid.UUID, uuid.UUID) (Dataset, error) {
	return Dataset{}, nil
}
func (*testRepository) Discard(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (*testRepository) Unpublish(context.Context, string, uuid.UUID, uuid.UUID) (Dataset, error) {
	return Dataset{}, nil
}

func (*testRepository) DraftFromPublication(context.Context, string, uuid.UUID) (Dataset, error) {
	return Dataset{}, nil
}

func (*testRepository) DraftFromLatestArchive(context.Context, string, uuid.UUID) (Dataset, error) {
	return Dataset{}, nil
}

func (*testRepository) CopyPublished(context.Context, string, uuid.UUID, uuid.UUID, string) (Dataset, error) {
	return Dataset{}, nil
}

func TestCreateDraftTrimsSourceBeforePersistence(t *testing.T) {
	auth := &testAuthorizer{allowed: true}
	repo := &testRepository{create: func(v metadata.Dataset) (Dataset, error) {
		if v.SourceIRI != "https://example.test/dataset" {
			t.Fatalf("source IRI = %q", v.SourceIRI)
		}
		return Dataset{}, nil
	}}
	_, err := New(auth, repo).CreateDraft(context.Background(), "user-1", uuid.New(), metadata.Dataset{SourceIRI: " https://example.test/dataset "})
	if err != nil {
		t.Fatal(err)
	}
	if auth.permission != permissionCreate {
		t.Fatalf("permission = %q", auth.permission)
	}
}

func TestRejectRequiresNoteBeforeAuthorization(t *testing.T) {
	auth := &testAuthorizer{allowed: true}
	repo := &testRepository{}
	_, err := New(auth, repo).Reject(context.Background(), "user-1", uuid.New(), uuid.New(), " \t")
	if err == nil || auth.permission != "" {
		t.Fatalf("err = %v, authorization = %q", err, auth.permission)
	}
}

func TestRejectTrimsNote(t *testing.T) {
	auth := &testAuthorizer{allowed: true}
	repo := &testRepository{}
	_, err := New(auth, repo).Reject(context.Background(), "user-1", uuid.New(), uuid.New(), " no metadata ")
	if err != nil {
		t.Fatal(err)
	}
	if repo.note != "no metadata" {
		t.Fatalf("note = %q", repo.note)
	}
}

func TestDeniedActionDoesNotReachRepository(t *testing.T) {
	auth := &testAuthorizer{}
	repo := &testRepository{create: func(metadata.Dataset) (Dataset, error) { t.Fatal("repository called"); return Dataset{}, nil }}
	_, err := New(auth, repo).CreateDraft(context.Background(), "user-1", uuid.New(), metadata.Dataset{SourceIRI: "x"})
	if !errors.Is(err, ErrDenied) {
		t.Fatalf("err = %v", err)
	}
}
