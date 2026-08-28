// Package discovery provides resource discovery for published and previewable resources.
package discovery

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/authz"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/datasetlifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwarelifecycle"
	"github.com/google/uuid"
)

type Repository interface {
	Search(context.Context, domain.DiscoveryFilter) ([]domain.DiscoveryCandidate, error)
}

type DatasetReader interface {
	Read(context.Context, *string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, bool, error)
}

type SoftwareApplicationReader interface {
	Read(context.Context, *string, uuid.UUID, uuid.UUID) (softwarelifecycle.Application, bool, error)
}

type SoftwareSourceCodeReader interface {
	Read(context.Context, *string, uuid.UUID, uuid.UUID) (softwarelifecycle.SourceCode, bool, error)
}

type GroupReader interface {
	Get(context.Context, uuid.UUID) (domain.Group, error)
}

type Authorizer interface {
	CanReadDatasetVersion(context.Context, *string, uuid.UUID, uuid.UUID) (authz.Result, error)
	CanReadSoftwareApplicationVersion(context.Context, *string, uuid.UUID, uuid.UUID) (authz.Result, error)
	CanReadSoftwareSourceCodeVersion(context.Context, *string, uuid.UUID, uuid.UUID) (authz.Result, error)
}

type Service struct {
	repository           Repository
	authorizer           Authorizer
	datasets             DatasetReader
	sotfwareApplications SoftwareApplicationReader
	softwareSourceCodes  SoftwareSourceCodeReader
	groups               GroupReader
}

func New(
	repository Repository,
	authorizer Authorizer,
	datasets DatasetReader,
	applications SoftwareApplicationReader,
	sourceCodes SoftwareSourceCodeReader,
	groups GroupReader,
) *Service {
	return &Service{
		repository:           repository,
		authorizer:           authorizer,
		datasets:             datasets,
		sotfwareApplications: applications,
		softwareSourceCodes:  sourceCodes,
		groups:               groups,
	}
}

func (s *Service) Search(ctx context.Context, actor *domain.Actor, filter domain.DiscoveryFilter) ([]domain.DiscoveryItem, error) {
	for _, status := range filter.VersioningStatuses {
		if status != domain.Published && actor == nil {
			return nil, fmt.Errorf("versioningStatus requires authentication: %w", domain.ErrDiscoveryDenied)
		}
	}

	candidates, err := s.repository.Search(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("search discovery: %w", err)
	}

	items := make([]domain.DiscoveryItem, 0, len(candidates))
	for _, candidate := range candidates {
		allowed, err := s.canRead(ctx, actor, candidate)
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}

		item, err := s.item(ctx, actor, candidate)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

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

func (s *Service) canRead(ctx context.Context, actor *domain.Actor, candidate domain.DiscoveryCandidate) (bool, error) {
	actorID := actor.OptionalString()
	var result authz.Result
	var err error
	switch candidate.Type {
	case domain.Dataset:
		result, err = s.authorizer.CanReadDatasetVersion(ctx, actorID, candidate.ResourceID, candidate.VersionID)
	case domain.SoftwareApplication:
		result, err = s.authorizer.CanReadSoftwareApplicationVersion(ctx, actorID, candidate.ResourceID, candidate.VersionID)
	case domain.SoftwareSourceCode:
		result, err = s.authorizer.CanReadSoftwareSourceCodeVersion(ctx, actorID, candidate.ResourceID, candidate.VersionID)
	default:
		return false, fmt.Errorf("unsupported resource type %q: %w", candidate.Type, domain.ErrInvalidDiscovery)
	}
	if err != nil {
		return false, fmt.Errorf("authorize discovery item: %w", err)
	}
	return result.Allowed, nil
}

func (s *Service) item(ctx context.Context, actor *domain.Actor, candidate domain.DiscoveryCandidate) (domain.DiscoveryItem, error) {
	group, err := s.groups.Get(ctx, candidate.GroupID)
	if err != nil {
		return domain.DiscoveryItem{}, fmt.Errorf("get discovery group: %w", err)
	}
	groupReference, err := domain.NewDiscoveryReference(group.ID, group.Name)
	if err != nil {
		return domain.DiscoveryItem{}, fmt.Errorf("construct discovery group: %w", err)
	}

	var providers []domain.DiscoveryReference
	var creators []domain.DiscoveryReference
	var categoryReferences []domain.DiscoveryReference
	var distributionReferences []domain.DiscoveryReference

	actorID := actor.OptionalString()
	switch candidate.Type {
	case domain.Dataset:
		value, _, err := s.datasets.Read(ctx, actorID, candidate.ResourceID, candidate.VersionID)
		if err != nil {
			return domain.DiscoveryItem{}, fmt.Errorf("read discovery dataset: %w", err)
		}
		providers, err = organizationReferences(value.Metadata.Providers)
		if err != nil {
			return domain.DiscoveryItem{}, fmt.Errorf("map discovery dataset providers: %w", err)
		}
		categoryReferences, err = categoryReferencesFor(value.Metadata.Categories)
		if err != nil {
			return domain.DiscoveryItem{}, fmt.Errorf("map discovery dataset categories: %w", err)
		}
		distributionReferences, err = distributionReferencesFor(value.Metadata.Distributions)
		if err != nil {
			return domain.DiscoveryItem{}, fmt.Errorf("map discovery dataset distributions: %w", err)
		}
	case domain.SoftwareApplication:
		value, _, err := s.sotfwareApplications.Read(ctx, actorID, candidate.ResourceID, candidate.VersionID)
		if err != nil {
			return domain.DiscoveryItem{}, fmt.Errorf("read discovery application: %w", err)
		}
		creators, err = organizationReferences(value.Metadata.Creators)
		if err != nil {
			return domain.DiscoveryItem{}, fmt.Errorf("map discovery application creators: %w", err)
		}
		categoryReferences, err = categoryReferencesFor(value.Metadata.Categories)
		if err != nil {
			return domain.DiscoveryItem{}, fmt.Errorf("map discovery application categories: %w", err)
		}
	case domain.SoftwareSourceCode:
		value, _, err := s.softwareSourceCodes.Read(ctx, actorID, candidate.ResourceID, candidate.VersionID)
		if err != nil {
			return domain.DiscoveryItem{}, fmt.Errorf("read discovery source code: %w", err)
		}
		creators, err = organizationReferences(value.Metadata.Creators)
		if err != nil {
			return domain.DiscoveryItem{}, fmt.Errorf("map discovery source code creators: %w", err)
		}
		categoryReferences, err = categoryReferencesFor(value.Metadata.Categories)
		if err != nil {
			return domain.DiscoveryItem{}, fmt.Errorf("map discovery source code categories: %w", err)
		}
	}

	item, err := domain.NewDiscoveryItem(candidate, groupReference, providers, creators, categoryReferences, distributionReferences)
	if err != nil {
		return domain.DiscoveryItem{}, fmt.Errorf("construct discovery item: %w", err)
	}
	return item, nil
}

func organizationReferences(values []metadata.Organization) ([]domain.DiscoveryReference, error) {
	result := make([]domain.DiscoveryReference, 0, len(values))
	for _, value := range values {
		reference, err := domain.NewDiscoveryReference(value.ID, value.Name)
		if err != nil {
			return nil, fmt.Errorf("construct organization reference: %w", err)
		}
		result = append(result, reference)
	}
	return result, nil
}

func categoryReferencesFor(values []metadata.Category) ([]domain.DiscoveryReference, error) {
	result := make([]domain.DiscoveryReference, 0, len(values))
	for _, value := range values {
		reference, err := domain.NewDiscoveryReference(value.ID, value.Name)
		if err != nil {
			return nil, fmt.Errorf("construct category reference: %w", err)
		}
		result = append(result, reference)
	}
	return result, nil
}

func distributionReferencesFor(values []metadata.Distribution) ([]domain.DiscoveryReference, error) {
	// TODO: don't need the ids of the distributions here it could be just a list with the names
	result := make([]domain.DiscoveryReference, 0, len(values))
	for _, value := range values {
		reference, err := domain.NewDiscoveryReference(value.ID, value.Title)
		if err != nil {
			return nil, fmt.Errorf("construct distribution reference: %w", err)
		}
		result = append(result, reference)
	}
	return result, nil
}
