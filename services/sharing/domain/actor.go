package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

type Actor string

func NewActor(value string) (Actor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
	return Actor(value), nil
}

// Caller contains a parsed identity and the Gateway's verified capability.
// Its zero value denotes an anonymous caller without management access.
type Caller struct {
	Actor     Actor
	CanManage bool
}
