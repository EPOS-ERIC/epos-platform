package domain

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// NewDatasetInput validates and normalizes caller-controlled dataset fields.
func NewDatasetInput(value DatasetInput) (DatasetInput, error) {
	value.Title = strings.TrimSpace(value.Title)
	if value.Title == "" {
		return DatasetInput{}, invalidField("title", "Title must not be blank.", nil)
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return DatasetInput{}, invalidField("description", "Description must not be blank.", nil)
	}
	value.AccrualPeriodicity = optionalTrimmedString(value.AccrualPeriodicity)
	value.QualityAnnotation = optionalTrimmedString(value.QualityAnnotation)
	value.AccessRight = optionalTrimmedString(value.AccessRight)
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return DatasetInput{}, invalidField("keywords", "Keywords must not be blank.", err)
	}
	value.Keywords = keywords
	if err := validateDatasetTemporal(value.TemporalCoverage, value.GeologicalTemporalCoverage); err != nil {
		target, detail := "temporal_coverage", "Temporal coverage is invalid."
		if value.GeologicalTemporalCoverage != nil {
			target, detail = "geological_temporal_coverage", "Temporal and geological coverage cannot both be set."
		}
		return DatasetInput{}, invalidField(target, detail, err)
	}
	if err := validateUUIDListField(value.ContactPointIDs, "contact_points", "Contact point"); err != nil {
		return DatasetInput{}, err
	}
	if err := validateUUIDListField(value.ProviderIDs, "providers", "Provider"); err != nil {
		return DatasetInput{}, err
	}
	if err := validateUUIDListField(value.CategoryIDs, "categories", "Category"); err != nil {
		return DatasetInput{}, err
	}

	identifiers := make([]IdentifierInput, len(value.PersistentIdentifiers))
	seenIdentifiers := make(map[string]struct{}, len(value.PersistentIdentifiers))
	for i, identifier := range value.PersistentIdentifiers {
		identifier, err := NewIdentifierInput(identifier)
		if err != nil {
			return DatasetInput{}, fmt.Errorf("validate dataset identifier: %w", apperror.PrefixTarget(err, fmt.Sprintf("persistent_identifiers[%d]", i)))
		}
		key := string(identifier.Type) + "\x00" + identifier.Value
		if _, exists := seenIdentifiers[key]; exists {
			return DatasetInput{}, invalidField(fmt.Sprintf("persistent_identifiers[%d]", i), "Identifier must be unique.", nil)
		}
		seenIdentifiers[key] = struct{}{}
		identifiers[i] = identifier
	}
	value.PersistentIdentifiers = identifiers

	distributions := make([]DistributionInput, len(value.Distributions))
	for i, distribution := range value.Distributions {
		distribution, err := NewDistributionInput(distribution)
		if err != nil {
			return DatasetInput{}, fmt.Errorf("validate dataset distribution: %w", apperror.PrefixTarget(err, fmt.Sprintf("distributions[%d]", i)))
		}
		distributions[i] = distribution
	}
	value.Distributions = distributions

	return value, nil
}

// NewDatasetDraftInput validates a dataset draft replacement command.
func NewDatasetDraftInput(value DatasetDraftInput) (DatasetDraftInput, error) {
	if value.Revision < 1 {
		return DatasetDraftInput{}, invalidField("revision", "Revision must be positive.", nil)
	}
	metadata, err := NewDatasetInput(value.Metadata)
	if err != nil {
		return DatasetDraftInput{}, fmt.Errorf("validate dataset draft metadata: %w", apperror.PrefixTarget(err, "metadata"))
	}
	value.Metadata = metadata
	return value, nil
}

// NewDatasetRejectionInput validates and normalizes a rejection note.
func NewDatasetRejectionInput(value DatasetRejectionInput) (DatasetRejectionInput, error) {
	value.Note = strings.TrimSpace(value.Note)
	if value.Note == "" {
		return DatasetRejectionInput{}, invalidField("note", "Rejection note must not be blank.", nil)
	}
	return value, nil
}

// NewDatasetMetadata validates persisted or imported dataset metadata.
func NewDatasetMetadata(value DatasetMetadata) (result DatasetMetadata, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.ID == uuid.Nil {
		return DatasetMetadata{}, fmt.Errorf("dataset metadata ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Title = strings.TrimSpace(value.Title)
	if value.Title == "" {
		return DatasetMetadata{}, fmt.Errorf("dataset title is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return DatasetMetadata{}, fmt.Errorf("dataset description is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	value.AccrualPeriodicity = optionalTrimmedString(value.AccrualPeriodicity)
	value.QualityAnnotation = optionalTrimmedString(value.QualityAnnotation)
	value.AccessRight = optionalTrimmedString(value.AccessRight)
	keywords, err := normalizeKeywords(value.Keywords)
	if err != nil {
		return DatasetMetadata{}, fmt.Errorf("dataset keywords: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	value.Keywords = keywords
	if err := validateDatasetTemporal(value.TemporalCoverage, value.GeologicalTemporalCoverage); err != nil {
		return DatasetMetadata{}, fmt.Errorf("dataset temporal coverage: %w", err)
	}

	identifiers := make([]Identifier, len(value.PersistentIdentifiers))
	seenIdentifiers := make(map[uuid.UUID]struct{}, len(value.PersistentIdentifiers))
	for i, identifier := range value.PersistentIdentifiers {
		if err := validateIdentifier(identifier, true); err != nil {
			return DatasetMetadata{}, fmt.Errorf("dataset identifier: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		if _, exists := seenIdentifiers[identifier.ID]; exists {
			return DatasetMetadata{}, fmt.Errorf("duplicate dataset identifier %q: %w", identifier.ID, apperror.Default(apperror.CodeValidationFailed))
		}
		seenIdentifiers[identifier.ID] = struct{}{}
		identifiers[i] = identifier
	}
	value.PersistentIdentifiers = identifiers

	if err := validateContactPoints(value.ContactPoints, "dataset contact point"); err != nil {
		return DatasetMetadata{}, fmt.Errorf("dataset contact points: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateOrganizations(value.Providers, "provider"); err != nil {
		return DatasetMetadata{}, fmt.Errorf("dataset providers: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validateCategories(value.Categories, "category"); err != nil {
		return DatasetMetadata{}, fmt.Errorf("dataset categories: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}

	distributions := make([]Distribution, len(value.Distributions))
	for i, distribution := range value.Distributions {
		distribution.License = optionalTrimmedString(distribution.License)
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
	submittedBy *string,
	group Group,
	createdAt, updatedAt time.Time,
	rejectedBy *string,
	rejectedAt *time.Time,
	rejectionNote *string,
	metadata DatasetMetadata,
) (result DatasetVersion, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if datasetID == uuid.Nil || versionID == uuid.Nil {
		return DatasetVersion{}, fmt.Errorf("dataset and version IDs are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if revision < 1 {
		return DatasetVersion{}, fmt.Errorf("revision must be positive: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if !status.Valid() {
		return DatasetVersion{}, fmt.Errorf("dataset status %q is invalid: %w", status, apperror.Default(apperror.CodeValidationFailed))
	}
	actor, err := NewActor(string(createdBy))
	if err != nil {
		return DatasetVersion{}, fmt.Errorf("dataset creator: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	var submittedByActor *Actor
	if submittedBy != nil {
		value, err := NewActor(*submittedBy)
		if err != nil {
			return DatasetVersion{}, fmt.Errorf("dataset submitter: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
		submittedByActor = &value
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
		return DatasetVersion{}, fmt.Errorf("dataset group: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	metadata, err = NewDatasetMetadata(metadata)
	if err != nil {
		return DatasetVersion{}, err
	}
	if createdAt.IsZero() || updatedAt.IsZero() {
		return DatasetVersion{}, fmt.Errorf("dataset timestamps are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	rejectionNote, err = NewRejectionNote(status, rejectionNote)
	if err != nil {
		return DatasetVersion{}, fmt.Errorf("dataset rejection note: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return DatasetVersion{
		DatasetID:     datasetID,
		VersionID:     versionID,
		Revision:      revision,
		Status:        status,
		CreatedBy:     actor,
		SubmittedBy:   submittedByActor,
		Group:         normalizedGroup,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		RejectedBy:    rejectedBy,
		RejectedAt:    rejectedAt,
		RejectionNote: rejectionNote,
		Metadata:      metadata,
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
		Title:                      value.Title,
		Description:                value.Description,
		Keywords:                   value.Keywords,
		AccrualPeriodicity:         value.AccrualPeriodicity,
		QualityAnnotation:          value.QualityAnnotation,
		AccessRight:                value.AccessRight,
		Distributions:              distributions,
		SpatialCoverage:            value.SpatialCoverage,
		TemporalCoverage:           value.TemporalCoverage,
		GeologicalTemporalCoverage: value.GeologicalTemporalCoverage,
		PersistentIdentifiers:      identifiers,
		ContactPointIDs:            contactPointIDs,
		ProviderIDs:                providerIDs,
		CategoryIDs:                categoryIDs,
		SourceCreated:              value.SourceCreated,
		SourceUpdated:              value.SourceUpdated,
	}
}

func NewDistributionInput(value DistributionInput) (DistributionInput, error) {
	value.Title = strings.TrimSpace(value.Title)
	if value.Title == "" {
		return DistributionInput{}, invalidField("title", "Title must not be blank.", nil)
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return DistributionInput{}, invalidField("description", "Description must not be blank.", nil)
	}
	value.License = optionalTrimmedString(value.License)
	if value.WebService != nil && value.DownloadableFile != nil {
		return DistributionInput{}, invalidField("type", "Choose either a downloadable file or a web service.", nil)
	}
	var expected DistributionType
	switch {
	case value.WebService != nil:
		expected = DistributionTypeWebService
	case value.DownloadableFile != nil:
		expected = DistributionTypeDownloadableFile
	default:
		return DistributionInput{}, invalidField("type", "A delivery method is required.", nil)
	}
	if value.Type != "" && value.Type != expected {
		return DistributionInput{}, invalidField("type", "Distribution type must match the selected delivery method.", nil)
	}
	value.Type = expected
	if value.DownloadableFile != nil {
		if strings.TrimSpace(value.DownloadableFile.URL) == "" {
			return DistributionInput{}, invalidField("downloadable_file.url", "URL must not be blank.", nil)
		}
		if strings.TrimSpace(value.DownloadableFile.Format) == "" {
			return DistributionInput{}, invalidField("downloadable_file.format", "Format must not be blank.", nil)
		}
	}
	if value.WebService != nil {
		webService, err := NewWebServiceInput(*value.WebService)
		if err != nil {
			return DistributionInput{}, fmt.Errorf("validate web service: %w", apperror.PrefixTarget(err, "web_service"))
		}
		value.WebService = &webService
	}
	return value, nil
}

func validateDistribution(value Distribution) error {
	if value.ID == uuid.Nil || strings.TrimSpace(value.Title) == "" {
		return fmt.Errorf("ID and title are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if strings.TrimSpace(value.Description) == "" {
		return fmt.Errorf("distribution description is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if !value.Type.Valid() {
		return fmt.Errorf("distribution type is invalid: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	expected, err := value.DeliveryType()
	if err != nil {
		return fmt.Errorf("%w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if value.Type != expected {
		return fmt.Errorf("distribution type does not match delivery method: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value.DownloadableFile != nil && (strings.TrimSpace(value.DownloadableFile.URL) == "" || strings.TrimSpace(value.DownloadableFile.Format) == "") {
		return fmt.Errorf("downloadable file URL and format are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value.WebService != nil {
		if _, err := NewWebService(*value.WebService); err != nil {
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
		input := value.WebService.Input()
		result.WebService = &input
	}
	return result
}

// Input returns caller-shaped fields from persisted web-service metadata.
func (value WebService) Input() WebServiceInput {
	var providerID *uuid.UUID
	if value.Provider != nil {
		providerID = &value.Provider.ID
	}
	contactPointIDs := make([]uuid.UUID, len(value.ContactPoints))
	for i, contact := range value.ContactPoints {
		contactPointIDs[i] = contact.ID
	}
	return WebServiceInput{
		Name:             value.Name,
		Description:      value.Description,
		Type:             value.Type,
		Configuration:    webServiceConfigurationInput(value.Configuration),
		VisibleOn:        value.VisibleOn,
		ProviderID:       providerID,
		Documentation:    value.Documentation,
		SpatialCoverage:  value.SpatialCoverage,
		TemporalCoverage: value.TemporalCoverage,
		ContactPointIDs:  contactPointIDs,
	}
}

func webServiceConfigurationInput(value WebServiceConfiguration) WebServiceConfigurationInput {
	if value == nil {
		return nil
	}
	return value.Input()
}

func (value GenericWebServiceConfiguration) Input() WebServiceConfigurationInput {
	return GenericWebServiceConfigurationInput{
		TemplateURL: value.TemplateURL,
		Method:      value.Method,
		Body:        requestBodyInput(value.Body),
		Parameters:  webServiceParameterInputs(value.Parameters),
		Format:      value.Format,
	}
}

func (value WFSConfiguration) Input() WebServiceConfigurationInput {
	return WFSConfigurationInput{
		GetFeatureTemplateURL: value.GetFeatureTemplateURL,
		Version:               value.Version,
		FeatureTypes:          value.FeatureTypes,
		DefaultFeatureType:    value.DefaultFeatureType,
		OutputFormats:         value.OutputFormats,
		DefaultOutputFormat:   value.DefaultOutputFormat,
		ResultLimit:           value.ResultLimit,
		ResultLimitEditable:   value.ResultLimitEditable,
		Parameters:            webServiceParameterInputs(value.Parameters),
	}
}

func (value WMSConfiguration) Input() WebServiceConfigurationInput {
	return WMSConfigurationInput{
		CapabilitiesURL:          value.CapabilitiesURL,
		DefaultLayer:             value.DefaultLayer,
		Layers:                   value.Layers,
		TransparencyEditable:     value.TransparencyEditable,
		DefaultTransparency:      value.DefaultTransparency,
		GetMapParameters:         webServiceParameterInputs(value.GetMapParameters),
		GetFeatureInfoParameters: webServiceParameterInputs(value.GetFeatureInfoParameters),
	}
}
