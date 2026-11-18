package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// ContactPoint is a complete canonical contact point returned by the domain.
type ContactPoint struct {
	ID    uuid.UUID
	Email string
}

// ContactPointInput contains fields editable through contact-point CRUD.
type ContactPointInput struct {
	Email string
}

// Normalize validates and normalizes editable contact-point fields.
func (value *ContactPointInput) Normalize() error {
	value.Email = strings.TrimSpace(value.Email)
	if value.Email == "" {
		return invalidField("email", "Email must not be blank.", nil)
	}
	return nil
}

// Normalize validates a persisted contact point.
func (value *ContactPoint) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.ID == uuid.Nil {
		return apperror.Default(apperror.CodeValidationFailed)
	}
	value.Email = strings.TrimSpace(value.Email)
	if value.Email == "" {
		return invalidField("email", "Email must not be blank.", nil)
	}
	return nil
}
