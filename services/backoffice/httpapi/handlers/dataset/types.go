package dataset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/datasetlifecycle"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/google/uuid"
)

type datasetDraftRequest struct {
	ID                    uuid.UUID             `json:"id"`
	SourceIRI             string                `json:"source_iri"`
	Title                 string                `json:"title"`
	Description           string                `json:"description"`
	Keywords              string                `json:"keywords"`
	AccrualPeriodicity    *string               `json:"accrual_periodicity"`
	QualityAnnotation     *string               `json:"quality_annotation"`
	AccessRight           *string               `json:"access_right"`
	Distributions         []distributionRequest `json:"distributions"`
	SpatialCoverage       *spatial              `json:"spatial_coverage"`
	TemporalCoverage      *temporal             `json:"temporal_coverage"`
	PersistentIdentifiers []identifier          `json:"persistent_identifiers"`
	ContactPoints         []canonicalReference  `json:"contact_points"`
	Providers             []canonicalReference  `json:"providers"`
	Categories            []canonicalReference  `json:"categories"`
	SourceCreated         *time.Time            `json:"source_created"`
	SourceUpdated         *time.Time            `json:"source_updated"`
}

type createRequest struct {
	GroupID  uuid.UUID           `json:"group_id"`
	Metadata datasetDraftRequest `json:"metadata"`
}

type distributionRequest struct {
	ID               uuid.UUID                 `json:"id"`
	Title            string                    `json:"title"`
	Description      string                    `json:"description"`
	License          *string                   `json:"license"`
	Type             metadata.DistributionType `json:"type"`
	Issued           *time.Time                `json:"issued"`
	Modified         *time.Time                `json:"modified"`
	ByteSize         *int64                    `json:"byte_size"`
	WebService       *webServiceRequest        `json:"web_service"`
	DownloadableFile *downloadableFile         `json:"downloadable_file"`
}

type webServiceRequest struct {
	ID               uuid.UUID            `json:"id"`
	Name             string               `json:"name"`
	Description      string               `json:"description"`
	TemplateURL      string               `json:"template_url"`
	Parameters       []parameterRequest   `json:"parameters"`
	Format           string               `json:"format"`
	Provider         *canonicalReference  `json:"provider"`
	Documentation    *documentation       `json:"documentation"`
	SpatialCoverage  *spatial             `json:"spatial_coverage"`
	TemporalCoverage *temporal            `json:"temporal_coverage"`
	ContactPoints    []canonicalReference `json:"contact_points"`
}

type canonicalReference struct {
	ID uuid.UUID `json:"id"`
}

type parameterRequest struct {
	value metadata.Parameter
}

func (p *parameterRequest) UnmarshalJSON(data []byte) error {
	var header struct {
		Metadata struct {
			Kind metadata.ParameterKind `json:"kind"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return fmt.Errorf("decode parameter metadata: %w", err)
	}

	switch header.Metadata.Kind {
	case metadata.ParameterKindString:
		var value stringParameter
		if err := decodeStrictBytes(data, &value); err != nil {
			return err
		}
		p.value = metadata.StringParameter{
			Metadata:      value.Metadata.domain(),
			Default:       value.Default,
			AllowedValues: value.AllowedValues,
			Pattern:       value.Pattern,
		}
	case metadata.ParameterKindBoolean:
		var value booleanParameter
		if err := decodeStrictBytes(data, &value); err != nil {
			return err
		}
		p.value = metadata.BooleanParameter{
			Metadata: value.Metadata.domain(),
			Default:  value.Default,
		}
	case metadata.ParameterKindInteger:
		var value integerParameter
		if err := decodeStrictBytes(data, &value); err != nil {
			return err
		}
		p.value = metadata.IntegerParameter{
			Metadata:      value.Metadata.domain(),
			Minimum:       value.Minimum,
			Maximum:       value.Maximum,
			Default:       value.Default,
			AllowedValues: value.AllowedValues,
		}
	case metadata.ParameterKindFloat:
		var value floatParameter
		if err := decodeStrictBytes(data, &value); err != nil {
			return err
		}
		p.value = metadata.FloatParameter{
			Metadata:      value.Metadata.domain(),
			Minimum:       value.Minimum,
			Maximum:       value.Maximum,
			Default:       value.Default,
			AllowedValues: value.AllowedValues,
		}
	case metadata.ParameterKindDateTime:
		var value dateTimeParameter
		if err := decodeStrictBytes(data, &value); err != nil {
			return err
		}
		p.value = metadata.DateTimeParameter{
			Metadata: value.Metadata.domain(),
			Minimum:  value.Minimum,
			Maximum:  value.Maximum,
			Default:  value.Default,
			Pattern:  value.Pattern,
		}
	default:
		return fmt.Errorf("unsupported parameter kind %q", header.Metadata.Kind)
	}
	if p.value.Kind() != header.Metadata.Kind || p.value.GetMetadata().Kind != header.Metadata.Kind {
		return fmt.Errorf("parameter kind mismatch: metadata=%q value=%q", header.Metadata.Kind, p.value.Kind())
	}
	return nil
}

func decodeStrictBytes(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}

type parameterMetadata struct {
	ID             uuid.UUID              `json:"id"`
	Kind           metadata.ParameterKind `json:"kind"`
	Variable       string                 `json:"variable"`
	Property       string                 `json:"property"`
	Label          string                 `json:"label"`
	Required       bool                   `json:"required"`
	MultipleValues bool                   `json:"multiple_values"`
	ReadOnly       bool                   `json:"read_only"`
}

func (p parameterMetadata) domain() metadata.ParameterMetadata {
	return metadata.ParameterMetadata{
		ID:             p.ID,
		Kind:           p.Kind,
		Variable:       p.Variable,
		Property:       p.Property,
		Label:          p.Label,
		Required:       p.Required,
		MultipleValues: p.MultipleValues,
		ReadOnly:       p.ReadOnly,
	}
}

type stringParameter struct {
	Metadata      parameterMetadata `json:"metadata"`
	Default       *string           `json:"default"`
	AllowedValues []string          `json:"allowed_values"`
	Pattern       string            `json:"pattern"`
}

func (stringParameter) parameterResponse() {}

type booleanParameter struct {
	Metadata parameterMetadata `json:"metadata"`
	Default  *bool             `json:"default"`
}

func (booleanParameter) parameterResponse() {}

type integerParameter struct {
	Metadata      parameterMetadata `json:"metadata"`
	Minimum       *int64            `json:"minimum"`
	Maximum       *int64            `json:"maximum"`
	Default       *int64            `json:"default"`
	AllowedValues []int64           `json:"allowed_values"`
}

func (integerParameter) parameterResponse() {}

type floatParameter struct {
	Metadata      parameterMetadata `json:"metadata"`
	Minimum       *float64          `json:"minimum"`
	Maximum       *float64          `json:"maximum"`
	Default       *float64          `json:"default"`
	AllowedValues []float64         `json:"allowed_values"`
}

func (floatParameter) parameterResponse() {}

type dateTimeParameter struct {
	Metadata parameterMetadata `json:"metadata"`
	Minimum  *time.Time        `json:"minimum"`
	Maximum  *time.Time        `json:"maximum"`
	Default  *time.Time        `json:"default"`
	Pattern  string            `json:"pattern"`
}

func (dateTimeParameter) parameterResponse() {}

type parameterResponse interface {
	parameterResponse()
}

type downloadableFile struct {
	URL    string `json:"url"`
	Format string `json:"format"`
}

type documentation struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	URI         string  `json:"uri"`
}

type coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type spatial struct {
	Point   *coordinate  `json:"point"`
	Polygon []coordinate `json:"polygon"`
}

type temporal struct {
	Start *time.Time `json:"start"`
	End   *time.Time `json:"end"`
}

type identifier struct {
	ID        uuid.UUID               `json:"id"`
	SourceIRI string                  `json:"source_iri"`
	Type      metadata.IdentifierType `json:"type"`
	Value     string                  `json:"value"`
}

func (r datasetDraftRequest) domain() metadata.Dataset {
	result := metadata.Dataset{
		ID:                    r.ID,
		SourceIRI:             r.SourceIRI,
		Title:                 r.Title,
		Description:           r.Description,
		Keywords:              r.Keywords,
		AccrualPeriodicity:    r.AccrualPeriodicity,
		QualityAnnotation:     r.QualityAnnotation,
		AccessRight:           r.AccessRight,
		Distributions:         make([]metadata.Distribution, 0, len(r.Distributions)),
		SpatialCoverage:       spatialDomain(r.SpatialCoverage),
		TemporalCoverage:      temporalDomain(r.TemporalCoverage),
		PersistentIdentifiers: make([]metadata.Identifier, 0, len(r.PersistentIdentifiers)),
		ContactPoints:         make([]metadata.ContactPoint, 0, len(r.ContactPoints)),
		Providers:             make([]metadata.Organization, 0, len(r.Providers)),
		Categories:            make([]metadata.Category, 0, len(r.Categories)),
		SourceCreated:         r.SourceCreated,
		SourceUpdated:         r.SourceUpdated,
		Status:                "",
		Created:               time.Time{},
		Updated:               time.Time{},
	}
	for _, value := range r.Distributions {
		result.Distributions = append(result.Distributions, value.domain())
	}
	for _, value := range r.PersistentIdentifiers {
		result.PersistentIdentifiers = append(result.PersistentIdentifiers, value.domain())
	}
	for _, value := range r.ContactPoints {
		result.ContactPoints = append(result.ContactPoints, metadata.ContactPoint{
			ID:        value.ID,
			SourceIRI: "",
			Email:     "",
			Role:      "",
		})
	}
	for _, value := range r.Providers {
		result.Providers = append(result.Providers, metadata.Organization{
			ID:            value.ID,
			SourceIRI:     "",
			Name:          "",
			Street:        nil,
			Locality:      nil,
			PostalCode:    nil,
			Country:       nil,
			LogoURL:       nil,
			PageURL:       nil,
			Email:         nil,
			LEICode:       nil,
			ContactPoints: nil,
			Identifiers:   nil,
		})
	}
	for _, value := range r.Categories {
		result.Categories = append(result.Categories, metadata.Category{
			ID:        value.ID,
			SourceIRI: "",
			Name:      "",
			Scheme: metadata.CategoryScheme{
				ID:          uuid.Nil,
				SourceIRI:   "",
				Name:        "",
				Description: nil,
				Label:       nil,
				Logo:        nil,
				Page:        nil,
				Color:       nil,
				Position:    0,
				Concept:     "",
			},
			Children: nil,
		})
	}
	return result
}

func (r distributionRequest) domain() metadata.Distribution {
	result := metadata.Distribution{
		ID:               r.ID,
		Title:            r.Title,
		Description:      r.Description,
		License:          r.License,
		Type:             r.Type,
		Issued:           r.Issued,
		Modified:         r.Modified,
		ByteSize:         r.ByteSize,
		WebService:       nil,
		DownloadableFile: nil,
	}
	if r.WebService != nil {
		value := r.WebService.domain()
		result.WebService = &value
	}
	if r.DownloadableFile != nil {
		result.DownloadableFile = &metadata.DownloadableFile{
			URL:    r.DownloadableFile.URL,
			Format: r.DownloadableFile.Format,
		}
	}
	return result
}

func (r webServiceRequest) domain() metadata.WebService {
	result := metadata.WebService{
		ID:               r.ID,
		Name:             r.Name,
		Description:      r.Description,
		TemplateURL:      r.TemplateURL,
		Parameters:       make([]metadata.Parameter, 0, len(r.Parameters)),
		Format:           r.Format,
		Provider:         nil,
		Documentation:    nil,
		SpatialCoverage:  spatialDomain(r.SpatialCoverage),
		TemporalCoverage: temporalDomain(r.TemporalCoverage),
		ContactPoints:    make([]metadata.ContactPoint, 0, len(r.ContactPoints)),
	}
	for _, value := range r.Parameters {
		result.Parameters = append(result.Parameters, value.value)
	}
	if r.Provider != nil {
		result.Provider = &metadata.Organization{
			ID:            r.Provider.ID,
			SourceIRI:     "",
			Name:          "",
			Street:        nil,
			Locality:      nil,
			PostalCode:    nil,
			Country:       nil,
			LogoURL:       nil,
			PageURL:       nil,
			Email:         nil,
			LEICode:       nil,
			ContactPoints: nil,
			Identifiers:   nil,
		}
	}
	if r.Documentation != nil {
		result.Documentation = &metadata.Documentation{
			Title:       r.Documentation.Title,
			Description: r.Documentation.Description,
			URI:         r.Documentation.URI,
		}
	}
	for _, value := range r.ContactPoints {
		result.ContactPoints = append(result.ContactPoints, metadata.ContactPoint{
			ID:        value.ID,
			SourceIRI: "",
			Email:     "",
			Role:      "",
		})
	}
	return result
}

func spatialDomain(value *spatial) *metadata.Spatial {
	if value == nil {
		return nil
	}
	result := &metadata.Spatial{
		Point:   nil,
		Polygon: make([]metadata.Coordinate, 0, len(value.Polygon)),
	}
	if value.Point != nil {
		result.Point = &metadata.Coordinate{
			Latitude:  value.Point.Latitude,
			Longitude: value.Point.Longitude,
		}
	}
	for _, coordinate := range value.Polygon {
		result.Polygon = append(result.Polygon, metadata.Coordinate{
			Latitude:  coordinate.Latitude,
			Longitude: coordinate.Longitude,
		})
	}
	return result
}

func temporalDomain(value *temporal) *metadata.Temporal {
	if value == nil {
		return nil
	}
	return &metadata.Temporal{
		Start: value.Start,
		End:   value.End,
	}
}

func (i identifier) domain() metadata.Identifier {
	return metadata.Identifier{
		ID:        i.ID,
		SourceIRI: i.SourceIRI,
		Type:      i.Type,
		Value:     i.Value,
	}
}

type record struct {
	ResourceID uuid.UUID              `json:"resource_id"`
	VersionID  uuid.UUID              `json:"version_id"`
	Revision   int32                  `json:"revision"`
	Status     metadata.DatasetStatus `json:"status"`
	CreatedBy  string                 `json:"created_by"`
	Group      categoryScheme         `json:"group"`
	Metadata   metadataResponse       `json:"metadata"`
}

type metadataResponse struct {
	ID                    uuid.UUID              `json:"id"`
	SourceIRI             string                 `json:"source_iri"`
	Title                 string                 `json:"title"`
	Description           string                 `json:"description"`
	Keywords              string                 `json:"keywords"`
	AccrualPeriodicity    *string                `json:"accrual_periodicity"`
	QualityAnnotation     *string                `json:"quality_annotation"`
	AccessRight           *string                `json:"access_right"`
	Distributions         []distribution         `json:"distributions"`
	SpatialCoverage       *spatial               `json:"spatial_coverage"`
	TemporalCoverage      *temporal              `json:"temporal_coverage"`
	PersistentIdentifiers []identifier           `json:"persistent_identifiers"`
	ContactPoints         *[]contactPoint        `json:"contact_points,omitempty"`
	Providers             []organization         `json:"providers"`
	Categories            []category             `json:"categories"`
	SourceCreated         *time.Time             `json:"source_created"`
	SourceUpdated         *time.Time             `json:"source_updated"`
	Status                metadata.DatasetStatus `json:"status"`
	Created               time.Time              `json:"created"`
	Updated               time.Time              `json:"updated"`
}

type distribution struct {
	ID               uuid.UUID                 `json:"id"`
	Title            string                    `json:"title"`
	Description      string                    `json:"description"`
	License          *string                   `json:"license"`
	Type             metadata.DistributionType `json:"type"`
	Issued           *time.Time                `json:"issued"`
	Modified         *time.Time                `json:"modified"`
	ByteSize         *int64                    `json:"byte_size"`
	WebService       *webService               `json:"web_service"`
	DownloadableFile *downloadableFile         `json:"downloadable_file"`
}

type webService struct {
	ID               uuid.UUID           `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	TemplateURL      string              `json:"template_url"`
	Parameters       []parameterResponse `json:"parameters"`
	Format           string              `json:"format"`
	Provider         *organization       `json:"provider"`
	Documentation    *documentation      `json:"documentation"`
	SpatialCoverage  *spatial            `json:"spatial_coverage"`
	TemporalCoverage *temporal           `json:"temporal_coverage"`
	ContactPoints    *[]contactPoint     `json:"contact_points,omitempty"`
}

type organization struct {
	ID            uuid.UUID       `json:"id"`
	SourceIRI     string          `json:"source_iri"`
	Name          string          `json:"name"`
	Street        *string         `json:"street"`
	Locality      *string         `json:"locality"`
	PostalCode    *string         `json:"postal_code"`
	Country       *string         `json:"country"`
	LogoURL       *string         `json:"logo_url"`
	PageURL       *string         `json:"page_url"`
	Email         *string         `json:"email"`
	LEICode       *string         `json:"lei_code"`
	ContactPoints *[]contactPoint `json:"contact_points,omitempty"`
	Identifiers   []identifier    `json:"identifiers"`
}

type category struct {
	ID        uuid.UUID  `json:"id"`
	SourceIRI string     `json:"source_iri"`
	Name      string     `json:"name"`
	Children  []category `json:"children"`
}

type categoryScheme struct {
	ID        uuid.UUID                `json:"id"`
	SourceIRI string                   `json:"source_iri"`
	Name      string                   `json:"name"`
	Label     *string                  `json:"label"`
	Logo      *string                  `json:"logo"`
	Page      *string                  `json:"page"`
	Color     *string                  `json:"color"`
	Position  int32                    `json:"position"`
	Concept   metadata.TopLevelConcept `json:"concept"`
}

type contactPoint struct {
	ID        uuid.UUID                 `json:"id"`
	SourceIRI string                    `json:"source_iri"`
	Email     string                    `json:"email"`
	Role      metadata.ContactPointRole `json:"role"`
}

func datasetResponse(value datasetlifecycle.Dataset, includeContacts bool) record {
	return record{
		ResourceID: value.DatasetID,
		VersionID:  value.VersionID,
		Revision:   value.Revision,
		Status:     value.Status,
		CreatedBy:  value.CreatedBy,
		Group:      categorySchemeDTO(value.Group),
		Metadata:   datasetMetadataDTO(value.Metadata, includeContacts),
	}
}

func datasetMetadataDTO(value metadata.Dataset, includeContacts bool) metadataResponse {
	result := metadataResponse{
		ID:                    value.ID,
		SourceIRI:             value.SourceIRI,
		Title:                 value.Title,
		Description:           value.Description,
		Keywords:              value.Keywords,
		AccrualPeriodicity:    value.AccrualPeriodicity,
		QualityAnnotation:     value.QualityAnnotation,
		AccessRight:           value.AccessRight,
		Distributions:         make([]distribution, 0, len(value.Distributions)),
		SpatialCoverage:       spatialResponse(value.SpatialCoverage),
		TemporalCoverage:      temporalResponse(value.TemporalCoverage),
		PersistentIdentifiers: make([]identifier, 0, len(value.PersistentIdentifiers)),
		ContactPoints:         nil,
		Providers:             make([]organization, 0, len(value.Providers)),
		Categories:            make([]category, 0, len(value.Categories)),
		SourceCreated:         value.SourceCreated,
		SourceUpdated:         value.SourceUpdated,
		Status:                value.Status,
		Created:               value.Created,
		Updated:               value.Updated,
	}
	for _, item := range value.Distributions {
		result.Distributions = append(result.Distributions, distributionDTO(item, includeContacts))
	}
	for _, item := range value.PersistentIdentifiers {
		result.PersistentIdentifiers = append(result.PersistentIdentifiers, identifierResponse(item))
	}
	if includeContacts {
		contacts := make([]contactPoint, 0, len(value.ContactPoints))
		for _, item := range value.ContactPoints {
			contacts = append(contacts, contactPointDTO(item))
		}
		result.ContactPoints = &contacts
	}
	for _, item := range value.Providers {
		result.Providers = append(result.Providers, organizationDTO(item, includeContacts))
	}
	for _, item := range value.Categories {
		result.Categories = append(result.Categories, categoryDTO(item))
	}
	return result
}

func distributionDTO(value metadata.Distribution, includeContacts bool) distribution {
	result := distribution{
		ID:               value.ID,
		Title:            value.Title,
		Description:      value.Description,
		License:          value.License,
		Type:             value.Type,
		Issued:           value.Issued,
		Modified:         value.Modified,
		ByteSize:         value.ByteSize,
		WebService:       nil,
		DownloadableFile: nil,
	}
	if value.WebService != nil {
		item := webServiceDTO(*value.WebService, includeContacts)
		result.WebService = &item
	}
	if value.DownloadableFile != nil {
		result.DownloadableFile = &downloadableFile{
			URL:    value.DownloadableFile.URL,
			Format: value.DownloadableFile.Format,
		}
	}
	return result
}

func webServiceDTO(value metadata.WebService, includeContacts bool) webService {
	result := webService{
		ID:               value.ID,
		Name:             value.Name,
		Description:      value.Description,
		TemplateURL:      value.TemplateURL,
		Parameters:       make([]parameterResponse, 0, len(value.Parameters)),
		Format:           value.Format,
		Provider:         nil,
		Documentation:    nil,
		SpatialCoverage:  spatialResponse(value.SpatialCoverage),
		TemporalCoverage: temporalResponse(value.TemporalCoverage),
		ContactPoints:    nil,
	}
	for _, item := range value.Parameters {
		result.Parameters = append(result.Parameters, parameterDTO(item))
	}
	if value.Provider != nil {
		item := organizationDTO(*value.Provider, includeContacts)
		result.Provider = &item
	}
	if value.Documentation != nil {
		result.Documentation = &documentation{
			Title:       value.Documentation.Title,
			Description: value.Documentation.Description,
			URI:         value.Documentation.URI,
		}
	}
	if includeContacts {
		contacts := make([]contactPoint, 0, len(value.ContactPoints))
		for _, item := range value.ContactPoints {
			contacts = append(contacts, contactPointDTO(item))
		}
		result.ContactPoints = &contacts
	}
	return result
}

func parameterDTO(value metadata.Parameter) parameterResponse {
	metadataValue := parameterMetadataResponse(value.GetMetadata())
	switch item := value.(type) {
	case metadata.StringParameter:
		return stringParameter{
			Metadata:      metadataValue,
			Default:       item.Default,
			AllowedValues: item.AllowedValues,
			Pattern:       item.Pattern,
		}
	case metadata.BooleanParameter:
		return booleanParameter{
			Metadata: metadataValue,
			Default:  item.Default,
		}
	case metadata.IntegerParameter:
		return integerParameter{
			Metadata:      metadataValue,
			Minimum:       item.Minimum,
			Maximum:       item.Maximum,
			Default:       item.Default,
			AllowedValues: item.AllowedValues,
		}
	case metadata.FloatParameter:
		return floatParameter{
			Metadata:      metadataValue,
			Minimum:       item.Minimum,
			Maximum:       item.Maximum,
			Default:       item.Default,
			AllowedValues: item.AllowedValues,
		}
	case metadata.DateTimeParameter:
		return dateTimeParameter{
			Metadata: metadataValue,
			Minimum:  item.Minimum,
			Maximum:  item.Maximum,
			Default:  item.Default,
			Pattern:  item.Pattern,
		}
	default:
		return nil
	}
}

func parameterMetadataResponse(value metadata.ParameterMetadata) parameterMetadata {
	return parameterMetadata{
		ID:             value.ID,
		Kind:           value.Kind,
		Variable:       value.Variable,
		Property:       value.Property,
		Label:          value.Label,
		Required:       value.Required,
		MultipleValues: value.MultipleValues,
		ReadOnly:       value.ReadOnly,
	}
}

func organizationDTO(value metadata.Organization, includeContacts bool) organization {
	result := organization{
		ID:            value.ID,
		SourceIRI:     value.SourceIRI,
		Name:          value.Name,
		Street:        value.Street,
		Locality:      value.Locality,
		PostalCode:    value.PostalCode,
		Country:       value.Country,
		LogoURL:       value.LogoURL,
		PageURL:       value.PageURL,
		Email:         value.Email,
		LEICode:       value.LEICode,
		ContactPoints: nil,
		Identifiers:   make([]identifier, 0, len(value.Identifiers)),
	}
	for _, item := range value.Identifiers {
		result.Identifiers = append(result.Identifiers, identifierResponse(item))
	}
	for _, item := range value.ContactPoints {
		if includeContacts {
			if result.ContactPoints == nil {
				result.ContactPoints = new([]contactPoint)
			}
			*result.ContactPoints = append(*result.ContactPoints, contactPointDTO(item))
		}
	}
	return result
}

func categoryDTO(value metadata.Category) category {
	result := category{
		ID:        value.ID,
		SourceIRI: value.SourceIRI,
		Name:      value.Name,
		Children:  make([]category, 0, len(value.Children)),
	}
	for _, item := range value.Children {
		result.Children = append(result.Children, categoryDTO(item))
	}
	return result
}

func categorySchemeDTO(value metadata.CategoryScheme) categoryScheme {
	return categoryScheme{
		ID:        value.ID,
		SourceIRI: value.SourceIRI,
		Name:      value.Name,
		Label:     value.Label,
		Logo:      value.Logo,
		Page:      value.Page,
		Color:     value.Color,
		Position:  value.Position,
		Concept:   value.Concept,
	}
}

func contactPointDTO(value metadata.ContactPoint) contactPoint {
	return contactPoint{
		ID:        value.ID,
		SourceIRI: value.SourceIRI,
		Email:     value.Email,
		Role:      value.Role,
	}
}

func identifierResponse(value metadata.Identifier) identifier {
	return identifier{
		ID:        value.ID,
		SourceIRI: value.SourceIRI,
		Type:      value.Type,
		Value:     value.Value,
	}
}

func spatialResponse(value *metadata.Spatial) *spatial {
	if value == nil {
		return nil
	}
	result := &spatial{
		Point:   nil,
		Polygon: make([]coordinate, 0, len(value.Polygon)),
	}
	if value.Point != nil {
		result.Point = &coordinate{
			Latitude:  value.Point.Latitude,
			Longitude: value.Point.Longitude,
		}
	}
	for _, item := range value.Polygon {
		result.Polygon = append(result.Polygon, coordinate{
			Latitude:  item.Latitude,
			Longitude: item.Longitude,
		})
	}
	return result
}

func temporalResponse(value *metadata.Temporal) *temporal {
	if value == nil {
		return nil
	}
	return &temporal{
		Start: value.Start,
		End:   value.End,
	}
}
