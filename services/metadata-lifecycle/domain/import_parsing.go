package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// NewImportGroup validates group metadata before resolving its concept.
func NewImportGroup(value ImportGroup) (ImportGroup, error) {
	value.Name = strings.TrimSpace(value.Name)
	value.Description = optionalTrimmedString(value.Description)
	value.Label = optionalTrimmedString(value.Label)
	value.Logo = optionalTrimmedString(value.Logo)
	value.Page = optionalTrimmedString(value.Page)
	value.Color = optionalTrimmedString(value.Color)
	if value.ID == uuid.Nil || value.Name == "" || !value.Concept.Valid() {
		return ImportGroup{}, fmt.Errorf("import group requires ID, name and known concept")
	}
	return value, nil
}

// NewImportBatch validates and constructs an import batch.
func NewImportBatch(value ImportBatch) (ImportBatch, error) {
	if err := validateDatasetGroups(value); err != nil {
		return ImportBatch{}, err
	}
	for _, software := range value.SoftwareApplications {
		if err := validateSoftwareGroups(software.ID, software.Categories); err != nil {
			return ImportBatch{}, err
		}
	}
	for _, software := range value.SoftwareSourceCodes {
		if err := validateSoftwareGroups(software.ID, software.Categories); err != nil {
			return ImportBatch{}, err
		}
	}
	return value, nil
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
