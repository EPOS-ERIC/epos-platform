package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// NewContactPointInput validates and normalizes editable contact-point fields.
func NewContactPointInput(value ContactPointInput) (ContactPointInput, error) {
	value.Email = strings.TrimSpace(value.Email)
	value.Role = strings.TrimSpace(value.Role)
	if value.Email == "" {
		return ContactPointInput{}, invalidField("email", "Email must not be blank.", nil)
	}
	if value.Role == "" {
		return ContactPointInput{}, invalidField("role", "Role must not be blank.", nil)
	}
	return value, nil
}

// NewContactPoint creates a complete canonical contact point from its identity and input.
func NewContactPoint(id uuid.UUID, value ContactPointInput) (result ContactPoint, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if id == uuid.Nil {
		return ContactPoint{}, apperror.Default(apperror.CodeValidationFailed)
	}
	value, err = NewContactPointInput(value)
	if err != nil {
		return ContactPoint{}, err
	}
	return ContactPoint{
		ID:    id,
		Email: value.Email,
		Role:  value.Role,
	}, nil
}
