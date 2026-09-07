package domain

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NewDatasetInput validates and normalizes caller-controlled dataset fields.
func NewDatasetInput(value DatasetInput) (DatasetInput, error) {
	value.Title = strings.TrimSpace(value.Title)
	if value.Title == "" {
		return DatasetInput{}, fmt.Errorf("dataset title is required: %w", ErrInvalidDataset)
	}
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return DatasetInput{}, fmt.Errorf("dataset keywords: %w: %w", err, ErrInvalidDataset)
	}
	value.Keywords = keywords
	if err := validateTemporal(value.TemporalCoverage); err != nil {
		return DatasetInput{}, fmt.Errorf("dataset temporal coverage: %w", err)
	}
	if err := validateUUIDList(value.ContactPointIDs, "contact point"); err != nil {
		return DatasetInput{}, fmt.Errorf("dataset contact points: %w: %w", err, ErrInvalidDataset)
	}
	if err := validateUUIDList(value.ProviderIDs, "provider"); err != nil {
		return DatasetInput{}, fmt.Errorf("dataset providers: %w: %w", err, ErrInvalidDataset)
	}
	if err := validateUUIDList(value.CategoryIDs, "category"); err != nil {
		return DatasetInput{}, fmt.Errorf("dataset categories: %w: %w", err, ErrInvalidDataset)
	}

	identifiers := make([]IdentifierInput, len(value.PersistentIdentifiers))
	seenIdentifiers := make(map[string]struct{}, len(value.PersistentIdentifiers))
	for i, identifier := range value.PersistentIdentifiers {
		identifier, err := NewIdentifierInput(identifier)
		if err != nil {
			return DatasetInput{}, fmt.Errorf("dataset identifier: %w: %w", err, ErrInvalidDataset)
		}
		key := string(identifier.Type) + "\x00" + identifier.Value
		if _, exists := seenIdentifiers[key]; exists {
			return DatasetInput{}, fmt.Errorf("duplicate dataset identifier %q: %w", key, ErrInvalidDataset)
		}
		seenIdentifiers[key] = struct{}{}
		identifiers[i] = identifier
	}
	value.PersistentIdentifiers = identifiers

	distributions := make([]DistributionInput, len(value.Distributions))
	for i, distribution := range value.Distributions {
		distribution, err := NewDistributionInput(distribution)
		if err != nil {
			return DatasetInput{}, fmt.Errorf("dataset distribution: %w", err)
		}
		distributions[i] = distribution
	}
	value.Distributions = distributions

	return value, nil
}

// NewDatasetDraftInput validates a dataset draft replacement command.
func NewDatasetDraftInput(value DatasetDraftInput) (DatasetDraftInput, error) {
	if value.Revision < 1 {
		return DatasetDraftInput{}, fmt.Errorf("revision must be positive: %w", ErrInvalidDataset)
	}
	metadata, err := NewDatasetInput(value.Metadata)
	if err != nil {
		return DatasetDraftInput{}, err
	}
	value.Metadata = metadata
	return value, nil
}

// NewDatasetRejectionInput validates and normalizes a rejection note.
func NewDatasetRejectionInput(value DatasetRejectionInput) (DatasetRejectionInput, error) {
	value.Note = strings.TrimSpace(value.Note)
	if value.Note == "" {
		return DatasetRejectionInput{}, fmt.Errorf("rejection note is required: %w", ErrInvalidDataset)
	}
	return value, nil
}

// NewDatasetMetadata validates persisted or imported dataset metadata.
func NewDatasetMetadata(value DatasetMetadata) (DatasetMetadata, error) {
	if value.ID == uuid.Nil {
		return DatasetMetadata{}, fmt.Errorf("dataset metadata ID is required: %w", ErrInvalidDataset)
	}
	value.Title = strings.TrimSpace(value.Title)
	if value.Title == "" {
		return DatasetMetadata{}, fmt.Errorf("dataset title is required: %w", ErrInvalidDataset)
	}
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return DatasetMetadata{}, fmt.Errorf("dataset keywords: %w: %w", err, ErrInvalidDataset)
	}
	value.Keywords = keywords
	if err := validateTemporal(value.TemporalCoverage); err != nil {
		return DatasetMetadata{}, fmt.Errorf("dataset temporal coverage: %w", err)
	}

	identifiers := make([]Identifier, len(value.PersistentIdentifiers))
	seenIdentifiers := make(map[uuid.UUID]struct{}, len(value.PersistentIdentifiers))
	for i, identifier := range value.PersistentIdentifiers {
		if err := validateIdentifier(identifier, true); err != nil {
			return DatasetMetadata{}, fmt.Errorf("dataset identifier: %w: %w", err, ErrInvalidDataset)
		}
		if _, exists := seenIdentifiers[identifier.ID]; exists {
			return DatasetMetadata{}, fmt.Errorf("duplicate dataset identifier %q: %w", identifier.ID, ErrInvalidDataset)
		}
		seenIdentifiers[identifier.ID] = struct{}{}
		identifiers[i] = identifier
	}
	value.PersistentIdentifiers = identifiers

	if err := validateContactPoints(value.ContactPoints, "dataset contact point"); err != nil {
		return DatasetMetadata{}, fmt.Errorf("dataset contact points: %w: %w", err, ErrInvalidDataset)
	}
	if err := validateOrganizations(value.Providers, "provider"); err != nil {
		return DatasetMetadata{}, fmt.Errorf("dataset providers: %w: %w", err, ErrInvalidDataset)
	}
	if err := validateCategories(value.Categories, "category"); err != nil {
		return DatasetMetadata{}, fmt.Errorf("dataset categories: %w: %w", err, ErrInvalidDataset)
	}

	distributions := make([]Distribution, len(value.Distributions))
	for i, distribution := range value.Distributions {
		if err := validateDistribution(distribution); err != nil {
			return DatasetMetadata{}, fmt.Errorf("dataset distribution: %w", err)
		}
		distributions[i] = distribution
	}
	value.Distributions = distributions

	return value, nil
}

// NewDatasetVersion validates a complete persisted dataset version.
//
//nolint:dupl // Resource version constructors intentionally share lifecycle validation.
func NewDatasetVersion(
	datasetID, versionID uuid.UUID,
	revision int32,
	status Status,
	createdBy Actor,
	group Group,
	createdAt, updatedAt time.Time,
	metadata DatasetMetadata,
) (DatasetVersion, error) {
	if datasetID == uuid.Nil || versionID == uuid.Nil {
		return DatasetVersion{}, fmt.Errorf("dataset and version IDs are required: %w", ErrInvalidDataset)
	}
	if revision < 1 {
		return DatasetVersion{}, fmt.Errorf("revision must be positive: %w", ErrInvalidDataset)
	}
	if !status.Valid() {
		return DatasetVersion{}, fmt.Errorf("dataset status %q is invalid: %w", status, ErrInvalidDataset)
	}
	actor, err := NewActor(string(createdBy))
	if err != nil {
		return DatasetVersion{}, fmt.Errorf("dataset creator: %w: %w", err, ErrInvalidDataset)
	}
	normalizedGroup, err := NewGroup(group.ID, GroupInput{
		Name:        group.Name,
		Description: group.Description,
		Label:       group.Label,
		Logo:        group.Logo,
		Page:        group.Page,
		Color:       group.Color,
		Position:    group.Position,
		Concept:     group.Concept,
	})
	if err != nil {
		return DatasetVersion{}, fmt.Errorf("dataset group: %w: %w", err, ErrInvalidDataset)
	}
	metadata, err = NewDatasetMetadata(metadata)
	if err != nil {
		return DatasetVersion{}, err
	}
	if createdAt.IsZero() || updatedAt.IsZero() {
		return DatasetVersion{}, fmt.Errorf("dataset timestamps are required: %w", ErrInvalidDataset)
	}
	return DatasetVersion{
		DatasetID: datasetID,
		VersionID: versionID,
		Revision:  revision,
		Status:    status,
		CreatedBy: actor,
		Group:     normalizedGroup,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Metadata:  metadata,
	}, nil
}

// Validate validates persisted or imported dataset metadata.
// TODO: delete this and make the callers instead just create their object using the correct parsing functions
func (value DatasetMetadata) Validate() error {
	_, err := NewDatasetMetadata(value)
	return err
}

// Input returns caller-shaped fields from persisted or imported metadata.
func (value DatasetMetadata) Input() DatasetInput {
	identifiers := make([]IdentifierInput, len(value.PersistentIdentifiers))
	for i, identifier := range value.PersistentIdentifiers {
		identifiers[i] = IdentifierInput{Type: identifier.Type, Value: identifier.Value}
	}
	providerIDs := make([]uuid.UUID, len(value.Providers))
	for i, provider := range value.Providers {
		providerIDs[i] = provider.ID
	}
	categoryIDs := make([]uuid.UUID, len(value.Categories))
	for i, category := range value.Categories {
		categoryIDs[i] = category.ID
	}
	contactPointIDs := make([]uuid.UUID, len(value.ContactPoints))
	for i, contact := range value.ContactPoints {
		contactPointIDs[i] = contact.ID
	}
	distributions := make([]DistributionInput, len(value.Distributions))
	for i, distribution := range value.Distributions {
		distributions[i] = distributionInput(distribution)
	}
	return DatasetInput{
		Title:                 value.Title,
		Description:           value.Description,
		Keywords:              value.Keywords,
		AccrualPeriodicity:    value.AccrualPeriodicity,
		QualityAnnotation:     value.QualityAnnotation,
		AccessRight:           value.AccessRight,
		Distributions:         distributions,
		SpatialCoverage:       value.SpatialCoverage,
		TemporalCoverage:      value.TemporalCoverage,
		PersistentIdentifiers: identifiers,
		ContactPointIDs:       contactPointIDs,
		ProviderIDs:           providerIDs,
		CategoryIDs:           categoryIDs,
		SourceCreated:         value.SourceCreated,
		SourceUpdated:         value.SourceUpdated,
	}
}

func NewDistributionInput(value DistributionInput) (DistributionInput, error) {
	value.Title = strings.TrimSpace(value.Title)
	if value.Title == "" {
		return DistributionInput{}, fmt.Errorf("distribution title is required: %w", ErrInvalidDataset)
	}
	if value.WebService != nil && value.DownloadableFile != nil {
		return DistributionInput{}, fmt.Errorf("distribution cannot contain both delivery methods: %w", ErrInvalidDataset)
	}
	var expected DistributionType
	switch {
	case value.WebService != nil:
		expected = DistributionTypeWebService
	case value.DownloadableFile != nil:
		expected = DistributionTypeDownloadableFile
	default:
		return DistributionInput{}, fmt.Errorf("distribution delivery method is required: %w", ErrInvalidDataset)
	}
	if value.Type != "" && value.Type != expected {
		return DistributionInput{}, fmt.Errorf("distribution type does not match delivery method: %w", ErrInvalidDataset)
	}
	value.Type = expected
	if value.DownloadableFile != nil {
		if strings.TrimSpace(value.DownloadableFile.URL) == "" || strings.TrimSpace(value.DownloadableFile.Format) == "" {
			return DistributionInput{}, fmt.Errorf("downloadable file URL and format are required: %w", ErrInvalidDataset)
		}
	}
	if value.WebService != nil {
		webService, err := NewWebServiceInput(*value.WebService)
		if err != nil {
			return DistributionInput{}, fmt.Errorf("web service: %w", err)
		}
		value.WebService = &webService
	}
	return value, nil
}

func NewWebServiceInput(value WebServiceInput) (WebServiceInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	value.TemplateURL = strings.TrimSpace(value.TemplateURL)
	value.Format = strings.TrimSpace(value.Format)
	if value.Name == "" || value.TemplateURL == "" || value.Format == "" {
		return WebServiceInput{}, fmt.Errorf("name, template URL, and format are required: %w", ErrInvalidDataset)
	}
	if err := validateTemporal(value.TemporalCoverage); err != nil {
		return WebServiceInput{}, fmt.Errorf("web service temporal coverage: %w", err)
	}
	if value.ProviderID != nil && *value.ProviderID == uuid.Nil {
		return WebServiceInput{}, fmt.Errorf("web service provider ID is required: %w", ErrInvalidDataset)
	}
	if err := validateUUIDList(value.ContactPointIDs, "web service contact point"); err != nil {
		return WebServiceInput{}, fmt.Errorf("web service contact points: %w: %w", err, ErrInvalidDataset)
	}
	if value.Documentation != nil && strings.TrimSpace(value.Documentation.URI) == "" {
		return WebServiceInput{}, fmt.Errorf("documentation URI is required: %w", ErrInvalidDataset)
	}
	seenVariables := make(map[string]struct{}, len(value.Parameters))
	for _, parameter := range value.Parameters {
		if err := validateParameter(parameter, false); err != nil {
			return WebServiceInput{}, fmt.Errorf("web service parameter: %w", err)
		}
		variable := parameter.GetMetadata().Variable
		if _, exists := seenVariables[variable]; exists {
			return WebServiceInput{}, fmt.Errorf("duplicate parameter variable %q: %w", variable, ErrInvalidDataset)
		}
		seenVariables[variable] = struct{}{}
	}
	return value, nil
}

func validateDistribution(value Distribution) error {
	if value.ID == uuid.Nil || strings.TrimSpace(value.Title) == "" {
		return fmt.Errorf("ID and title are required: %w", ErrInvalidDataset)
	}
	if !value.Type.Valid() {
		return fmt.Errorf("distribution type is invalid: %w", ErrInvalidDataset)
	}
	expected, err := value.DeliveryType()
	if err != nil {
		return fmt.Errorf("%w: %w", err, ErrInvalidDataset)
	}
	if value.Type != expected {
		return fmt.Errorf("distribution type does not match delivery method: %w", ErrInvalidDataset)
	}
	if value.DownloadableFile != nil && (strings.TrimSpace(value.DownloadableFile.URL) == "" || strings.TrimSpace(value.DownloadableFile.Format) == "") {
		return fmt.Errorf("downloadable file URL and format are required: %w", ErrInvalidDataset)
	}
	if value.WebService != nil {
		if err := validateWebService(*value.WebService); err != nil {
			return fmt.Errorf("web service: %w", err)
		}
	}
	return nil
}

func (d Distribution) DeliveryType() (DistributionType, error) {
	switch {
	case d.WebService != nil && d.DownloadableFile == nil:
		return DistributionTypeWebService, nil
	case d.WebService == nil && d.DownloadableFile != nil:
		return DistributionTypeDownloadableFile, nil
	default:
		return "", fmt.Errorf("exactly one delivery method is required")
	}
}

func validateWebService(value WebService) error {
	if value.ID == uuid.Nil || strings.TrimSpace(value.Name) == "" || strings.TrimSpace(value.TemplateURL) == "" || strings.TrimSpace(value.Format) == "" {
		return fmt.Errorf("ID, name, template URL, and format are required: %w", ErrInvalidDataset)
	}
	if err := validateTemporal(value.TemporalCoverage); err != nil {
		return fmt.Errorf("temporal coverage: %w", err)
	}
	if value.Documentation != nil && strings.TrimSpace(value.Documentation.URI) == "" {
		return fmt.Errorf("documentation URI is required: %w", ErrInvalidDataset)
	}
	if value.Provider != nil {
		if err := validateOrganization(*value.Provider); err != nil {
			return fmt.Errorf("provider: %w: %w", err, ErrInvalidDataset)
		}
	}
	if err := validateContactPoints(value.ContactPoints, "web service contact point"); err != nil {
		return fmt.Errorf("contact points: %w: %w", err, ErrInvalidDataset)
	}
	seenVariables := make(map[string]struct{}, len(value.Parameters))
	for _, parameter := range value.Parameters {
		if err := validateParameter(parameter, true); err != nil {
			return fmt.Errorf("parameter: %w", err)
		}
		variable := parameter.GetMetadata().Variable
		if _, exists := seenVariables[variable]; exists {
			return fmt.Errorf("duplicate parameter variable %q: %w", variable, ErrInvalidDataset)
		}
		seenVariables[variable] = struct{}{}
	}
	return nil
}

func validateTemporal(value *Temporal) error {
	if value != nil && value.Start != nil && value.End != nil && value.Start.After(*value.End) {
		return fmt.Errorf("start must not be after end: %w", ErrInvalidDataset)
	}
	return nil
}

func validateParameter(value Parameter, requireID bool) error {
	if value == nil {
		return fmt.Errorf("parameter is required: %w", ErrInvalidDataset)
	}
	metadata := value.GetMetadata()
	if requireID && metadata.ID == uuid.Nil {
		return fmt.Errorf("parameter ID is required: %w", ErrInvalidDataset)
	}
	if !value.Kind().Valid() || (metadata.Kind != "" && metadata.Kind != value.Kind()) {
		return fmt.Errorf("parameter kind is invalid: %w", ErrInvalidDataset)
	}
	if strings.TrimSpace(metadata.Variable) == "" || strings.TrimSpace(metadata.Property) == "" || strings.TrimSpace(metadata.Label) == "" {
		return fmt.Errorf("parameter variable, property, and label are required: %w", ErrInvalidDataset)
	}
	switch parameter := value.(type) {
	case StringParameter:
		if err := validatePattern(parameter.Pattern); err != nil {
			return err
		}
		if parameter.Default != nil && len(parameter.AllowedValues) > 0 && !slices.Contains(parameter.AllowedValues, *parameter.Default) {
			return fmt.Errorf("parameter default is not an allowed value: %w", ErrInvalidDataset)
		}
	case BooleanParameter:
	case IntegerParameter:
		if parameter.Minimum != nil && parameter.Maximum != nil && *parameter.Minimum > *parameter.Maximum {
			return fmt.Errorf("parameter minimum must not exceed maximum: %w", ErrInvalidDataset)
		}
		if parameter.Default != nil && ((parameter.Minimum != nil && *parameter.Default < *parameter.Minimum) || (parameter.Maximum != nil && *parameter.Default > *parameter.Maximum)) {
			return fmt.Errorf("parameter default is outside allowed range: %w", ErrInvalidDataset)
		}
		if parameter.Default != nil && len(parameter.AllowedValues) > 0 && !slices.Contains(parameter.AllowedValues, *parameter.Default) {
			return fmt.Errorf("parameter default is not an allowed value: %w", ErrInvalidDataset)
		}
	case FloatParameter:
		if (parameter.Minimum != nil && !isFinite(parameter.Minimum)) || (parameter.Maximum != nil && !isFinite(parameter.Maximum)) || (parameter.Default != nil && !isFinite(parameter.Default)) {
			return fmt.Errorf("parameter values must be finite: %w", ErrInvalidDataset)
		}
		if parameter.Minimum != nil && parameter.Maximum != nil && *parameter.Minimum > *parameter.Maximum {
			return fmt.Errorf("parameter minimum must not exceed maximum: %w", ErrInvalidDataset)
		}
		if parameter.Default != nil && ((parameter.Minimum != nil && *parameter.Default < *parameter.Minimum) || (parameter.Maximum != nil && *parameter.Default > *parameter.Maximum)) {
			return fmt.Errorf("parameter default is outside allowed range: %w", ErrInvalidDataset)
		}
		if parameter.Default != nil && len(parameter.AllowedValues) > 0 && !slices.Contains(parameter.AllowedValues, *parameter.Default) {
			return fmt.Errorf("parameter default is not an allowed value: %w", ErrInvalidDataset)
		}
	case DateTimeParameter:
		if err := validatePattern(parameter.Pattern); err != nil {
			return err
		}
		if parameter.Minimum != nil && parameter.Maximum != nil && parameter.Minimum.After(*parameter.Maximum) {
			return fmt.Errorf("parameter minimum must not be after maximum: %w", ErrInvalidDataset)
		}
		if parameter.Default != nil && ((parameter.Minimum != nil && parameter.Default.Before(*parameter.Minimum)) || (parameter.Maximum != nil && parameter.Default.After(*parameter.Maximum))) {
			return fmt.Errorf("parameter default is outside allowed range: %w", ErrInvalidDataset)
		}
	default:
		return fmt.Errorf("unsupported parameter type %T: %w", value, ErrInvalidDataset)
	}
	return nil
}

func validatePattern(pattern string) error {
	if pattern == "" {
		return nil
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("invalid parameter pattern: %w: %w", err, ErrInvalidDataset)
	}
	return nil
}

func isFinite(value *float64) bool {
	return value != nil && !math.IsNaN(*value) && !math.IsInf(*value, 0)
}

func distributionInput(value Distribution) DistributionInput {
	result := DistributionInput{
		Title:            value.Title,
		Description:      value.Description,
		License:          value.License,
		Type:             value.Type,
		Issued:           value.Issued,
		Modified:         value.Modified,
		ByteSize:         value.ByteSize,
		WebService:       nil,
		DownloadableFile: value.DownloadableFile,
	}
	if value.WebService != nil {
		providerID := uuid.Nil
		if value.WebService.Provider != nil {
			providerID = value.WebService.Provider.ID
		}
		var providerIDPointer *uuid.UUID
		if value.WebService.Provider != nil {
			providerIDPointer = &providerID
		}
		contactPointIDs := make([]uuid.UUID, len(value.WebService.ContactPoints))
		for i, contact := range value.WebService.ContactPoints {
			contactPointIDs[i] = contact.ID
		}
		result.WebService = &WebServiceInput{
			Name:             value.WebService.Name,
			Description:      value.WebService.Description,
			TemplateURL:      value.WebService.TemplateURL,
			Parameters:       value.WebService.Parameters,
			Format:           value.WebService.Format,
			ProviderID:       providerIDPointer,
			Documentation:    value.WebService.Documentation,
			SpatialCoverage:  value.WebService.SpatialCoverage,
			TemporalCoverage: value.WebService.TemporalCoverage,
			ContactPointIDs:  contactPointIDs,
		}
	}
	return result
}
