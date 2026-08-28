package domain

import (
	"errors"
	"strings"
)

var ErrInvalidActor = errors.New("invalid actor")

// NewActor validates and normalizes an actor identifier.
func NewActor(value string) (Actor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrInvalidActor
	}
	return Actor(value), nil
}

func (a *Actor) OptionalString() *string {
	if a == nil {
		return nil
	}
	return new(string(*a))
}
