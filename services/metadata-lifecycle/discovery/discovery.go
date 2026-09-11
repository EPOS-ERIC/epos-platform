// Package discovery provides resource discovery for published and previewable resources.
package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type Repository interface {
	Search(context.Context, domain.DiscoveryFilter) ([]domain.DiscoveryCandidate, error)
	DatasetRelations(context.Context, []uuid.UUID) (map[uuid.UUID]domain.DiscoveryRelations, error)
	SoftwareApplicationRelations(context.Context, []uuid.UUID) (map[uuid.UUID]domain.DiscoveryRelations, error)
	SoftwareSourceCodeRelations(context.Context, []uuid.UUID) (map[uuid.UUID]domain.DiscoveryRelations, error)
}

type Authorizer interface {
	AuthorizeDiscovery(context.Context, *domain.Actor, []domain.DiscoveryCandidate) (domain.DiscoveryAuthorization, error)
}

type Service struct {
	repository Repository
	authorizer Authorizer
	log        *slog.Logger
}

// New creates a discovery service backed by batched authorization and projections.
func New(
	log *slog.Logger,
	repository Repository,
	authorizer Authorizer,
) *Service {
	return &Service{
		repository: repository,
		authorizer: authorizer,
		log:        log.With("component", "discovery-service"),
	}
}

// Search returns sorted discovery items visible to actor.
func (s *Service) Search(ctx context.Context, actor *domain.Actor, filter domain.DiscoveryFilter) ([]domain.DiscoveryItem, error) {
	if actor == nil {
		filter.VersioningStatuses = []domain.Status{domain.Published}
	}

	candidates, err := s.repository.Search(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("search discovery: %w", err)
	}

	s.log.DebugContext(ctx, "discovery search", "candidates", len(candidates))
	authorization, err := s.authorizer.AuthorizeDiscovery(ctx, actor, candidates)
	if err != nil {
		return nil, fmt.Errorf("authorize discovery: %w", err)
	}
	relations, err := s.loadRelations(ctx, authorization.AllowedCandidates)
	if err != nil {
		return nil, err
	}

	items := make([]domain.DiscoveryItem, 0, len(authorization.AllowedCandidates))
	for _, candidate := range authorization.AllowedCandidates {
		item, err := s.item(candidate, relations[candidate.VersionID], authorization.IncludeCreatedBy)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	s.log.DebugContext(ctx, "discovery search", "items filtered", len(items))

	slices.SortFunc(items, func(left, right domain.DiscoveryItem) int {
		leftName, rightName := strings.ToLower(left.Name), strings.ToLower(right.Name)
		if leftName != rightName {
			return strings.Compare(leftName, rightName)
		}
		if left.Type != right.Type {
			return strings.Compare(string(left.Type), string(right.Type))
		}
		return strings.Compare(left.ResourceID.String(), right.ResourceID.String())
	})

	return items, nil
}

// loadRelations fetches only discovery relationships for authorized candidates.
func (s *Service) loadRelations(ctx context.Context, candidates []domain.DiscoveryCandidate) (map[uuid.UUID]domain.DiscoveryRelations, error) {
	result := make(map[uuid.UUID]domain.DiscoveryRelations, len(candidates))
	datasetVersionIDs := make([]uuid.UUID, 0, len(candidates))
	applicationVersionIDs := make([]uuid.UUID, 0, len(candidates))
	sourceCodeVersionIDs := make([]uuid.UUID, 0, len(candidates))
	for _, candidate := range candidates {
		switch candidate.Type {
		case domain.Dataset:
			datasetVersionIDs = append(datasetVersionIDs, candidate.VersionID)
		case domain.SoftwareApplication:
			applicationVersionIDs = append(applicationVersionIDs, candidate.VersionID)
		case domain.SoftwareSourceCode:
			sourceCodeVersionIDs = append(sourceCodeVersionIDs, candidate.VersionID)
		default:
			return nil, fmt.Errorf("unsupported resource type %q: %w", candidate.Type, domain.ErrInvalidDiscovery)
		}
	}

	if len(datasetVersionIDs) > 0 {
		relations, err := s.repository.DatasetRelations(ctx, datasetVersionIDs)
		if err != nil {
			return nil, fmt.Errorf("load discovery dataset relations: %w", err)
		}
		maps.Copy(result, relations)
	}
	if len(applicationVersionIDs) > 0 {
		relations, err := s.repository.SoftwareApplicationRelations(ctx, applicationVersionIDs)
		if err != nil {
			return nil, fmt.Errorf("load discovery application relations: %w", err)
		}
		maps.Copy(result, relations)
	}
	if len(sourceCodeVersionIDs) > 0 {
		relations, err := s.repository.SoftwareSourceCodeRelations(ctx, sourceCodeVersionIDs)
		if err != nil {
			return nil, fmt.Errorf("load discovery source code relations: %w", err)
		}
		maps.Copy(result, relations)
	}

	return result, nil
}

// item combines a candidate and its already-loaded relationships into an API item.
func (s *Service) item(candidate domain.DiscoveryCandidate, relations domain.DiscoveryRelations, includeCreatedBy bool) (domain.DiscoveryItem, error) {
	groupReference, err := domain.NewDiscoveryReference(candidate.GroupID, candidate.GroupName)
	if err != nil {
		return domain.DiscoveryItem{}, fmt.Errorf("construct discovery group: %w", err)
	}
	item, err := domain.NewDiscoveryItem(candidate, groupReference, relations.Providers, relations.Creators, relations.Categories, relations.Distributions, includeCreatedBy)
	if err != nil {
		return domain.DiscoveryItem{}, fmt.Errorf("construct discovery item: %w", err)
	}
	return item, nil
}
