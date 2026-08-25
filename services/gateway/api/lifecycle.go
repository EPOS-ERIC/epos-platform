package api

// TODO: Define complete typed resource objects for create and replace requests,
// including all Backoffice dataset and software metadata fields.

type DatasetDraftMetadata struct {
	SourceIRI   string  `json:"source_iri"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Keywords    *string `json:"keywords,omitempty"`
}

func (DatasetDraftMetadata) draftMetadata() {}

type SoftwareApplicationDraftMetadata struct {
	SourceIRI        string  `json:"source_iri"`
	Name             string  `json:"name"`
	Description      *string `json:"description,omitempty"`
	Keywords         *string `json:"keywords,omitempty"`
	DownloadURL      *string `json:"download_url,omitempty"`
	MainEntityOfPage *string `json:"main_entity_of_page,omitempty"`
}

func (SoftwareApplicationDraftMetadata) draftMetadata() {}

type SoftwareSourceCodeDraftMetadata struct {
	SourceIRI           string  `json:"source_iri"`
	Name                string  `json:"name"`
	Description         *string `json:"description,omitempty"`
	Keywords            *string `json:"keywords,omitempty"`
	Repository          *string `json:"repository,omitempty"`
	ProgrammingLanguage *string `json:"programming_language,omitempty"`
}

func (SoftwareSourceCodeDraftMetadata) draftMetadata() {}

type ReplaceDatasetDraftRequest struct {
	Revision int32                `json:"revision"`
	Metadata DatasetDraftMetadata `json:"metadata"`
}

type ReplaceSoftwareApplicationDraftRequest struct {
	Revision int32                            `json:"revision"`
	Metadata SoftwareApplicationDraftMetadata `json:"metadata"`
}

type ReplaceSoftwareSourceCodeDraftRequest struct {
	Revision int32                          `json:"revision"`
	Metadata SoftwareSourceCodeDraftMetadata `json:"metadata"`
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

// TODO: can we delete this?
type WriteResult struct {
	ResourceType ResourceType     `json:"resourceType"`
	ID           string           `json:"id"`
	Status       VersioningStatus `json:"status"`
}
