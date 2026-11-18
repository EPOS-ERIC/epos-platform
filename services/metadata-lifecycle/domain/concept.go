package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

type Concept struct {
	ID   uuid.UUID
	Name string
}

type ConceptInput struct {
	Name string
}

func (value *ConceptInput) Normalize() error {
	value.Name = strings.TrimSpace(value.Name)
	if value.Name == "" {
		return invalidField("name", "Name must not be blank.", nil)
	}

	return nil
}

func (value *Concept) Normalize() error {
	if value.ID == uuid.Nil {
		return apperror.Default(apperror.CodeInternal)
	}

	input := ConceptInput{Name: value.Name}
	if err := input.Normalize(); err != nil {
		return internalValidationError(err)
	}

	value.Name = input.Name

	return nil
}
