package domain

import (
	"fmt"

	"github.com/google/uuid"
)

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
		schemes := make(map[uuid.UUID]struct{})
		for _, category := range dataset.Categories {
			schemes[category.Scheme.ID] = struct{}{}
		}
		if len(schemes) != 1 {
			return fmt.Errorf("dataset %q must reference categories from exactly one scheme", dataset.ID)
		}
	}
	return nil
}

func validateSoftwareGroups(id uuid.UUID, categories []CategoryMetadata) error {
	if len(categories) == 0 {
		return fmt.Errorf("software %q must reference at least one category", id)
	}
	schemes := make(map[uuid.UUID]struct{})
	for _, category := range categories {
		schemes[category.Scheme.ID] = struct{}{}
	}
	if len(schemes) != 1 {
		return fmt.Errorf("software %q must reference categories from exactly one scheme", id)
	}
	return nil
}
