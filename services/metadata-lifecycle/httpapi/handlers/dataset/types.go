package dataset

import (
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type datasetDraftRequest struct {
	Title                 string                `json:"title"`
	Description           string                `json:"description"`
	Keywords              string                `json:"keywords"`
	AccrualPeriodicity    *string               `json:"accrual_periodicity"`
	QualityAnnotation     *string               `json:"quality_annotation"`
	AccessRight           *string               `json:"access_right"`
	Distributions         []distributionRequest `json:"distributions"`
	SpatialCoverage       *spatial              `json:"spatial_coverage"`
	TemporalCoverage      *temporal             `json:"temporal_coverage"`
	PersistentIdentifiers []identifierDraft     `json:"persistent_identifiers"`
	ContactPoints         []uuid.UUID           `json:"contact_points"`
	Providers             []uuid.UUID           `json:"providers"`
	Categories            []uuid.UUID           `json:"categories"`
	Created               *time.Time            `json:"created"`
	Updated               *time.Time            `json:"updated"`
}

type createRequest struct {
	ResourceType string              `json:"resource_type"`
	Metadata     datasetDraftRequest `json:"metadata"`
}

type replaceDraftRequest struct {
	ResourceType string              `json:"resource_type"`
	Revision     int32               `json:"revision"`
	Metadata     datasetDraftRequest `json:"metadata"`
}

type rejectionRequest struct {
	Note string `json:"note"`
}

type distributionRequest struct {
	Title            string                  `json:"title"`
	Description      string                  `json:"description"`
	License          *string                 `json:"license"`
	Type             domain.DistributionType `json:"type"`
	Issued           *time.Time              `json:"issued"`
	Modified         *time.Time              `json:"modified"`
	ByteSize         *int64                  `json:"byte_size"`
	WebService       *webServiceRequest      `json:"web_service"`
	DownloadableFile *downloadableFile       `json:"downloadable_file"`
}

type webServiceRequest struct {
	Name             string             `json:"name"`
	Description      string             `json:"description"`
	TemplateURL      string             `json:"template_url"`
	Parameters       []parameterRequest `json:"parameters"`
	Format           string             `json:"format"`
	Provider         *uuid.UUID         `json:"provider"`
	Documentation    *documentation     `json:"documentation"`
	SpatialCoverage  *spatial           `json:"spatial_coverage"`
	TemporalCoverage *temporal          `json:"temporal_coverage"`
	ContactPoints    []uuid.UUID        `json:"contact_points"`
}

type parameterRequest struct {
	value domain.Parameter
}

type parameterMetadata struct {
	ID             uuid.UUID            `json:"id"`
	Kind           domain.ParameterKind `json:"kind"`
	Variable       string               `json:"variable"`
	Property       string               `json:"property"`
	Label          string               `json:"label"`
	Required       bool                 `json:"required"`
	MultipleValues bool                 `json:"multiple_values"`
	ReadOnly       bool                 `json:"read_only"`
}

type parameterDraftMetadata struct {
	Kind           domain.ParameterKind `json:"kind"`
	Variable       string               `json:"variable"`
	Property       string               `json:"property"`
	Label          string               `json:"label"`
	Required       bool                 `json:"required"`
	MultipleValues bool                 `json:"multiple_values"`
	ReadOnly       bool                 `json:"read_only"`
}

type stringParameter struct {
	Kind          domain.ParameterKind `json:"kind"`
	Metadata      parameterMetadata    `json:"metadata"`
	Default       *string              `json:"default"`
	AllowedValues []string             `json:"allowed_values"`
	Pattern       string               `json:"pattern"`
}

type stringParameterDraft struct {
	Kind          domain.ParameterKind   `json:"kind"`
	Metadata      parameterDraftMetadata `json:"metadata"`
	Default       *string                `json:"default"`
	AllowedValues []string               `json:"allowed_values"`
	Pattern       string                 `json:"pattern"`
}

func (stringParameter) parameterResponse() {}

type booleanParameter struct {
	Kind     domain.ParameterKind `json:"kind"`
	Metadata parameterMetadata    `json:"metadata"`
	Default  *bool                `json:"default"`
}

type booleanParameterDraft struct {
	Kind     domain.ParameterKind   `json:"kind"`
	Metadata parameterDraftMetadata `json:"metadata"`
	Default  *bool                  `json:"default"`
}

func (booleanParameter) parameterResponse() {}

type integerParameter struct {
	Kind          domain.ParameterKind `json:"kind"`
	Metadata      parameterMetadata    `json:"metadata"`
	Minimum       *int64               `json:"minimum"`
	Maximum       *int64               `json:"maximum"`
	Default       *int64               `json:"default"`
	AllowedValues []int64              `json:"allowed_values"`
}

type integerParameterDraft struct {
	Kind          domain.ParameterKind   `json:"kind"`
	Metadata      parameterDraftMetadata `json:"metadata"`
	Minimum       *int64                 `json:"minimum"`
	Maximum       *int64                 `json:"maximum"`
	Default       *int64                 `json:"default"`
	AllowedValues []int64                `json:"allowed_values"`
}

func (integerParameter) parameterResponse() {}

type floatParameter struct {
	Kind          domain.ParameterKind `json:"kind"`
	Metadata      parameterMetadata    `json:"metadata"`
	Minimum       *float64             `json:"minimum"`
	Maximum       *float64             `json:"maximum"`
	Default       *float64             `json:"default"`
	AllowedValues []float64            `json:"allowed_values"`
}

type floatParameterDraft struct {
	Kind          domain.ParameterKind   `json:"kind"`
	Metadata      parameterDraftMetadata `json:"metadata"`
	Minimum       *float64               `json:"minimum"`
	Maximum       *float64               `json:"maximum"`
	Default       *float64               `json:"default"`
	AllowedValues []float64              `json:"allowed_values"`
}

func (floatParameter) parameterResponse() {}

type dateTimeParameter struct {
	Kind     domain.ParameterKind `json:"kind"`
	Metadata parameterMetadata    `json:"metadata"`
	Minimum  *time.Time           `json:"minimum"`
	Maximum  *time.Time           `json:"maximum"`
	Default  *time.Time           `json:"default"`
	Pattern  string               `json:"pattern"`
}

type dateTimeParameterDraft struct {
	Kind     domain.ParameterKind   `json:"kind"`
	Metadata parameterDraftMetadata `json:"metadata"`
	Minimum  *time.Time             `json:"minimum"`
	Maximum  *time.Time             `json:"maximum"`
	Default  *time.Time             `json:"default"`
	Pattern  string                 `json:"pattern"`
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
	Point   *coordinate  `json:"point,omitempty"`
	Polygon []coordinate `json:"polygon,omitempty"`
}

type temporal struct {
	Start *time.Time `json:"start"`
	End   *time.Time `json:"end"`
}

type identifierDraft struct {
	Type  domain.IdentifierType `json:"type"`
	Value string                `json:"value"`
}

type identifier struct {
	ID    uuid.UUID             `json:"id"`
	Type  domain.IdentifierType `json:"type"`
	Value string                `json:"value"`
}

type record struct {
	ResourceID uuid.UUID        `json:"resource_id"`
	VersionID  uuid.UUID        `json:"version_id"`
	Revision   int32            `json:"revision"`
	Status     domain.Status    `json:"status"`
	CreatedBy  string           `json:"created_by"`
	GroupID    uuid.UUID        `json:"group_id"`
	GroupName  string           `json:"group_name"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
	Metadata   metadataResponse `json:"metadata"`
}

type metadataResponse struct {
	ID                    uuid.UUID           `json:"id"`
	Title                 string              `json:"title"`
	Description           string              `json:"description"`
	Keywords              string              `json:"keywords"`
	AccrualPeriodicity    *string             `json:"accrual_periodicity"`
	QualityAnnotation     *string             `json:"quality_annotation"`
	AccessRight           *string             `json:"access_right"`
	Distributions         []distribution      `json:"distributions"`
	SpatialCoverage       *spatial            `json:"spatial_coverage"`
	TemporalCoverage      *temporal           `json:"temporal_coverage"`
	PersistentIdentifiers []identifier        `json:"persistent_identifiers"`
	ContactPoints         *[]uuid.UUID        `json:"contact_points,omitempty"`
	Providers             []resourceReference `json:"providers"`
	Categories            []resourceReference `json:"categories"`
	Created               *time.Time          `json:"created"`
	Updated               *time.Time          `json:"updated"`
}

type distribution struct {
	ID               uuid.UUID               `json:"id"`
	Title            string                  `json:"title"`
	Description      string                  `json:"description"`
	License          *string                 `json:"license"`
	Type             domain.DistributionType `json:"type"`
	Issued           *time.Time              `json:"issued"`
	Modified         *time.Time              `json:"modified"`
	ByteSize         *int64                  `json:"byte_size"`
	WebService       *webService             `json:"web_service,omitempty"`
	DownloadableFile *downloadableFile       `json:"downloadable_file,omitempty"`
}

type webService struct {
	ID               uuid.UUID           `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	TemplateURL      string              `json:"template_url"`
	Parameters       []parameterResponse `json:"parameters"`
	Format           string              `json:"format"`
	Provider         *resourceReference  `json:"provider"`
	Documentation    *documentation      `json:"documentation"`
	SpatialCoverage  *spatial            `json:"spatial_coverage"`
	TemporalCoverage *temporal           `json:"temporal_coverage"`
	ContactPoints    *[]uuid.UUID        `json:"contact_points,omitempty"`
}

type resourceReference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
