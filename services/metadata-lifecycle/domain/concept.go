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

func NewConcept(id uuid.UUID, name string) (Concept, error) {
	if id == uuid.Nil {
		return Concept{}, apperror.Default(apperror.CodeInternal)
	}

	name, err := NewConceptName(name)
	if err != nil {
		return Concept{}, internalValidationError(err)
	}

	return Concept{ID: id, Name: name}, nil
}

func NewConceptName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalidField("name", "Name must not be blank.", nil)
	}

	return name, nil
}
