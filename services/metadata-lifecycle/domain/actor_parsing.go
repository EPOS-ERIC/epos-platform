package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

// NewActor validates and normalizes an actor identifier.
func NewActor(value string) (Actor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
	return Actor(value), nil
}

func (a *Actor) OptionalString() *string {
	if a == nil {
		return nil
	}
	return new(string(*a))
}
