package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkt"
)

// NewDiscoveryFilter validates and normalizes discovery search criteria.
func NewDiscoveryFilter(value DiscoveryFilter) (DiscoveryFilter, error) {
	value.Query = normalizedOptionalString(value.Query)
	value.EditorID = normalizedOptionalString(value.EditorID)
	concepts := make([]TopLevelConcept, 0, len(value.Concepts))
	for _, concept := range value.Concepts {
		concept = TopLevelConcept(strings.ToUpper(strings.TrimSpace(string(concept))))
		if concept == "" {
			continue
		}
		if !concept.Valid() {
			return DiscoveryFilter{}, apperror.Default(apperror.CodeValidationFailed)
		}
		if !slices.Contains(concepts, concept) {
			concepts = append(concepts, concept)
		}
	}
	value.Concepts = concepts
	if value.GroupID != nil && *value.GroupID == uuid.Nil {
		return DiscoveryFilter{}, apperror.Default(apperror.CodeValidationFailed)
	}

	types := make([]ResourceType, 0, len(value.Types))
	for _, resourceType := range value.Types {
		resourceType = ResourceType(strings.TrimSpace(string(resourceType)))
		if resourceType == "" {
			continue
		}
		if !validResourceType(resourceType) {
			return DiscoveryFilter{}, apperror.Default(apperror.CodeValidationFailed)
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
			return DiscoveryFilter{}, apperror.Default(apperror.CodeValidationFailed)
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
		return DiscoveryFilter{}, apperror.Default(apperror.CodeValidationFailed)
	}
	providers := make([]uuid.UUID, len(value.Providers))
	copy(providers, value.Providers)
	value.Providers = providers
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return DiscoveryFilter{}, fmt.Errorf("discovery keywords: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	value.Keywords = keywords

	if value.DateFrom != nil && value.DateTo != nil && value.DateFrom.After(*value.DateTo) {
		return DiscoveryFilter{}, apperror.Default(apperror.CodeValidationFailed)
	}

	return value, nil
}

// NewSpatialFilter parses a WKT spatial filter.
func NewSpatialFilter(value string) (geom.T, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	geometry, err := wkt.Unmarshal(value)
	if err != nil {
		return nil, fmt.Errorf("invalid spatial filter: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	geometry, err = geom.SetSRID(geometry, 4326)
	if err != nil {
		return nil, fmt.Errorf("set spatial filter SRID: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}

	return geometry, nil
}

// NewDiscoveryCandidate validates a persisted discovery result.
func NewDiscoveryCandidate(value DiscoveryCandidate) (DiscoveryCandidate, error) {
	value.Name = strings.TrimSpace(value.Name)
	if value.ResourceID == uuid.Nil || value.VersionID == uuid.Nil || value.GroupID == uuid.Nil {
		return DiscoveryCandidate{}, apperror.Default(apperror.CodeInternal)
	}
	if !validResourceType(value.Type) || !validStatus(value.Status) || value.Name == "" {
		return DiscoveryCandidate{}, apperror.Default(apperror.CodeInternal)
	}
	if value.CreatedBy != nil {
		createdBy, err := NewDiscoveryCreatedBy(value.CreatedBy.ID, value.CreatedBy.Name)
		if err != nil {
			return DiscoveryCandidate{}, err
		}
		value.CreatedBy = &createdBy
	}
	return value, nil
}

// NewDiscoveryReference creates a valid relationship projection.
func NewDiscoveryReference(id uuid.UUID, name string) (DiscoveryReference, error) {
	if id == uuid.Nil {
		return DiscoveryReference{}, apperror.Default(apperror.CodeInternal)
	}
	return DiscoveryReference{ID: id, Name: name}, nil
}

// NewDiscoveryCreatedBy creates a valid creator projection.
func NewDiscoveryCreatedBy(id, name string) (DiscoveryCreatedBy, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return DiscoveryCreatedBy{}, apperror.Default(apperror.CodeInternal)
	}
	return DiscoveryCreatedBy{ID: id, Name: name}, nil
}

// NewDiscoveryItem assembles a validated discovery item from its candidate and relationships.
func NewDiscoveryItem(
	candidate DiscoveryCandidate,
	group DiscoveryReference,
	providers, creators, categories []DiscoveryReference,
	distributions []DiscoveryDistribution,
	includeCreatedBy bool,
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
	validatedDistributions := make([]DiscoveryDistribution, 0, len(distributions))
	for _, distribution := range distributions {
		validated, err := NewDiscoveryDistribution(distribution)
		if err != nil {
			return DiscoveryItem{}, err
		}
		validatedDistributions = append(validatedDistributions, validated)
	}
	var validatedCreatedBy *DiscoveryCreatedBy
	if includeCreatedBy && validatedCandidate.CreatedBy != nil {
		createdBy, err := NewDiscoveryCreatedBy(validatedCandidate.CreatedBy.ID, validatedCandidate.CreatedBy.Name)
		if err != nil {
			return DiscoveryItem{}, err
		}
		validatedCreatedBy = &createdBy
	}

	return DiscoveryItem{
		Type:          validatedCandidate.Type,
		ResourceID:    validatedCandidate.ResourceID,
		VersionID:     validatedCandidate.VersionID,
		Name:          validatedCandidate.Name,
		Status:        validatedCandidate.Status,
		UpdatedAt:     validatedCandidate.UpdatedAt,
		Concept:       validatedCandidate.Concept,
		CreatedBy:     validatedCreatedBy,
		Group:         validatedGroup,
		Providers:     validatedProviders,
		Creators:      validatedCreators,
		Categories:    validatedCategories,
		Distributions: validatedDistributions,
	}, nil
}

// NewDiscoveryDistribution validates a persisted distribution summary.
func NewDiscoveryDistribution(value DiscoveryDistribution) (DiscoveryDistribution, error) {
	if value.ID == uuid.Nil || !value.Type.Valid() {
		return DiscoveryDistribution{}, apperror.Default(apperror.CodeInternal)
	}

	if value.Type == DistributionTypeDownloadableFile {
		if value.WebServiceType != "" || len(value.VisibleOn) != 0 || value.Provider != nil {
			return DiscoveryDistribution{}, apperror.Default(apperror.CodeInternal)
		}
		return value, nil
	}

	if !value.WebServiceType.Valid() {
		return DiscoveryDistribution{}, apperror.Default(apperror.CodeInternal)
	}

	visibleOn, err := normalizeWebServiceVisibleOn(value.VisibleOn)
	if err != nil {
		return DiscoveryDistribution{}, internalValidationError(err)
	}

	value.VisibleOn = visibleOn
	if value.Provider != nil {
		provider, err := NewDiscoveryReference(value.Provider.ID, value.Provider.Name)
		if err != nil {
			return DiscoveryDistribution{}, err
		}
		value.Provider = &provider
	}

	return value, nil
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
