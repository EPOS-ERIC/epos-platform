package discovery

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type discoveryRepositoryFake struct {
	candidates           []domain.DiscoveryCandidate
	filter               domain.DiscoveryFilter
	datasetCalls         int
	applicationCalls     int
	sourceCodeCalls      int
	datasetRelations     map[uuid.UUID]domain.DiscoveryRelations
	applicationRelations map[uuid.UUID]domain.DiscoveryRelations
	sourceCodeRelations  map[uuid.UUID]domain.DiscoveryRelations
}

func (r *discoveryRepositoryFake) Search(_ context.Context, filter domain.DiscoveryFilter) ([]domain.DiscoveryCandidate, error) {
	r.filter = filter
	return r.candidates, nil
}

func (r *discoveryRepositoryFake) DatasetRelations(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]domain.DiscoveryRelations, error) {
	r.datasetCalls++
	return r.datasetRelations, nil
}

func (r *discoveryRepositoryFake) SoftwareApplicationRelations(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]domain.DiscoveryRelations, error) {
	r.applicationCalls++
	return r.applicationRelations, nil
}

func (r *discoveryRepositoryFake) SoftwareSourceCodeRelations(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]domain.DiscoveryRelations, error) {
	r.sourceCodeCalls++
	return r.sourceCodeRelations, nil
}

type discoveryAuthorizerFake struct {
	authorization domain.DiscoveryAuthorization
	actor         *domain.Actor
	candidates    []domain.DiscoveryCandidate
	calls         int
}

func (a *discoveryAuthorizerFake) AuthorizeDiscovery(_ context.Context, actor *domain.Actor, candidates []domain.DiscoveryCandidate) (domain.DiscoveryAuthorization, error) {
	a.calls++
	a.actor = actor
	a.candidates = candidates
	return a.authorization, nil
}

func TestSearchUsesAuthorizedCandidatesAndBatchedRelations(t *testing.T) {
	t.Parallel()
	groupID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	datasetID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	applicationID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	sourceCodeID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	datasetVersionID := uuid.MustParse("00000000-0000-0000-0000-000000000011")
	applicationVersionID := uuid.MustParse("00000000-0000-0000-0000-000000000012")
	sourceCodeVersionID := uuid.MustParse("00000000-0000-0000-0000-000000000013")

	candidates := []domain.DiscoveryCandidate{
		discoveryCandidate(domain.Dataset, datasetID, datasetVersionID, groupID, "Same"),
		discoveryCandidate(domain.SoftwareApplication, applicationID, applicationVersionID, groupID, "same"),
		discoveryCandidate(domain.SoftwareSourceCode, sourceCodeID, sourceCodeVersionID, groupID, "Alpha"),
	}
	repository := &discoveryRepositoryFake{
		candidates: candidates,
		datasetRelations: map[uuid.UUID]domain.DiscoveryRelations{
			datasetVersionID: {
				Providers: []domain.DiscoveryReference{{ID: uuid.MustParse("00000000-0000-0000-0000-000000000021"), Name: "Provider"}},
			},
		},
		applicationRelations: map[uuid.UUID]domain.DiscoveryRelations{
			applicationVersionID: {
				Creators: []domain.DiscoveryReference{{ID: uuid.MustParse("00000000-0000-0000-0000-000000000022"), Name: "Creator"}},
			},
		},
		sourceCodeRelations: map[uuid.UUID]domain.DiscoveryRelations{
			sourceCodeVersionID: {
				Categories: []domain.DiscoveryReference{{ID: uuid.MustParse("00000000-0000-0000-0000-000000000023"), Name: "Category"}},
			},
		},
	}
	authorizer := &discoveryAuthorizerFake{authorization: domain.DiscoveryAuthorization{
		AllowedCandidates: candidates,
		CreatedByGroupIDs: map[uuid.UUID]struct{}{groupID: {}},
	}}
	actor := domain.Actor("user-1")
	items, err := New(slog.Default(), repository, authorizer).Search(context.Background(), &actor, domain.DiscoveryFilter{})
	if err != nil {
		t.Fatalf("search discovery: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}
	if items[0].Name != "Alpha" || items[1].Type != domain.Dataset || items[2].Type != domain.SoftwareApplication {
		t.Fatalf("items order = %#v", items)
	}
	if len(items[1].Providers) != 1 || items[1].Providers[0].Name != "Provider" {
		t.Fatalf("dataset providers = %#v", items[1].Providers)
	}
	if len(items[2].Creators) != 1 || items[2].Creators[0].Name != "Creator" {
		t.Fatalf("application creators = %#v", items[2].Creators)
	}
	if len(items[0].Categories) != 1 || items[0].Categories[0].Name != "Category" {
		t.Fatalf("source code categories = %#v", items[0].Categories)
	}
	if items[0].CreatedBy == nil || authorizer.calls != 1 {
		t.Fatalf("created by = %#v, authorization calls = %d", items[0].CreatedBy, authorizer.calls)
	}
	if repository.datasetCalls != 1 || repository.applicationCalls != 1 || repository.sourceCodeCalls != 1 {
		t.Fatalf("relation calls = dataset:%d application:%d source-code:%d", repository.datasetCalls, repository.applicationCalls, repository.sourceCodeCalls)
	}
}

func TestSearchProjectsOnlyAuthorizedCandidates(t *testing.T) {
	t.Parallel()
	groupID := uuid.New()
	allowed := discoveryCandidate(domain.Dataset, uuid.New(), uuid.New(), groupID, "Allowed")
	denied := discoveryCandidate(domain.SoftwareApplication, uuid.New(), uuid.New(), groupID, "Denied")
	repository := &discoveryRepositoryFake{candidates: []domain.DiscoveryCandidate{allowed, denied}}
	authorizer := &discoveryAuthorizerFake{authorization: domain.DiscoveryAuthorization{
		AllowedCandidates: []domain.DiscoveryCandidate{allowed},
	}}

	items, err := New(slog.Default(), repository, authorizer).Search(context.Background(), nil, domain.DiscoveryFilter{})
	if err != nil {
		t.Fatalf("search discovery: %v", err)
	}
	if len(items) != 1 || items[0].ResourceID != allowed.ResourceID {
		t.Fatalf("items = %#v", items)
	}
	if repository.applicationCalls != 0 || repository.datasetCalls != 1 {
		t.Fatalf("relation calls = dataset:%d application:%d", repository.datasetCalls, repository.applicationCalls)
	}
	if repository.filter.VersioningStatuses[0] != domain.Published {
		t.Fatalf("anonymous statuses = %#v", repository.filter.VersioningStatuses)
	}
	if items[0].CreatedBy != nil {
		t.Fatalf("anonymous created by = %#v", items[0].CreatedBy)
	}
}

func discoveryCandidate(typ domain.ResourceType, resourceID, versionID, groupID uuid.UUID, name string) domain.DiscoveryCandidate {
	return domain.DiscoveryCandidate{
		Type:       typ,
		ResourceID: resourceID,
		VersionID:  versionID,
		GroupID:    groupID,
		GroupName:  "Group",
		Status:     domain.Published,
		Name:       name,
		UpdatedAt:  time.Unix(0, 0),
		Concept:    domain.DiscoveryReference{ID: uuid.New(), Name: "Software"},
		CreatedBy:  &domain.DiscoveryCreatedBy{ID: "creator", Name: "Creator"},
	}
}
