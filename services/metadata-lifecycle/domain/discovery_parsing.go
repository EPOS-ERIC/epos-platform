package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkt"
)

// NewDiscoveryFilter validates and normalizes discovery search criteria.
func NewDiscoveryFilter(value DiscoveryFilter) (DiscoveryFilter, error) {
	value.Query = normalizedOptionalString(value.Query)
	value.EditorID = normalizedOptionalString(value.EditorID)
	if err := validateSpatialFilter(value.SpatialFilter); err != nil {
		return DiscoveryFilter{}, err
	}
	concepts := make([]TopLevelConcept, 0, len(value.Concepts))
	for _, concept := range value.Concepts {
		concept = TopLevelConcept(strings.ToUpper(strings.TrimSpace(string(concept))))
		if concept == "" {
			continue
		}
		if !concept.Valid() {
			return DiscoveryFilter{}, ErrInvalidDiscovery
		}
		if !slices.Contains(concepts, concept) {
			concepts = append(concepts, concept)
		}
	}
	value.Concepts = concepts
	if value.GroupID != nil && *value.GroupID == uuid.Nil {
		return DiscoveryFilter{}, ErrInvalidDiscovery
	}

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
	providers := make([]uuid.UUID, len(value.Providers))
	copy(providers, value.Providers)
	value.Providers = providers
	value.Keywords = normalizedStrings(value.Keywords)

	if value.DateFrom != nil && value.DateTo != nil && value.DateFrom.After(*value.DateTo) {
		return DiscoveryFilter{}, ErrInvalidDiscovery
	}

	return value, nil
}

// NewSpatialFilter parses and validates a WKT POINT or POLYGON filter.
func NewSpatialFilter(value string) (*SpatialFilter, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	geometry, err := wkt.Unmarshal(value)
	if err != nil {
		return nil, fmt.Errorf("invalid spatial filter: %w: %w", err, ErrInvalidDiscovery)
	}

	filter := &SpatialFilter{
		Point:   nil,
		Polygon: nil,
	}
	switch geometry := geometry.(type) {
	case *geom.Point:
		filter.Point = &Coordinate{Longitude: geometry.X(), Latitude: geometry.Y()}
	case *geom.Polygon:
		rings := geometry.Coords()
		if len(rings) != 1 {
			return nil, fmt.Errorf("spatial filter polygon must have one ring: %w", ErrInvalidDiscovery)
		}
		filter.Polygon = make([]Coordinate, len(rings[0]))
		for i, coordinate := range rings[0] {
			filter.Polygon[i] = Coordinate{Longitude: coordinate.X(), Latitude: coordinate.Y()}
		}
	default:
		return nil, fmt.Errorf("spatial filter must be POINT or POLYGON: %w", ErrInvalidDiscovery)
	}

	if err := validateSpatialFilter(filter); err != nil {
		return nil, err
	}
	return filter, nil
}

func validateSpatialFilter(value *SpatialFilter) error {
	if value == nil {
		return nil
	}
	if value.Point != nil && len(value.Polygon) != 0 {
		return fmt.Errorf("spatial filter cannot contain both point and polygon: %w", ErrInvalidDiscovery)
	}
	if value.Point != nil {
		if err := validateCoordinate(*value.Point); err != nil {
			return fmt.Errorf("spatial filter point: %w: %w", err, ErrInvalidDiscovery)
		}
		return nil
	}
	if len(value.Polygon) < 4 {
		return fmt.Errorf("spatial filter polygon must contain at least four coordinates: %w", ErrInvalidDiscovery)
	}
	for _, coordinate := range value.Polygon {
		if err := validateCoordinate(coordinate); err != nil {
			return fmt.Errorf("spatial filter polygon: %w: %w", err, ErrInvalidDiscovery)
		}
	}
	if value.Polygon[0] != value.Polygon[len(value.Polygon)-1] {
		return fmt.Errorf("spatial filter polygon must be closed: %w", ErrInvalidDiscovery)
	}
	return nil
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
