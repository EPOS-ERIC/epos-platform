package domain

import (
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"
)

var ErrInvalidDiscovery = errors.New("invalid discovery")

// NewDiscoveryFilter validates and normalizes discovery search criteria.
func NewDiscoveryFilter(value DiscoveryFilter) (DiscoveryFilter, error) {
	value.Query = normalizedOptionalString(value.Query)
	value.SpatialFilter = normalizedOptionalString(value.SpatialFilter)

	if value.Types != nil {
		types := make([]ResourceType, 0, len(value.Types))
		for _, resourceType := range value.Types {
			resourceType = ResourceType(strings.TrimSpace(string(resourceType)))
			if resourceType == "" {
				continue
			}
			if !validResourceType(resourceType) {
				return DiscoveryFilter{}, ErrInvalidDiscovery
			}
			if !slices.Contains(types, resourceType) {
				types = append(types, resourceType)
			}
		}
		value.Types = types
	}

	statuses := make([]Status, 0, len(value.VersioningStatuses))
	for _, status := range value.VersioningStatuses {
		status = Status(strings.ToUpper(strings.TrimSpace(string(status))))
		if status == "" {
			continue
		}
		if !validStatus(status) {
			return DiscoveryFilter{}, ErrInvalidDiscovery
		}
		if !slices.Contains(statuses, status) {
			statuses = append(statuses, status)
		}
	}
	if len(statuses) == 0 {
		statuses = []Status{Published}
	}
	value.VersioningStatuses = statuses

	if slices.Contains(value.Providers, uuid.Nil) {
		return DiscoveryFilter{}, ErrInvalidDiscovery
	}
	value.Providers = slices.Clone(value.Providers)
	value.Keywords = normalizedStrings(value.Keywords)

	if value.DateFrom != nil && value.DateTo != nil && value.DateFrom.After(*value.DateTo) {
		return DiscoveryFilter{}, ErrInvalidDiscovery
	}

	return value, nil
}

// NewDiscoveryCandidate validates a persisted discovery result.
func NewDiscoveryCandidate(value DiscoveryCandidate) (DiscoveryCandidate, error) {
	value.Name = strings.TrimSpace(value.Name)
	if value.ResourceID == uuid.Nil || value.VersionID == uuid.Nil || value.GroupID == uuid.Nil {
		return DiscoveryCandidate{}, ErrInvalidDiscovery
	}
	if !validResourceType(value.Type) || !validStatus(value.Status) || value.Name == "" {
		return DiscoveryCandidate{}, ErrInvalidDiscovery
	}
	return value, nil
}

// NewDiscoveryReference creates a valid relationship projection.
func NewDiscoveryReference(id uuid.UUID, name string) (DiscoveryReference, error) {
	if id == uuid.Nil {
		return DiscoveryReference{}, ErrInvalidDiscovery
	}
	return DiscoveryReference{ID: id, Name: name}, nil
}

// NewDiscoveryItem assembles a validated discovery item from its candidate and relationships.
func NewDiscoveryItem(
	candidate DiscoveryCandidate,
	group DiscoveryReference,
	providers, creators, categories, distributions []DiscoveryReference,
) (DiscoveryItem, error) {
	validatedCandidate, err := NewDiscoveryCandidate(candidate)
	if err != nil {
		return DiscoveryItem{}, err
	}
	validatedGroup, err := NewDiscoveryReference(group.ID, group.Name)
	if err != nil {
		return DiscoveryItem{}, err
	}
	validatedProviders, err := discoveryReferences(providers)
	if err != nil {
		return DiscoveryItem{}, err
	}
	validatedCreators, err := discoveryReferences(creators)
	if err != nil {
		return DiscoveryItem{}, err
	}
	validatedCategories, err := discoveryReferences(categories)
	if err != nil {
		return DiscoveryItem{}, err
	}
	validatedDistributions, err := discoveryReferences(distributions)
	if err != nil {
		return DiscoveryItem{}, err
	}

	return DiscoveryItem{
		Type:          validatedCandidate.Type,
		ResourceID:    validatedCandidate.ResourceID,
		VersionID:     validatedCandidate.VersionID,
		Name:          validatedCandidate.Name,
		Status:        validatedCandidate.Status,
		Group:         validatedGroup,
		Providers:     validatedProviders,
		Creators:      validatedCreators,
		Categories:    validatedCategories,
		Distributions: validatedDistributions,
	}, nil
}

func discoveryReferences(values []DiscoveryReference) ([]DiscoveryReference, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]DiscoveryReference, 0, len(values))
	for _, value := range values {
		reference, err := NewDiscoveryReference(value.ID, value.Name)
		if err != nil {
			return nil, err
		}
		result = append(result, reference)
	}
	return result, nil
}

func validResourceType(value ResourceType) bool {
	switch value {
	case Dataset, SoftwareApplication, SoftwareSourceCode:
		return true
	default:
		return false
	}
}

func validStatus(value Status) bool {
	switch value {
	case Published, Draft, Submitted, Archived, Rejected:
		return true
	default:
		return false
	}
}

func normalizedStrings(values []string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func normalizedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
