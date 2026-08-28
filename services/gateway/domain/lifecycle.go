package domain

import (
	"fmt"
	"time"
)

type DraftMetadata interface {
	draftMetadata()
}

type DatasetDraftMetadata struct {
	Title                 string
	Description           *string
	Keywords              *string
	AccrualPeriodicity    *string
	QualityAnnotation     *string
	AccessRight           *string
	Distributions         []DistributionDraft
	SpatialCoverage       *SpatialCoverage
	TemporalCoverage      *TemporalCoverage
	PersistentIdentifiers []IdentifierDraft
	ContactPoints         []string
	Providers             []string
	Categories            []string
	SourceCreated         *time.Time
	SourceUpdated         *time.Time
}

func (DatasetDraftMetadata) draftMetadata() {}

type SoftwareApplicationDraftMetadata struct {
	Name             string
	Description      *string
	Keywords         *string
	License          *string
	Version          *string
	DownloadURL      *string
	Requirements     *string
	OperatingSystem  *string
	MainEntityOfPage *string
	Creators         []string
	Identifiers      []IdentifierDraft
	Categories       []string
	ContactPoints    []string
}

func (SoftwareApplicationDraftMetadata) draftMetadata() {}

type SoftwareSourceCodeDraftMetadata struct {
	Name                 string
	Description          *string
	Keywords             *string
	Repository           *string
	ProgrammingLanguage  *string
	License              *string
	SoftwareRequirements *string
	RuntimePlatform      *string
	MainEntityOfPage     *string
	Version              *string
	Creators             []string
	Identifiers          []IdentifierDraft
	Categories           []string
	ContactPoints        []string
}

func (SoftwareSourceCodeDraftMetadata) draftMetadata() {}

type Identifier struct {
	ID    string
	Type  string
	Value string
}

type IdentifierDraft struct {
	Type  string
	Value string
}

type DistributionDraft struct {
	Title            string
	Description      string
	License          *string
	Type             string
	Issued           *time.Time
	Modified         *time.Time
	ByteSize         *int64
	WebService       *WebServiceDraft
	DownloadableFile *DownloadableFile
}

type WebServiceDraft struct {
	Name             string
	Description      string
	TemplateURL      string
	Parameters       []ParameterDraft
	Format           string
	Provider         *string
	Documentation    *Documentation
	SpatialCoverage  *SpatialCoverage
	TemporalCoverage *TemporalCoverage
	ContactPoints    []string
}

type DownloadableFile struct {
	URL    string
	Format string
}

type Documentation struct {
	Title       *string
	Description *string
	URI         string
}

type SpatialCoverage struct {
	Point   *Coordinate
	Polygon []Coordinate
}

type Coordinate struct {
	Latitude  float64
	Longitude float64
}

type TemporalCoverage struct {
	Start *time.Time
	End   *time.Time
}

// TODO: instead of wrapping the metadata in a metadata field, let's just put them directly in the parent struct. no need for the custom "metadata". or if there is actually a need for it we them maybe should embed it in the parent struct?

type CreateResourceInput struct {
	Metadata DraftMetadata
}

type ReplaceResourceDraftInput struct {
	Revision int32
	Metadata DraftMetadata
}

type RejectResourceInput struct {
	Note string
}

type ResourceVersion struct {
	ResourceID string
	VersionID  string
	Revision   int32
	Status     string
	CreatedBy  string
	GroupID    string
	GroupName  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Metadata   ResourceVersionMetadata
}

type ResourceVersionMetadata interface {
	resourceVersionMetadata()
}

type DatasetResourceMetadata struct {
	ID                    string
	Title                 string
	Description           string
	Keywords              string
	AccrualPeriodicity    *string
	QualityAnnotation     *string
	AccessRight           *string
	Distributions         []Distribution
	SpatialCoverage       *SpatialCoverage
	TemporalCoverage      *TemporalCoverage
	PersistentIdentifiers []Identifier
	ContactPoints         []string
	Providers             []ResourceVersionReference
	Categories            []ResourceVersionReference
	SourceCreated         *time.Time
	SourceUpdated         *time.Time
}

func (DatasetResourceMetadata) resourceVersionMetadata() {}

type SoftwareApplicationResourceMetadata struct {
	ID               string
	Name             string
	Description      string
	MainEntityOfPage *string
	License          *string
	Version          *string
	Keywords         string
	DownloadURL      *string
	Requirements     *string
	OperatingSystem  *string
	Creators         []ResourceVersionReference
	Identifiers      []Identifier
	ContactPoints    []string
	Categories       []ResourceVersionReference
}

func (SoftwareApplicationResourceMetadata) resourceVersionMetadata() {}

type SoftwareSourceCodeResourceMetadata struct {
	ID                   string
	Name                 string
	Description          string
	ProgrammingLanguage  *string
	Keywords             string
	Repository           *string
	License              *string
	SoftwareRequirements *string
	RuntimePlatform      *string
	MainEntityOfPage     *string
	Version              *string
	Creators             []ResourceVersionReference
	Identifiers          []Identifier
	ContactPoints        []string
	Categories           []ResourceVersionReference
}

func (SoftwareSourceCodeResourceMetadata) resourceVersionMetadata() {}

type ResourceVersionReference struct {
	ID   string
	Name string
}

type Distribution interface {
	distribution()
}

type WebServiceDistribution struct {
	ID          string
	Title       string
	Description string
	License     *string
	Type        string
	Issued      *time.Time
	Modified    *time.Time
	ByteSize    *int64
	WebService  WebService
}

func (WebServiceDistribution) distribution() {}

type DownloadableFileDistribution struct {
	ID               string
	Title            string
	Description      string
	License          *string
	Type             string
	Issued           *time.Time
	Modified         *time.Time
	ByteSize         *int64
	DownloadableFile DownloadableFile
}

func (DownloadableFileDistribution) distribution() {}

type WebService struct {
	ID               string
	Name             string
	Description      string
	TemplateURL      string
	Parameters       []Parameter
	Format           string
	Provider         *ResourceVersionReference
	Documentation    *Documentation
	SpatialCoverage  *SpatialCoverage
	TemporalCoverage *TemporalCoverage
	ContactPoints    []string
}

type Parameter interface {
	parameter()
}

type ParameterDraft interface {
	parameterDraft()
}

type ParameterDraftMetadata struct {
	Kind           string
	Variable       string
	Property       string
	Label          string
	Required       bool
	MultipleValues bool
	ReadOnly       bool
}

type StringParameterDraft struct {
	Metadata      ParameterDraftMetadata
	Default       *string
	AllowedValues []string
	Pattern       string
}

func (StringParameterDraft) parameterDraft() {}

type BooleanParameterDraft struct {
	Metadata ParameterDraftMetadata
	Default  *bool
}

func (BooleanParameterDraft) parameterDraft() {}

type IntegerParameterDraft struct {
	Metadata      ParameterDraftMetadata
	Minimum       *int64
	Maximum       *int64
	Default       *int64
	AllowedValues []int64
}

func (IntegerParameterDraft) parameterDraft() {}

type FloatParameterDraft struct {
	Metadata      ParameterDraftMetadata
	Minimum       *float64
	Maximum       *float64
	Default       *float64
	AllowedValues []float64
}

func (FloatParameterDraft) parameterDraft() {}

type DateTimeParameterDraft struct {
	Metadata ParameterDraftMetadata
	Minimum  *time.Time
	Maximum  *time.Time
	Default  *time.Time
	Pattern  string
}

func (DateTimeParameterDraft) parameterDraft() {}

type ParameterMetadata struct {
	ID             string
	Kind           string
	Variable       string
	Property       string
	Label          string
	Required       bool
	MultipleValues bool
	ReadOnly       bool
}

type StringParameter struct {
	Metadata      ParameterMetadata
	Default       *string
	AllowedValues []string
	Pattern       string
}

func (StringParameter) parameter() {}

type BooleanParameter struct {
	Metadata ParameterMetadata
	Default  *bool
}

func (BooleanParameter) parameter() {}

type IntegerParameter struct {
	Metadata      ParameterMetadata
	Minimum       *int64
	Maximum       *int64
	Default       *int64
	AllowedValues []int64
}

func (IntegerParameter) parameter() {}

type FloatParameter struct {
	Metadata      ParameterMetadata
	Minimum       *float64
	Maximum       *float64
	Default       *float64
	AllowedValues []float64
}

func (FloatParameter) parameter() {}

type DateTimeParameter struct {
	Metadata ParameterMetadata
	Minimum  *time.Time
	Maximum  *time.Time
	Default  *time.Time
	Pattern  string
}

func (DateTimeParameter) parameter() {}

// UpstreamError represents a failed request to a downstream service.
// Response bodies stay inside the gateway and are not part of the public API.
type UpstreamError struct {
	StatusCode   int
	WrappedError error
}

func (e *UpstreamError) Error() string {
	if e.WrappedError != nil {
		return fmt.Sprintf("upstream request failed: %s", e.WrappedError.Error())
	}
	return fmt.Sprintf("upstream request failed: %d", e.StatusCode)
}
