package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkt"
)

// DiscoveryFilter contains validated search criteria for resource discovery.
type DiscoveryFilter struct {
	Query              *string
	Types              []ResourceType
	Concepts           []uuid.UUID
	GroupID            *uuid.UUID
	SpatialFilter      geom.T
	DateFrom           *time.Time
	DateTo             *time.Time
	Providers          []uuid.UUID
	Keywords           []string
	EditorID           *string
	VersioningStatuses []Status
}

// DiscoveryCandidate is the resource summary returned by persistence.
type DiscoveryCandidate struct {
	Type       ResourceType
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	GroupID    uuid.UUID
	GroupName  string
	Status     Status
	Name       string
	UpdatedAt  time.Time
	Concept    DiscoveryReference
	CreatedBy  *DiscoveryCreatedBy
}

// DiscoveryRelations contains only relationships exposed by discovery.
type DiscoveryRelations struct {
	Providers     []DiscoveryReference
	Creators      []DiscoveryReference
	Categories    []DiscoveryReference
	Distributions []DiscoveryDistribution
}

// DiscoveryItem is a resource summary returned by discovery.
type DiscoveryItem struct {
	Type          ResourceType
	ResourceID    uuid.UUID
	VersionID     uuid.UUID
	Name          string
	Status        Status
	UpdatedAt     time.Time
	Concept       DiscoveryReference
	CreatedBy     *DiscoveryCreatedBy
	Group         DiscoveryReference
	Providers     []DiscoveryReference
	Creators      []DiscoveryReference
	Categories    []DiscoveryReference
	Distributions []DiscoveryDistribution
}

// DiscoveryDistribution is a typed distribution summary for discovery.
type DiscoveryDistribution struct {
	ID             uuid.UUID
	Name           string
	Type           DistributionType
	WebServiceType WebServiceType
	VisibleOn      []WebServiceVisibleOn
	Provider       *DiscoveryReference
}

// DiscoveryReference identifies related metadata attached to a resource.
type DiscoveryReference struct {
	ID   uuid.UUID
	Name string
}

// DiscoveryCreatedBy identifies the user who created a resource version.
type DiscoveryCreatedBy struct {
	ID   string
	Name string
}

// Normalize validates and normalizes discovery search criteria.
func (value *DiscoveryFilter) Normalize() error {
	value.Query = optionalTrimmedString(value.Query)
	value.EditorID = optionalTrimmedString(value.EditorID)
	concepts := make([]uuid.UUID, 0, len(value.Concepts))
	for _, concept := range value.Concepts {
		if concept == uuid.Nil {
			return apperror.Field("concepts", "Concept IDs must be valid UUIDs.", nil)
		}
		if !slices.Contains(concepts, concept) {
			concepts = append(concepts, concept)
		}
	}
	value.Concepts = concepts
	if value.GroupID != nil && *value.GroupID == uuid.Nil {
		return apperror.Field("group_id", "Group ID must be a valid UUID.", nil)
	}

	types := make([]ResourceType, 0, len(value.Types))
	for _, resourceType := range value.Types {
		resourceType = ResourceType(strings.TrimSpace(string(resourceType)))
		if resourceType == "" {
			continue
		}
		if !validResourceType(resourceType) {
			return apperror.Field("types", "Resource type is not supported.", nil)
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
		if !status.Valid() {
			return apperror.Field("versioning_statuses", "Status is not supported.", nil)
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
		return apperror.Field("providers", "Provider IDs must be valid UUIDs.", nil)
	}
	providers := make([]uuid.UUID, len(value.Providers))
	copy(providers, value.Providers)
	value.Providers = providers
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return apperror.Field("keywords", "Keywords are invalid.", err)
	}
	value.Keywords = keywords

	if value.DateFrom != nil && value.DateTo != nil && value.DateFrom.After(*value.DateTo) {
		return apperror.Field("date_from", "Start date must not be after end date.", nil)
	}

	return nil
}

// ParseSpatialFilter parses a WKT spatial filter.
func ParseSpatialFilter(value string) (geom.T, error) {
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

// Normalize validates a persisted discovery result.
func (value *DiscoveryCandidate) Normalize() error {
	value.Name = strings.TrimSpace(value.Name)
	if value.ResourceID == uuid.Nil || value.VersionID == uuid.Nil || value.GroupID == uuid.Nil {
		return apperror.Default(apperror.CodeInternal)
	}
	if !validResourceType(value.Type) || !value.Status.Valid() || value.Name == "" {
		return apperror.Default(apperror.CodeInternal)
	}
	if err := value.Concept.Normalize(); err != nil {
		return err
	}
	if value.CreatedBy != nil {
		if err := value.CreatedBy.Normalize(); err != nil {
			return err
		}
	}
	return nil
}

// Normalize validates a relationship projection.
func (value *DiscoveryReference) Normalize() error {
	if value.ID == uuid.Nil {
		return apperror.Default(apperror.CodeInternal)
	}
	return nil
}

// Normalize validates a creator projection.
func (value *DiscoveryCreatedBy) Normalize() error {
	value.ID = strings.TrimSpace(value.ID)
	value.Name = strings.TrimSpace(value.Name)
	if value.ID == "" || value.Name == "" {
		return apperror.Default(apperror.CodeInternal)
	}
	return nil
}

// Normalize validates a discovery item and its relationships.
func (value *DiscoveryItem) Normalize() error {
	value.Name = strings.TrimSpace(value.Name)
	if value.ResourceID == uuid.Nil || value.VersionID == uuid.Nil || value.Name == "" || !validResourceType(value.Type) || !value.Status.Valid() {
		return apperror.Default(apperror.CodeInternal)
	}
	if err := value.Concept.Normalize(); err != nil {
		return err
	}
	if err := value.Group.Normalize(); err != nil {
		return err
	}
	validatedProviders, err := discoveryReferences(value.Providers)
	if err != nil {
		return err
	}
	validatedCreators, err := discoveryReferences(value.Creators)
	if err != nil {
		return err
	}
	validatedCategories, err := discoveryReferences(value.Categories)
	if err != nil {
		return err
	}
	validatedDistributions := make([]DiscoveryDistribution, 0, len(value.Distributions))
	for _, distribution := range value.Distributions {
		if err := distribution.Normalize(); err != nil {
			return err
		}
		validatedDistributions = append(validatedDistributions, distribution)
	}
	if value.CreatedBy != nil {
		if err := value.CreatedBy.Normalize(); err != nil {
			return err
		}
	}
	value.Providers = validatedProviders
	value.Creators = validatedCreators
	value.Categories = validatedCategories
	value.Distributions = validatedDistributions
	return nil
}

// Normalize validates a persisted distribution summary.
func (value *DiscoveryDistribution) Normalize() error {
	if value.ID == uuid.Nil || !value.Type.Valid() {
		return apperror.Default(apperror.CodeInternal)
	}

	if value.Type == DistributionTypeDownloadableFile {
		if value.WebServiceType != "" || len(value.VisibleOn) != 0 || value.Provider != nil {
			return apperror.Default(apperror.CodeInternal)
		}
		return nil
	}

	if !value.WebServiceType.Valid() {
		return apperror.Default(apperror.CodeInternal)
	}

	visibleOn, err := normalizeWebServiceVisibleOn(value.VisibleOn)
	if err != nil {
		return internalValidationError(err)
	}

	value.VisibleOn = visibleOn
	if value.Provider != nil {
		if err := value.Provider.Normalize(); err != nil {
			return err
		}
	}

	return nil
}

func discoveryReferences(values []DiscoveryReference) ([]DiscoveryReference, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]DiscoveryReference, 0, len(values))
	for _, value := range values {
		if err := value.Normalize(); err != nil {
			return nil, err
		}
		result = append(result, value)
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
