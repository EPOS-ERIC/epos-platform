package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// TODO: check the whole apis for inconsistencies and try to clean it up. like we could have the draft bodies be struct and then embed them into normal get responses that adds the ids and stuff

type DatasetDraftMetadata struct {
	Title                 string              `json:"title"`
	Description           *string             `json:"description,omitempty"`
	Keywords              *string             `json:"keywords,omitempty"`
	AccrualPeriodicity    *string             `json:"accrual_periodicity,omitempty"`
	QualityAnnotation     *string             `json:"quality_annotation,omitempty"`
	AccessRight           *string             `json:"access_right,omitempty"`
	Distributions         []DistributionDraft `json:"distributions,omitempty"`
	SpatialCoverage       *SpatialCoverage    `json:"spatial_coverage,omitempty"`
	TemporalCoverage      *TemporalCoverage   `json:"temporal_coverage,omitempty"`
	PersistentIdentifiers []IdentifierDraft   `json:"persistent_identifiers,omitempty"`
	ContactPoints         []string            `json:"contact_points,omitempty"`
	Providers             []string            `json:"providers,omitempty"`
	Categories            []string            `json:"categories,omitempty"`
	Created               *time.Time          `json:"created,omitempty"`
	Updated               *time.Time          `json:"updated,omitempty"`
}

type SoftwareApplicationDraftMetadata struct {
	Name             string            `json:"name"`
	Description      *string           `json:"description,omitempty"`
	Keywords         *string           `json:"keywords,omitempty"`
	License          *string           `json:"license,omitempty"`
	Version          *string           `json:"version,omitempty"`
	DownloadURL      *string           `json:"download_url,omitempty"`
	Requirements     *string           `json:"requirements,omitempty"`
	OperatingSystem  *string           `json:"operating_system,omitempty"`
	MainEntityOfPage *string           `json:"main_entity_of_page,omitempty"`
	Creators         []string          `json:"creators,omitempty"`
	Identifiers      []IdentifierDraft `json:"identifiers,omitempty"`
	Categories       []string          `json:"categories,omitempty"`
	ContactPoints    []string          `json:"contact_points,omitempty"`
}

type SoftwareSourceCodeDraftMetadata struct {
	Name                 string            `json:"name"`
	Description          *string           `json:"description,omitempty"`
	Keywords             *string           `json:"keywords,omitempty"`
	Repository           *string           `json:"repository,omitempty"`
	ProgrammingLanguage  *string           `json:"programming_language,omitempty"`
	License              *string           `json:"license,omitempty"`
	SoftwareRequirements *string           `json:"software_requirements,omitempty"`
	RuntimePlatform      *string           `json:"runtime_platform,omitempty"`
	MainEntityOfPage     *string           `json:"main_entity_of_page,omitempty"`
	Version              *string           `json:"version,omitempty"`
	Creators             []string          `json:"creators,omitempty"`
	Identifiers          []IdentifierDraft `json:"identifiers,omitempty"`
	Categories           []string          `json:"categories,omitempty"`
	ContactPoints        []string          `json:"contact_points,omitempty"`
}

type IdentifierDraft struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Identifier struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type DistributionDraft struct {
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	License          *string           `json:"license,omitempty"`
	Type             string            `json:"type"`
	Issued           *time.Time        `json:"issued,omitempty"`
	Modified         *time.Time        `json:"modified,omitempty"`
	ByteSize         *int64            `json:"byte_size,omitempty"`
	WebService       *WebServiceDraft  `json:"web_service,omitempty"`
	DownloadableFile *DownloadableFile `json:"downloadable_file,omitempty"`
}

type WebServiceDraft struct {
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	TemplateURL      string            `json:"template_url"`
	Parameters       []ParameterDraft  `json:"parameters,omitempty"`
	Format           string            `json:"format"`
	Provider         *string           `json:"provider,omitempty"`
	Documentation    *Documentation    `json:"documentation,omitempty"`
	SpatialCoverage  *SpatialCoverage  `json:"spatial_coverage,omitempty"`
	TemporalCoverage *TemporalCoverage `json:"temporal_coverage,omitempty"`
	ContactPoints    []string          `json:"contact_points,omitempty"`
}

func (w *WebServiceDraft) UnmarshalJSON(data []byte) error {
	var value struct {
		Name             string            `json:"name"`
		Description      string            `json:"description"`
		TemplateURL      string            `json:"template_url"`
		Parameters       []json.RawMessage `json:"parameters"`
		Format           string            `json:"format"`
		Provider         *string           `json:"provider"`
		Documentation    *Documentation    `json:"documentation"`
		SpatialCoverage  *SpatialCoverage  `json:"spatial_coverage"`
		TemporalCoverage *TemporalCoverage `json:"temporal_coverage"`
		ContactPoints    []string          `json:"contact_points"`
	}
	if err := DecodeDraftMetadata(data, &value); err != nil {
		return err
	}

	parameters := make([]ParameterDraft, 0, len(value.Parameters))
	for _, raw := range value.Parameters {
		parameter, err := decodeParameterDraft(raw)
		if err != nil {
			return err
		}
		parameters = append(parameters, parameter)
	}

	*w = WebServiceDraft{
		Name:             value.Name,
		Description:      value.Description,
		TemplateURL:      value.TemplateURL,
		Parameters:       parameters,
		Format:           value.Format,
		Provider:         value.Provider,
		Documentation:    value.Documentation,
		SpatialCoverage:  value.SpatialCoverage,
		TemporalCoverage: value.TemporalCoverage,
		ContactPoints:    value.ContactPoints,
	}
	return nil
}

func decodeParameterDraft(data json.RawMessage) (ParameterDraft, error) {
	var header struct {
		Kind     string `json:"kind"`
		Metadata struct {
			Kind string `json:"kind"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}
	if header.Kind == "" {
		return nil, fmt.Errorf("parameter kind is required")
	}
	if header.Metadata.Kind == "" {
		return nil, fmt.Errorf("parameter metadata kind is required")
	}
	if header.Kind != header.Metadata.Kind {
		return nil, fmt.Errorf("parameter kind mismatch: kind=%q metadata=%q", header.Kind, header.Metadata.Kind)
	}

	switch header.Kind {
	case "STRING":
		var value StringParameterDraft
		return value, DecodeDraftMetadata(data, &value)
	case "BOOLEAN":
		var value BooleanParameterDraft
		return value, DecodeDraftMetadata(data, &value)
	case "INTEGER":
		var value IntegerParameterDraft
		return value, DecodeDraftMetadata(data, &value)
	case "FLOAT":
		var value FloatParameterDraft
		return value, DecodeDraftMetadata(data, &value)
	case "DATETIME":
		var value DateTimeParameterDraft
		return value, DecodeDraftMetadata(data, &value)
	default:
		return nil, fmt.Errorf("unsupported parameter kind %q", header.Kind)
	}
}

// DecodeDraftMetadata rejects fields outside the public draft contract.
func DecodeDraftMetadata(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}

type DownloadableFile struct {
	URL    string `json:"url"`
	Format string `json:"format"`
}

type Documentation struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	URI         string  `json:"uri"`
}

type SpatialCoverage struct {
	Point   *Coordinate  `json:"point,omitempty"`
	Polygon []Coordinate `json:"polygon,omitempty"`
}

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type TemporalCoverage struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

type CreateDatasetResourceRequest struct {
	ResourceType string               `json:"resource_type"`
	Metadata     DatasetDraftMetadata `json:"metadata"`
}

type CreateSoftwareApplicationResourceRequest struct {
	ResourceType string                           `json:"resource_type"`
	Metadata     SoftwareApplicationDraftMetadata `json:"metadata"`
}

type CreateSoftwareSourceCodeResourceRequest struct {
	ResourceType string                          `json:"resource_type"`
	Metadata     SoftwareSourceCodeDraftMetadata `json:"metadata"`
}

type ReplaceDatasetDraftRequest struct {
	ResourceType string               `json:"resource_type"`
	Revision     int32                `json:"revision"`
	Metadata     DatasetDraftMetadata `json:"metadata"`
}

type ReplaceSoftwareApplicationDraftRequest struct {
	ResourceType string                           `json:"resource_type"`
	Revision     int32                            `json:"revision"`
	Metadata     SoftwareApplicationDraftMetadata `json:"metadata"`
}

type ReplaceSoftwareSourceCodeDraftRequest struct {
	ResourceType string                          `json:"resource_type"`
	Revision     int32                           `json:"revision"`
	Metadata     SoftwareSourceCodeDraftMetadata `json:"metadata"`
}

type RejectResourceRequest struct {
	Note string `json:"note"`
}

type ResourceVersion struct {
	ResourceID string                  `json:"resource_id"`
	VersionID  string                  `json:"version_id"`
	Revision   int32                   `json:"revision"`
	Status     string                  `json:"status"`
	CreatedBy  string                  `json:"created_by"`
	GroupID    string                  `json:"group_id"`
	GroupName  string                  `json:"group_name"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedAt  time.Time               `json:"updated_at"`
	Metadata   ResourceVersionMetadata `json:"metadata"`
}

type ResourceVersionMetadata interface {
	resourceVersionMetadata()
}

type DatasetResourceMetadata struct {
	ID                    string                     `json:"id"`
	Title                 string                     `json:"title"`
	Description           string                     `json:"description"`
	Keywords              string                     `json:"keywords"`
	AccrualPeriodicity    *string                    `json:"accrual_periodicity"`
	QualityAnnotation     *string                    `json:"quality_annotation"`
	AccessRight           *string                    `json:"access_right"`
	Distributions         []Distribution             `json:"distributions"`
	SpatialCoverage       *SpatialCoverage           `json:"spatial_coverage,omitempty"`
	TemporalCoverage      *TemporalCoverage          `json:"temporal_coverage,omitempty"`
	PersistentIdentifiers []Identifier               `json:"persistent_identifiers"`
	ContactPoints         []string                   `json:"contact_points"`
	Providers             []ResourceVersionReference `json:"providers"`
	Categories            []ResourceVersionReference `json:"categories"`
	Created               *time.Time                 `json:"created"`
	Updated               *time.Time                 `json:"updated"`
}

func (DatasetResourceMetadata) resourceVersionMetadata() {}

type SoftwareApplicationResourceMetadata struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	Description      string                     `json:"description"`
	MainEntityOfPage *string                    `json:"main_entity_of_page"`
	License          *string                    `json:"license"`
	Version          *string                    `json:"version"`
	Keywords         string                     `json:"keywords"`
	DownloadURL      *string                    `json:"download_url"`
	Requirements     *string                    `json:"requirements"`
	OperatingSystem  *string                    `json:"operating_system"`
	Creators         []ResourceVersionReference `json:"creators"`
	Identifiers      []Identifier               `json:"identifiers"`
	ContactPoints    []string                   `json:"contact_points"`
	Categories       []ResourceVersionReference `json:"categories"`
}

func (SoftwareApplicationResourceMetadata) resourceVersionMetadata() {}

type SoftwareSourceCodeResourceMetadata struct {
	ID                   string                     `json:"id"`
	Name                 string                     `json:"name"`
	Description          string                     `json:"description"`
	ProgrammingLanguage  *string                    `json:"programming_language"`
	Keywords             string                     `json:"keywords"`
	Repository           *string                    `json:"repository"`
	License              *string                    `json:"license"`
	SoftwareRequirements *string                    `json:"software_requirements"`
	RuntimePlatform      *string                    `json:"runtime_platform"`
	MainEntityOfPage     *string                    `json:"main_entity_of_page"`
	Version              *string                    `json:"version"`
	Creators             []ResourceVersionReference `json:"creators"`
	Identifiers          []Identifier               `json:"identifiers"`
	ContactPoints        []string                   `json:"contact_points"`
	Categories           []ResourceVersionReference `json:"categories"`
}

func (SoftwareSourceCodeResourceMetadata) resourceVersionMetadata() {}

type ResourceVersionReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Distribution interface {
	distribution()
}

type WebServiceDistribution struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	License     *string    `json:"license"`
	Type        string     `json:"type"`
	Issued      *time.Time `json:"issued"`
	Modified    *time.Time `json:"modified"`
	ByteSize    *int64     `json:"byte_size"`
	WebService  WebService `json:"web_service"`
}

func (WebServiceDistribution) distribution() {}

type DownloadableFileDistribution struct {
	ID               string           `json:"id"`
	Title            string           `json:"title"`
	Description      string           `json:"description"`
	License          *string          `json:"license"`
	Type             string           `json:"type"`
	Issued           *time.Time       `json:"issued"`
	Modified         *time.Time       `json:"modified"`
	ByteSize         *int64           `json:"byte_size"`
	DownloadableFile DownloadableFile `json:"downloadable_file"`
}

func (DownloadableFileDistribution) distribution() {}

type WebService struct {
	ID               string                    `json:"id"`
	Name             string                    `json:"name"`
	Description      string                    `json:"description"`
	TemplateURL      string                    `json:"template_url"`
	Parameters       []Parameter               `json:"parameters,omitempty"`
	Format           string                    `json:"format"`
	Provider         *ResourceVersionReference `json:"provider,omitempty"`
	Documentation    *Documentation            `json:"documentation,omitempty"`
	SpatialCoverage  *SpatialCoverage          `json:"spatial_coverage,omitempty"`
	TemporalCoverage *TemporalCoverage         `json:"temporal_coverage,omitempty"`
	ContactPoints    []string                  `json:"contact_points,omitempty"`
}

type Parameter interface {
	parameter()
}

type ParameterDraft interface {
	parameterDraft()
}

type ParameterDraftMetadata struct {
	Kind           string `json:"kind"`
	Variable       string `json:"variable"`
	Property       string `json:"property"`
	Label          string `json:"label"`
	Required       bool   `json:"required"`
	MultipleValues bool   `json:"multiple_values"`
	ReadOnly       bool   `json:"read_only"`
}

type StringParameterDraft struct {
	Kind          string                 `json:"kind"`
	Metadata      ParameterDraftMetadata `json:"metadata"`
	Default       *string                `json:"default"`
	AllowedValues []string               `json:"allowed_values"`
	Pattern       string                 `json:"pattern"`
}

func (StringParameterDraft) parameterDraft() {}

type BooleanParameterDraft struct {
	Kind     string                 `json:"kind"`
	Metadata ParameterDraftMetadata `json:"metadata"`
	Default  *bool                  `json:"default"`
}

func (BooleanParameterDraft) parameterDraft() {}

type IntegerParameterDraft struct {
	Kind          string                 `json:"kind"`
	Metadata      ParameterDraftMetadata `json:"metadata"`
	Minimum       *int64                 `json:"minimum"`
	Maximum       *int64                 `json:"maximum"`
	Default       *int64                 `json:"default"`
	AllowedValues []int64                `json:"allowed_values"`
}

func (IntegerParameterDraft) parameterDraft() {}

type FloatParameterDraft struct {
	Kind          string                 `json:"kind"`
	Metadata      ParameterDraftMetadata `json:"metadata"`
	Minimum       *float64               `json:"minimum"`
	Maximum       *float64               `json:"maximum"`
	Default       *float64               `json:"default"`
	AllowedValues []float64              `json:"allowed_values"`
}

func (FloatParameterDraft) parameterDraft() {}

type DateTimeParameterDraft struct {
	Kind     string                 `json:"kind"`
	Metadata ParameterDraftMetadata `json:"metadata"`
	Minimum  *time.Time             `json:"minimum"`
	Maximum  *time.Time             `json:"maximum"`
	Default  *time.Time             `json:"default"`
	Pattern  string                 `json:"pattern"`
}

func (DateTimeParameterDraft) parameterDraft() {}

type ParameterMetadata struct {
	ID             string `json:"id"`
	Variable       string `json:"variable"`
	Property       string `json:"property"`
	Label          string `json:"label"`
	Required       bool   `json:"required"`
	MultipleValues bool   `json:"multiple_values"`
	ReadOnly       bool   `json:"read_only"`
}

type StringParameter struct {
	Kind          string            `json:"kind"`
	Metadata      ParameterMetadata `json:"metadata"`
	Default       *string           `json:"default"`
	AllowedValues []string          `json:"allowed_values"`
	Pattern       string            `json:"pattern"`
}

func (StringParameter) parameter() {}

type BooleanParameter struct {
	Kind     string            `json:"kind"`
	Metadata ParameterMetadata `json:"metadata"`
	Default  *bool             `json:"default"`
}

func (BooleanParameter) parameter() {}

type IntegerParameter struct {
	Kind          string            `json:"kind"`
	Metadata      ParameterMetadata `json:"metadata"`
	Minimum       *int64            `json:"minimum"`
	Maximum       *int64            `json:"maximum"`
	Default       *int64            `json:"default"`
	AllowedValues []int64           `json:"allowed_values"`
}

func (IntegerParameter) parameter() {}

type FloatParameter struct {
	Kind          string            `json:"kind"`
	Metadata      ParameterMetadata `json:"metadata"`
	Minimum       *float64          `json:"minimum"`
	Maximum       *float64          `json:"maximum"`
	Default       *float64          `json:"default"`
	AllowedValues []float64         `json:"allowed_values"`
}

func (FloatParameter) parameter() {}

type DateTimeParameter struct {
	Kind     string            `json:"kind"`
	Metadata ParameterMetadata `json:"metadata"`
	Minimum  *time.Time        `json:"minimum"`
	Maximum  *time.Time        `json:"maximum"`
	Default  *time.Time        `json:"default"`
	Pattern  string            `json:"pattern"`
}

func (DateTimeParameter) parameter() {}

type WriteResult struct {
	ResourceType ResourceType     `json:"resourceType"`
	ID           string           `json:"id"`
	Status       VersioningStatus `json:"status"`
}
