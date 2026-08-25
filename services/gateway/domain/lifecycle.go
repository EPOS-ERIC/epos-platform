package domain

import "time"

type DraftMetadata interface {
	draftMetadata()
}

type DatasetDraftMetadata struct {
	ID                    string
	SourceIRI             string
	Title                 string
	Description           *string
	Keywords              *string
	AccrualPeriodicity    *string
	QualityAnnotation     *string
	AccessRight           *string
	Distributions         []DistributionDraft
	SpatialCoverage       *SpatialCoverage
	TemporalCoverage      *TemporalCoverage
	PersistentIdentifiers []Identifier
	ContactPoints         []ResourceReference
	Providers             []ResourceReference
	Categories            []ResourceReference
	SourceCreated         *time.Time
	SourceUpdated         *time.Time
}

func (DatasetDraftMetadata) draftMetadata() {}

type SoftwareApplicationDraftMetadata struct {
	SourceIRI        string
	Name             string
	Description      *string
	Keywords         *string
	License          *string
	Version          *string
	DownloadURL      *string
	Requirements     *string
	OperatingSystem  *string
	MainEntityOfPage *string
	Creators         []ResourceReference
	Identifiers      []Identifier
	Categories       []ResourceReference
	ContactPoints    []ResourceReference
}

func (SoftwareApplicationDraftMetadata) draftMetadata() {}

type SoftwareSourceCodeDraftMetadata struct {
	SourceIRI            string
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
	Creators             []ResourceReference
	Identifiers          []Identifier
	Categories           []ResourceReference
	ContactPoints        []ResourceReference
}

func (SoftwareSourceCodeDraftMetadata) draftMetadata() {}

type ResourceReference struct {
	ID string
}

type Identifier struct {
	ID        string
	SourceIRI string
	Type      string
	Value     string
}

type DistributionDraft struct {
	ID               string
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
	ID               string
	Name             string
	Description      string
	TemplateURL      string
	Parameters       []any
	Format           string
	Provider         *ResourceReference
	Documentation    *Documentation
	SpatialCoverage  *SpatialCoverage
	TemporalCoverage *TemporalCoverage
	ContactPoints    []ResourceReference
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

type CopyResourceInput struct {
	SourceIRI string
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
	Group      ResourceGroupReference
	Metadata   ResourceVersionMetadata
}

type ResourceGroupReference struct {
	ID   string
	Name string
}

type ResourceVersionMetadata interface {
	resourceVersionMetadata()
}

type DatasetResourceMetadata struct {
	ID          string
	SourceIRI   string
	Title       string
	Description string
	Keywords    string
	Status      string
	Created     string
	Updated     string
}

func (DatasetResourceMetadata) resourceVersionMetadata() {}

type SoftwareApplicationResourceMetadata struct {
	ID          string
	SourceIRI   string
	Name        string
	Description string
	Keywords    string
}

func (SoftwareApplicationResourceMetadata) resourceVersionMetadata() {}

type SoftwareSourceCodeResourceMetadata struct {
	ID          string
	SourceIRI   string
	Name        string
	Description string
	Keywords    string
}

func (SoftwareSourceCodeResourceMetadata) resourceVersionMetadata() {}

// UpstreamError represents a failed request to a downstream service.
// Response bodies stay inside the gateway and are not part of the public API.
type UpstreamError struct {
	StatusCode int
}

func (e *UpstreamError) Error() string {
	return "upstream request failed"
}
