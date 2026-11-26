package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

// NormalizeActor validates and normalizes an actor identifier.
func NormalizeActor(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperror.Default(apperror.CodeValidationFailed)
	}
	return value, nil
}
