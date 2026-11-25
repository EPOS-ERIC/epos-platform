package domain

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
	"github.com/twpayne/go-geom"
)

type AccessRight string

const (
	AccessRightNonPublic    AccessRight = "NON_PUBLIC"
	AccessRightNormal       AccessRight = "NORMAL"
	AccessRightOpDatpro     AccessRight = "OP_DATPRO"
	AccessRightPublic       AccessRight = "PUBLIC"
	AccessRightRestricted   AccessRight = "RESTRICTED"
	AccessRightConfidential AccessRight = "CONFIDENTIAL"
	AccessRightSensitive    AccessRight = "SENSITIVE"
)

func (value AccessRight) Valid() bool {
	switch value {
	case AccessRightNonPublic,
		AccessRightNormal,
		AccessRightOpDatpro,
		AccessRightPublic,
		AccessRightRestricted,
		AccessRightConfidential,
		AccessRightSensitive:
		return true
	default:
		return false
	}
}

type AccrualPeriodicity string

const (
	AccrualPeriodicityTriennial        AccrualPeriodicity = "TRIENNIAL"
	AccrualPeriodicityBiennial         AccrualPeriodicity = "BIENNIAL"
	AccrualPeriodicityAnnual           AccrualPeriodicity = "ANNUAL"
	AccrualPeriodicitySemiannual       AccrualPeriodicity = "SEMIANNUAL"
	AccrualPeriodicityThreeTimesAYear  AccrualPeriodicity = "THREE_TIMES_A_YEAR"
	AccrualPeriodicityQuarterly        AccrualPeriodicity = "QUARTERLY"
	AccrualPeriodicityBimonthly        AccrualPeriodicity = "BIMONTHLY"
	AccrualPeriodicityMonthly          AccrualPeriodicity = "MONTHLY"
	AccrualPeriodicitySemimonthly      AccrualPeriodicity = "SEMIMONTHLY"
	AccrualPeriodicityBiweekly         AccrualPeriodicity = "BIWEEKLY"
	AccrualPeriodicityThreeTimesAMonth AccrualPeriodicity = "THREE_TIMES_A_MONTH"
	AccrualPeriodicityWeekly           AccrualPeriodicity = "WEEKLY"
	AccrualPeriodicitySemiweekly       AccrualPeriodicity = "SEMIWEEKLY"
	AccrualPeriodicityThreeTimesAWeek  AccrualPeriodicity = "THREE_TIMES_A_WEEK"
	AccrualPeriodicityDaily            AccrualPeriodicity = "DAILY"
	AccrualPeriodicityContinuous       AccrualPeriodicity = "CONTINUOUS"
	AccrualPeriodicityIrregular        AccrualPeriodicity = "IRREGULAR"
)

func (value AccrualPeriodicity) Valid() bool {
	switch value {
	case AccrualPeriodicityTriennial,
		AccrualPeriodicityBiennial,
		AccrualPeriodicityAnnual,
		AccrualPeriodicitySemiannual,
		AccrualPeriodicityThreeTimesAYear,
		AccrualPeriodicityQuarterly,
		AccrualPeriodicityBimonthly,
		AccrualPeriodicityMonthly,
		AccrualPeriodicitySemimonthly,
		AccrualPeriodicityBiweekly,
		AccrualPeriodicityThreeTimesAMonth,
		AccrualPeriodicityWeekly,
		AccrualPeriodicitySemiweekly,
		AccrualPeriodicityThreeTimesAWeek,
		AccrualPeriodicityDaily,
		AccrualPeriodicityContinuous,
		AccrualPeriodicityIrregular:
		return true
	default:
		return false
	}
}

// DatasetInput contains caller-controlled dataset version fields.
type DatasetInput struct {
	Title                      string
	Description                string
	Keywords                   []string
	AccrualPeriodicity         *AccrualPeriodicity
	QualityAnnotation          *string
	ConformsTo                 *string
	Provenance                 *string
	Type                       *string
	Version                    *string
	SpatialResolutionInMeters  *float64
	TemporalResolution         *float64
	AccessRight                *AccessRight
	Distributions              []DistributionInput
	SpatialCoverage            geom.T
	TemporalCoverage           *Temporal
	GeologicalTemporalCoverage *GeologicalTemporal
	PersistentIdentifiers      []IdentifierInput
	ContactPointIDs            []uuid.UUID
	ProviderIDs                []uuid.UUID
	CategoryIDs                []uuid.UUID
	SourceCreated              *time.Time
	SourceUpdated              *time.Time
}

// DatasetDraftInput contains a validated draft replacement command.
type DatasetDraftInput struct {
	Revision int32
	Metadata DatasetInput
}

// DatasetRejectionInput contains a validated rejection command.
type DatasetRejectionInput struct {
	Note string
}

// DatasetMetadata is persisted dataset version metadata.
type DatasetMetadata struct {
	ID                         uuid.UUID
	Title                      string
	Description                string
	Keywords                   []string
	AccrualPeriodicity         *AccrualPeriodicity
	QualityAnnotation          *string
	ConformsTo                 *string
	Provenance                 *string
	Type                       *string
	Version                    *string
	SpatialResolutionInMeters  *float64
	TemporalResolution         *float64
	AccessRight                *AccessRight
	Distributions              []Distribution
	SpatialCoverage            geom.T
	TemporalCoverage           *Temporal
	GeologicalTemporalCoverage *GeologicalTemporal
	PersistentIdentifiers      []Identifier
	ContactPoints              []ContactPoint
	Providers                  []OrganizationMetadata
	Categories                 []CategoryMetadata
	SourceCreated              *time.Time
	SourceUpdated              *time.Time
}

type Temporal struct {
	Start *time.Time
	End   *time.Time
}

type GeologicalTemporal struct {
	StartMYA *float64
	EndMYA   *float64
}

// DatasetVersion is dataset metadata with lifecycle identity.
type DatasetVersion struct {
	DatasetID     uuid.UUID
	VersionID     uuid.UUID
	Revision      int32
	Status        Status
	CreatedBy     Actor
	SubmittedBy   *Actor
	Group         Group
	CreatedAt     time.Time
	UpdatedAt     time.Time
	RejectedBy    *string
	RejectedAt    *time.Time
	RejectionNote *string
	Metadata      DatasetMetadata
}

// Normalize validates and normalizes caller-controlled dataset fields.
func (value *DatasetInput) Normalize() error {
	value.Title = strings.TrimSpace(value.Title)
	if value.Title == "" {
		return apperror.Field("title", "Title must not be blank.", nil)
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return apperror.Field("description", "Description must not be blank.", nil)
	}
	accrualPeriodicity, err := optionalAccrualPeriodicity(value.AccrualPeriodicity)
	if err != nil {
		return err
	}
	value.AccrualPeriodicity = accrualPeriodicity
	if err := normalizeOptionalHTTPURL(value.QualityAnnotation, "quality_annotation"); err != nil {
		return err
	}
	value.ConformsTo = optionalTrimmedString(value.ConformsTo)
	value.Provenance = optionalTrimmedString(value.Provenance)
	value.Type = optionalTrimmedString(value.Type)
	value.Version = optionalTrimmedString(value.Version)
	if err := validateResolutions(value.SpatialResolutionInMeters, value.TemporalResolution); err != nil {
		return err
	}
	accessRight, err := optionalAccessRight(value.AccessRight)
	if err != nil {
		return err
	}
	value.AccessRight = accessRight
	keywords, err := normalizeKeywordInput(value.Keywords)
	if err != nil {
		return err
	}
	value.Keywords = keywords
	if err := validateDatasetTemporal(value.TemporalCoverage, value.GeologicalTemporalCoverage); err != nil {
		target, detail := "temporal_coverage", "Temporal coverage is invalid."
		if value.GeologicalTemporalCoverage != nil {
			target, detail = "geological_temporal_coverage", "Temporal and geological coverage cannot both be set."
		}
		return apperror.Field(target, detail, err)
	}
	if err := validateUUIDListField(value.ContactPointIDs, "contact_points", "Contact point"); err != nil {
		return err
	}
	if err := validateUUIDListField(value.ProviderIDs, "providers", "Provider"); err != nil {
		return err
	}
	if err := validateUUIDListField(value.CategoryIDs, "categories", "Category"); err != nil {
		return err
	}

	identifiers, err := normalizeIdentifierInputs(value.PersistentIdentifiers, "persistent_identifiers", "dataset")
	if err != nil {
		return err
	}
	value.PersistentIdentifiers = identifiers

	distributions := make([]DistributionInput, len(value.Distributions))
	for i, distribution := range value.Distributions {
		if err := distribution.Normalize(); err != nil {
			return fmt.Errorf("validate dataset distribution: %w", apperror.PrefixTarget(err, fmt.Sprintf("distributions[%d]", i)))
		}
		distributions[i] = distribution
	}
	value.Distributions = distributions

	return nil
}

// Normalize validates a dataset draft replacement command.
func (value *DatasetDraftInput) Normalize() error {
	if value.Revision < 1 {
		return apperror.Field("revision", "Revision must be positive.", nil)
	}
	if err := value.Metadata.Normalize(); err != nil {
		return fmt.Errorf("validate dataset draft metadata: %w", apperror.PrefixTarget(err, "metadata"))
	}
	return nil
}

// Normalize validates and normalizes a rejection note.
func (value *DatasetRejectionInput) Normalize() error {
	value.Note = strings.TrimSpace(value.Note)
	if value.Note == "" {
		return apperror.Field("note", "Rejection note must not be blank.", nil)
	}
	return nil
}

// Normalize validates persisted or imported dataset metadata.
func (value *DatasetMetadata) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.ID == uuid.Nil {
		return fmt.Errorf("dataset metadata ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Title = strings.TrimSpace(value.Title)
	if value.Title == "" {
		return fmt.Errorf("dataset title is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return fmt.Errorf("dataset description is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.AccrualPeriodicity, err = optionalAccrualPeriodicity(value.AccrualPeriodicity)
	if err != nil {
		return fmt.Errorf("dataset accrual periodicity: %w", err)
	}
	if err := normalizeOptionalHTTPURL(value.QualityAnnotation, "quality_annotation"); err != nil {
		return err
	}
	value.ConformsTo = optionalTrimmedString(value.ConformsTo)
	value.Provenance = optionalTrimmedString(value.Provenance)
	value.Type = optionalTrimmedString(value.Type)
	value.Version = optionalTrimmedString(value.Version)
	if err := validateResolutions(value.SpatialResolutionInMeters, value.TemporalResolution); err != nil {
		return err
	}
	value.AccessRight, err = optionalAccessRight(value.AccessRight)
	if err != nil {
		return fmt.Errorf("dataset access right: %w", err)
	}
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return fmt.Errorf("dataset keywords: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	value.Keywords = keywords
	if err := validateDatasetTemporal(value.TemporalCoverage, value.GeologicalTemporalCoverage); err != nil {
		return fmt.Errorf("dataset temporal coverage: %w", err)
	}

	identifiers := make([]Identifier, len(value.PersistentIdentifiers))
	seenIdentifiers := make(map[uuid.UUID]struct{}, len(value.PersistentIdentifiers))
	for i, identifier := range value.PersistentIdentifiers {
		if err := validateIdentifier(identifier); err != nil {
			return fmt.Errorf("dataset identifier: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		if _, exists := seenIdentifiers[identifier.ID]; exists {
			return fmt.Errorf("duplicate dataset identifier %q: %w", identifier.ID, apperror.Default(apperror.CodeValidationFailed))
		}
		seenIdentifiers[identifier.ID] = struct{}{}
		identifiers[i] = identifier
	}
	value.PersistentIdentifiers = identifiers

	if err := validateContactPoints(value.ContactPoints, "dataset contact point"); err != nil {
		return fmt.Errorf("dataset contact points: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateOrganizations(value.Providers, "provider"); err != nil {
		return fmt.Errorf("dataset providers: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateCategories(value.Categories, "category"); err != nil {
		return fmt.Errorf("dataset categories: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}

	distributions := make([]Distribution, len(value.Distributions))
	for i, distribution := range value.Distributions {
		if err := distribution.Normalize(); err != nil {
			return fmt.Errorf("dataset distribution: %w", err)
		}
		distributions[i] = distribution
	}
	value.Distributions = distributions

	return nil
}

func validateResolutions(spatial, temporal *float64) error {
	if spatial != nil && *spatial < 0 {
		return apperror.Field("spatial_resolution_in_meters", "Resolution must be a non-negative number.", nil)
	}
	if temporal != nil && *temporal < 0 {
		return apperror.Field("temporal_resolution", "Resolution must be a non-negative number.", nil)
	}
	return nil
}

// Normalize validates a complete persisted dataset version.
//
//nolint:dupl // Resource version constructors intentionally share lifecycle validation.
func (value *DatasetVersion) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.DatasetID == uuid.Nil || value.VersionID == uuid.Nil {
		return fmt.Errorf("dataset and version IDs are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value.Revision < 1 {
		return fmt.Errorf("revision must be positive: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if !value.Status.Valid() {
		return fmt.Errorf("dataset status %q is invalid: %w", value.Status, apperror.Default(apperror.CodeValidationFailed))
	}
	actor, err := NewActor(string(value.CreatedBy))
	if err != nil {
		return fmt.Errorf("dataset creator: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	value.CreatedBy = actor
	if value.SubmittedBy != nil {
		submittedBy, err := NewActor(string(*value.SubmittedBy))
		if err != nil {
			return fmt.Errorf("dataset submitter: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		value.SubmittedBy = &submittedBy
	}
	if err := value.Group.Normalize(); err != nil {
		return fmt.Errorf("dataset group: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := value.Metadata.Normalize(); err != nil {
		return err
	}
	if value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() {
		return fmt.Errorf("dataset timestamps are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.RejectionNote, err = normalizeRejectionNote(value.Status, value.RejectionNote)
	if err != nil {
		return fmt.Errorf("dataset rejection note: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

func validateTemporal(value *Temporal) error {
	if value != nil && value.Start != nil && value.End != nil && value.Start.After(*value.End) {
		return fmt.Errorf("start must not be after end: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

func validateDatasetTemporal(temporal *Temporal, geological *GeologicalTemporal) error {
	if temporal != nil && geological != nil {
		return fmt.Errorf("calendar and geological temporal coverage are mutually exclusive: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateTemporal(temporal); err != nil {
		return err
	}
	if geological == nil {
		return nil
	}
	if value := geological.StartMYA; value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0) {
		return fmt.Errorf("start_mya must be finite and non-negative: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value := geological.EndMYA; value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0) {
		return fmt.Errorf("end_mya must be finite and non-negative: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if geological.StartMYA != nil && geological.EndMYA != nil && *geological.StartMYA < *geological.EndMYA {
		return fmt.Errorf("start_mya must not be less than end_mya: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

func optionalAccessRight(value *AccessRight) (*AccessRight, error) {
	if value == nil {
		return nil, nil
	}
	normalized := AccessRight(strings.ToUpper(strings.TrimSpace(string(*value))))
	if normalized == "" {
		return nil, nil
	}
	if !normalized.Valid() {
		return nil, apperror.Field("access_right", "Access right is not supported.", nil)
	}
	return &normalized, nil
}

func optionalAccrualPeriodicity(value *AccrualPeriodicity) (*AccrualPeriodicity, error) {
	if value == nil {
		return nil, nil
	}
	normalized := AccrualPeriodicity(strings.ToUpper(strings.TrimSpace(string(*value))))
	if normalized == "" {
		return nil, nil
	}
	if !normalized.Valid() {
		return nil, apperror.Field("accrual_periodicity", "Accrual periodicity is not supported.", nil)
	}
	return &normalized, nil
}
