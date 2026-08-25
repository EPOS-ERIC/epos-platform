package domain

type DraftMetadata interface {
	draftMetadata()
}

type DatasetDraftMetadata struct {
	SourceIRI   string
	Title       string
	Description *string
	Keywords    *string
}

func (DatasetDraftMetadata) draftMetadata() {}

type SoftwareApplicationDraftMetadata struct {
	SourceIRI        string
	Name             string
	Description      *string
	Keywords         *string
	DownloadURL      *string
	MainEntityOfPage *string
}

func (SoftwareApplicationDraftMetadata) draftMetadata() {}

type SoftwareSourceCodeDraftMetadata struct {
	SourceIRI           string
	Name                string
	Description         *string
	Keywords            *string
	Repository          *string
	ProgrammingLanguage *string
}

func (SoftwareSourceCodeDraftMetadata) draftMetadata() {}

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
