package api

import "time"

type DatasetDraftMetadata struct {
	ID                    string              `json:"id,omitempty"`
	SourceIRI             string              `json:"source_iri"`
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
	ContactPoints         []ResourceReference `json:"contact_points,omitempty"`
	Providers             []ResourceReference `json:"providers,omitempty"`
	Categories            []ResourceReference `json:"categories,omitempty"`
	SourceCreated         *time.Time          `json:"source_created,omitempty"`
	SourceUpdated         *time.Time          `json:"source_updated,omitempty"`
}

type SoftwareApplicationDraftMetadata struct {
	SourceIRI        string              `json:"source_iri"`
	Name             string              `json:"name"`
	Description      *string             `json:"description,omitempty"`
	Keywords         *string             `json:"keywords,omitempty"`
	License          *string             `json:"license,omitempty"`
	Version          *string             `json:"version,omitempty"`
	DownloadURL      *string             `json:"download_url,omitempty"`
	Requirements     *string             `json:"requirements,omitempty"`
	OperatingSystem  *string             `json:"operating_system,omitempty"`
	MainEntityOfPage *string             `json:"main_entity_of_page,omitempty"`
	Creators         []ResourceReference `json:"creators,omitempty"`
	Identifiers      []IdentifierDraft   `json:"identifiers,omitempty"`
	Categories       []ResourceReference `json:"categories,omitempty"`
	ContactPoints    []ResourceReference `json:"contact_points,omitempty"`
}

type SoftwareSourceCodeDraftMetadata struct {
	SourceIRI            string              `json:"source_iri"`
	Name                 string              `json:"name"`
	Description          *string             `json:"description,omitempty"`
	Keywords             *string             `json:"keywords,omitempty"`
	Repository           *string             `json:"repository,omitempty"`
	ProgrammingLanguage  *string             `json:"programming_language,omitempty"`
	License              *string             `json:"license,omitempty"`
	SoftwareRequirements *string             `json:"software_requirements,omitempty"`
	RuntimePlatform      *string             `json:"runtime_platform,omitempty"`
	MainEntityOfPage     *string             `json:"main_entity_of_page,omitempty"`
	Version              *string             `json:"version,omitempty"`
	Creators             []ResourceReference `json:"creators,omitempty"`
	Identifiers          []IdentifierDraft   `json:"identifiers,omitempty"`
	Categories           []ResourceReference `json:"categories,omitempty"`
	ContactPoints        []ResourceReference `json:"contact_points,omitempty"`
}

type ResourceReference struct {
	ID string `json:"id"`
}

type IdentifierDraft struct {
	ID        string `json:"id,omitempty"`
	SourceIRI string `json:"source_iri"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

type DistributionDraft struct {
	ID               string            `json:"id,omitempty"`
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
	ID               string              `json:"id,omitempty"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	TemplateURL      string              `json:"template_url"`
	Parameters       []any               `json:"parameters,omitempty"`
	Format           string              `json:"format"`
	Provider         *ResourceReference  `json:"provider,omitempty"`
	Documentation    *Documentation      `json:"documentation,omitempty"`
	SpatialCoverage  *SpatialCoverage    `json:"spatial_coverage,omitempty"`
	TemporalCoverage *TemporalCoverage   `json:"temporal_coverage,omitempty"`
	ContactPoints    []ResourceReference `json:"contact_points,omitempty"`
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

type CopyResourceRequest struct {
	SourceIRI string `json:"source_iri"`
}

type RejectResourceRequest struct {
	Note string `json:"note"`
}

type ResourceGroupReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResourceVersion struct {
	ResourceID string                  `json:"resource_id"`
	VersionID  string                  `json:"version_id"`
	Revision   int32                   `json:"revision"`
	Status     string                  `json:"status"`
	CreatedBy  string                  `json:"created_by"`
	Group      ResourceGroupReference  `json:"group"`
	Metadata   ResourceVersionMetadata `json:"metadata"`
}

type ResourceVersionMetadata interface {
	resourceVersionMetadata()
}

type DatasetResourceMetadata struct {
	ID          string `json:"id"`
	SourceIRI   string `json:"source_iri"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
	Status      string `json:"status"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
}

func (DatasetResourceMetadata) resourceVersionMetadata() {}

type SoftwareApplicationResourceMetadata struct {
	ID          string `json:"id"`
	SourceIRI   string `json:"source_iri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
}

func (SoftwareApplicationResourceMetadata) resourceVersionMetadata() {}

type SoftwareSourceCodeResourceMetadata struct {
	ID          string `json:"id"`
	SourceIRI   string `json:"source_iri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
}

func (SoftwareSourceCodeResourceMetadata) resourceVersionMetadata() {}

type WriteResult struct {
	ResourceType ResourceType     `json:"resourceType"`
	ID           string           `json:"id"`
	Status       VersioningStatus `json:"status"`
}
