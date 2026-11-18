package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ImportBatch struct {
	ResolveSyncedOrganizations bool
	Groups                     []ImportGroup
	Categories                 []CategoryMetadata
	Organizations              []OrganizationMetadata
	ContactPoints              []ContactPoint
	Datasets                   []DatasetMetadata
	SoftwareApplications       []SoftwareApplicationMetadata
	SoftwareSourceCodes        []SoftwareSourceCodeMetadata
	CategoryEdges              []ImportCategoryEdge
	OrganizationEdges          []ImportOrganizationEdge
	Warnings                   []ImportDiagnostic
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

// Normalize validates group metadata before resolving its concept.
func (value *ImportGroup) Normalize() error {
	value.Name = strings.TrimSpace(value.Name)
	value.Description = optionalTrimmedString(value.Description)
	value.Label = optionalTrimmedString(value.Label)
	value.Logo = optionalTrimmedString(value.Logo)
	value.Page = optionalTrimmedString(value.Page)
	value.Color = optionalTrimmedString(value.Color)
	if value.ID == uuid.Nil || value.Name == "" || !value.Concept.Valid() {
		return fmt.Errorf("import group requires ID, name and known concept")
	}
	return nil
}

// Normalize validates an import batch and its nested records.
func (value *ImportBatch) Normalize() error {
	for i := range value.Groups {
		if err := value.Groups[i].Normalize(); err != nil {
			return err
		}
	}
	for i := range value.Categories {
		if err := value.Categories[i].Normalize(); err != nil {
			return err
		}
	}
	for i := range value.Organizations {
		if err := value.Organizations[i].Normalize(); err != nil {
			return err
		}
	}
	for i := range value.ContactPoints {
		if err := value.ContactPoints[i].Normalize(); err != nil {
			return err
		}
	}
	for i := range value.Datasets {
		if err := value.Datasets[i].Normalize(); err != nil {
			return err
		}
	}
	for i := range value.SoftwareApplications {
		if err := value.SoftwareApplications[i].Normalize(); err != nil {
			return err
		}
	}
	for i := range value.SoftwareSourceCodes {
		if err := value.SoftwareSourceCodes[i].Normalize(); err != nil {
			return err
		}
	}
	if err := validateDatasetGroups(*value); err != nil {
		return err
	}
	for _, software := range value.SoftwareApplications {
		if err := validateSoftwareGroups(software.ID, software.Categories); err != nil {
			return err
		}
	}
	for _, software := range value.SoftwareSourceCodes {
		if err := validateSoftwareGroups(software.ID, software.Categories); err != nil {
			return err
		}
	}
	return nil
}

func validateDatasetGroups(batch ImportBatch) error {
	for _, dataset := range batch.Datasets {
		groups := make(map[uuid.UUID]struct{})
		for _, category := range dataset.Categories {
			if category.GroupID == uuid.Nil {
				return fmt.Errorf("dataset %q has category without a group", dataset.ID)
			}

			groups[category.GroupID] = struct{}{}
		}
		if len(groups) != 1 {
			return fmt.Errorf("dataset %q must reference categories from exactly one group", dataset.ID)
		}
	}

	return nil
}

func validateSoftwareGroups(id uuid.UUID, categories []CategoryMetadata) error {
	if len(categories) == 0 {
		return fmt.Errorf("software %q must reference at least one category", id)
	}
	groups := make(map[uuid.UUID]struct{})
	for _, category := range categories {
		if category.GroupID == uuid.Nil {
			return fmt.Errorf("software %q has category without a group", id)
		}

		groups[category.GroupID] = struct{}{}
	}
	if len(groups) != 1 {
		return fmt.Errorf("software %q must reference categories from exactly one group", id)
	}

	return nil
}
