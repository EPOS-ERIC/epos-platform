package domain

import (
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
	"github.com/google/uuid"
)

type ImportBatch struct {
	CategorySchemes      []metadata.CategoryScheme
	Categories           []metadata.Category
	Organizations        []metadata.Organization
	ContactPoints        []metadata.ContactPoint
	Datasets             []metadata.Dataset
	SoftwareApplications []metadata.SoftwareApplication
	SoftwareSourceCodes  []metadata.SoftwareSourceCode
	CategoryEdges        []ImportCategoryEdge
	OrganizationEdges    []ImportOrganizationEdge
	Warnings             []ImportDiagnostic
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
