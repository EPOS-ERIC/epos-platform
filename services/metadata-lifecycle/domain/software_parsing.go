package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// TODO: parse don't validate

// Validate checks software application identity and references.
func (value SoftwareApplicationMetadata) Validate() error {
	if value.ID == uuid.Nil || strings.TrimSpace(value.Name) == "" {
		return fmt.Errorf("ID and name are required: %w", ErrInvalidResource)
	}
	return validateContactPoints(value.ContactPoints, "software application contact point")
}

// Validate checks software source-code identity and references.
func (value SoftwareSourceCodeMetadata) Validate() error {
	if value.ID == uuid.Nil || strings.TrimSpace(value.Name) == "" {
		return fmt.Errorf("ID and name are required: %w", ErrInvalidResource)
	}
	return validateContactPoints(value.ContactPoints, "software source code contact point")
}
