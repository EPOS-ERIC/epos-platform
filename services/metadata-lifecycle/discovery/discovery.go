// Package discovery provides resource discovery for published and previewable resources.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/authz"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/datasetlifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/groups"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/softwarelifecycle"
	"github.com/google/uuid"
)

var (
	ErrInvalid = errors.New("invalid discovery search")
	ErrDenied  = errors.New("discovery preview requires authentication")
)

type ResourceType string

const (
	Dataset             ResourceType = "dataset"
	SoftwareApplication ResourceType = "software-application"
	SoftwareSourceCode  ResourceType = "software-source-code"
)

type Status string

const (
	Published Status = "PUBLISHED"
	Draft     Status = "DRAFT"
	Submitted Status = "SUBMITTED"
	Archived  Status = "ARCHIVED"
	Rejected  Status = "REJECTED"
)

type Filter struct {
	Query              string
	Types              []ResourceType
	SpatialFilter      string
	DateFrom           *time.Time
	DateTo             *time.Time
	Providers          []uuid.UUID
	Keywords           []string
	VersioningStatuses []Status
}

// TODO: use a domain enum!

func (f Filter) Validate(actorID *string) error {
	for _, status := range f.VersioningStatuses {
		if status != Published && actorID == nil {
			return fmt.Errorf("versioningStatus requires authentication: %w", ErrDenied)
		}
	}
	if f.DateFrom != nil && f.DateTo != nil && f.DateFrom.After(*f.DateTo) {
		return fmt.Errorf("dateFrom must not be after dateTo: %w", ErrInvalid)
	}
	return nil
}

type Candidate struct {
	Type       ResourceType
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	GroupID    uuid.UUID
	Status     Status
	Name       string
}

type Repository interface {
	Search(context.Context, Filter) ([]Candidate, error)
}

type DatasetReader interface {
	Read(context.Context, *string, uuid.UUID, uuid.UUID) (datasetlifecycle.Dataset, bool, error)
}

type ApplicationReader interface {
	Read(context.Context, *string, uuid.UUID, uuid.UUID) (softwarelifecycle.Application, bool, error)
}

type SourceCodeReader interface {
	Read(context.Context, *string, uuid.UUID, uuid.UUID) (softwarelifecycle.SourceCode, bool, error)
}

type GroupReader interface {
	Get(context.Context, uuid.UUID) (groups.Group, error)
}

type Authorizer interface {
	CanReadDatasetVersion(context.Context, *string, uuid.UUID, uuid.UUID) (authz.Result, error)
	CanReadSoftwareApplicationVersion(context.Context, *string, uuid.UUID, uuid.UUID) (authz.Result, error)
	CanReadSoftwareSourceCodeVersion(context.Context, *string, uuid.UUID, uuid.UUID) (authz.Result, error)
}

type Service struct {
	repository   Repository
	authorizer   Authorizer
	datasets     DatasetReader
	applications ApplicationReader
	sourceCodes  SourceCodeReader
	groups       GroupReader
}

func New(
	repository Repository,
	authorizer Authorizer,
	datasets DatasetReader,
	applications ApplicationReader,
	sourceCodes SourceCodeReader,
	groups GroupReader,
) *Service {
	return &Service{
		repository:   repository,
		authorizer:   authorizer,
		datasets:     datasets,
		applications: applications,
		sourceCodes:  sourceCodes,
		groups:       groups,
	}
}

type Item struct {
	Type          ResourceType `json:"type"`
	ResourceID    uuid.UUID    `json:"resource_id"`
	VersionID     uuid.UUID    `json:"version_id"`
	Name          string       `json:"name"`
	Status        Status       `json:"status"`
	Group         Reference    `json:"group"`
	Providers     []Reference  `json:"providers,omitempty"`
	Creators      []Reference  `json:"creators,omitempty"`
	Categories    []Reference  `json:"categories,omitempty"`
	Distributions []Reference  `json:"distributions,omitempty"`
}

type Reference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (s *Service) Search(ctx context.Context, actorID *string, filter Filter) ([]Item, error) {
	if err := filter.Validate(actorID); err != nil {
		return nil, err
	}

	candidates, err := s.repository.Search(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("search discovery: %w", err)
	}

	items := make([]Item, 0, len(candidates))
	for _, candidate := range candidates {
		allowed, err := s.canRead(ctx, actorID, candidate)
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}

		item, err := s.item(ctx, actorID, candidate)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	slices.SortFunc(items, func(left, right Item) int {
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

func (s *Service) canRead(ctx context.Context, actorID *string, candidate Candidate) (bool, error) {
	var result authz.Result
	var err error
	switch candidate.Type {
	case Dataset:
		result, err = s.authorizer.CanReadDatasetVersion(ctx, actorID, candidate.ResourceID, candidate.VersionID)
	case SoftwareApplication:
		result, err = s.authorizer.CanReadSoftwareApplicationVersion(ctx, actorID, candidate.ResourceID, candidate.VersionID)
	case SoftwareSourceCode:
		result, err = s.authorizer.CanReadSoftwareSourceCodeVersion(ctx, actorID, candidate.ResourceID, candidate.VersionID)
	default:
		return false, fmt.Errorf("unsupported resource type %q: %w", candidate.Type, ErrInvalid)
	}
	if err != nil {
		return false, fmt.Errorf("authorize discovery item: %w", err)
	}
	return result.Allowed, nil
}

func (s *Service) item(ctx context.Context, actorID *string, candidate Candidate) (Item, error) {
	group, err := s.groups.Get(ctx, candidate.GroupID)
	if err != nil {
		return Item{}, fmt.Errorf("get discovery group: %w", err)
	}
	item := Item{
		Type:       candidate.Type,
		ResourceID: candidate.ResourceID,
		VersionID:  candidate.VersionID,
		Name:       candidate.Name,
		Status:     candidate.Status,
		Group: Reference{
			ID:   group.ID,
			Name: group.Name,
		},
		Providers:     nil,
		Creators:      nil,
		Categories:    nil,
		Distributions: nil,
	}
	switch candidate.Type {
	case Dataset:
		value, _, err := s.datasets.Read(ctx, actorID, candidate.ResourceID, candidate.VersionID)
		if err != nil {
			return Item{}, fmt.Errorf("read discovery dataset: %w", err)
		}
		item.Providers = organizations(value.Metadata.Providers)
		item.Categories = categories(value.Metadata.Categories)
		item.Distributions = distributions(value.Metadata.Distributions)
	case SoftwareApplication:
		value, _, err := s.applications.Read(ctx, actorID, candidate.ResourceID, candidate.VersionID)
		if err != nil {
			return Item{}, fmt.Errorf("read discovery application: %w", err)
		}
		item.Creators = organizations(value.Metadata.Creators)
		item.Categories = categories(value.Metadata.Categories)
	case SoftwareSourceCode:
		value, _, err := s.sourceCodes.Read(ctx, actorID, candidate.ResourceID, candidate.VersionID)
		if err != nil {
			return Item{}, fmt.Errorf("read discovery source code: %w", err)
		}
		item.Creators = organizations(value.Metadata.Creators)
		item.Categories = categories(value.Metadata.Categories)
	}
	return item, nil
}

func organizations(values []metadata.Organization) []Reference {
	result := make([]Reference, 0, len(values))
	for _, value := range values {
		result = append(result, Reference{
			ID:   value.ID,
			Name: value.Name,
		})
	}
	return result
}

func categories(values []metadata.Category) []Reference {
	result := make([]Reference, 0, len(values))
	for _, value := range values {
		result = append(result, Reference{
			ID:   value.ID,
			Name: value.Name,
		})
	}
	return result
}

func distributions(values []metadata.Distribution) []Reference {
	result := make([]Reference, 0, len(values))
	for _, value := range values {
		result = append(result, Reference{
			ID:   value.ID,
			Name: value.Title,
		})
	}
	return result
}
