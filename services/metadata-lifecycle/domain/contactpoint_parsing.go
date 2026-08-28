package domain

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrInvalidContactPoint = errors.New("invalid contact point")

// NewContactPointInput validates and normalizes editable contact-point fields.
func NewContactPointInput(value ContactPointInput) (ContactPointInput, error) {
	value.Email = strings.TrimSpace(value.Email)
	value.Role = strings.TrimSpace(value.Role)
	if value.Email == "" || value.Role == "" {
		return ContactPointInput{}, ErrInvalidContactPoint
	}
	return value, nil
}

// NewContactPoint creates a complete canonical contact point from its identity and input.
func NewContactPoint(id uuid.UUID, value ContactPointInput) (ContactPoint, error) {
	if id == uuid.Nil {
		return ContactPoint{}, ErrInvalidContactPoint
	}
	value, err := NewContactPointInput(value)
	if err != nil {
		return ContactPoint{}, err
	}
	return ContactPoint{
		ID:    id,
		Email: value.Email,
		Role:  value.Role,
	}, nil
}
