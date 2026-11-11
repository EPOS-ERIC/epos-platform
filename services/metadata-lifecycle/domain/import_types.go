package domain

import "github.com/google/uuid"

type ImportBatch struct {
	Groups               []ImportGroup
	Categories           []CategoryMetadata
	Organizations        []OrganizationMetadata
	ContactPoints        []ContactPoint
	Datasets             []DatasetMetadata
	SoftwareApplications []SoftwareApplicationMetadata
	SoftwareSourceCodes  []SoftwareSourceCodeMetadata
	CategoryEdges        []ImportCategoryEdge
	OrganizationEdges    []ImportOrganizationEdge
	Warnings             []ImportDiagnostic
}

// ImportGroup contains Turtle group data before its concept name is resolved.
type ImportGroup struct {
	ID          uuid.UUID
	Name        string
	Description *string
	Label       *string
	Logo        *string
	Page        *string
	Color       *string
	Position    int32
	Concept     ImportedConcept
}

type ImportCategoryEdge struct {
	ParentID uuid.UUID
	ChildID  uuid.UUID
}

type ImportOrganizationEdge struct {
	ParentID uuid.UUID
	ChildID  uuid.UUID
}

type ImportSeverity string

const (
	ImportSeverityWarning ImportSeverity = "warning"
	ImportSeverityError   ImportSeverity = "error"
)

type ImportDiagnostic struct {
	Severity  ImportSeverity
	Subject   string
	Predicate string
	Message   string
}

type ImportResult struct {
	Groups               int
	Categories           int
	Organizations        int
	ContactPoints        int
	Datasets             int
	SoftwareApplications int
	SoftwareSourceCodes  int
	Warnings             []ImportDiagnostic
}
