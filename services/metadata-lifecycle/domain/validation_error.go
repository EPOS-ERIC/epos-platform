package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

func internalValidationError(err error) error {
	if apperror.IsCode(err, apperror.CodeInternal) {
		return err
	}

	return apperror.Wrap(apperror.CodeInternal, "", err)
}

func prefixOrField(err error, prefix, target, detail string) error {
	if appErr, ok := errors.AsType[*apperror.AppError](err); ok && appErr.Target != "" {
		return apperror.PrefixTarget(err, prefix)
	}

	return apperror.Field(target, detail, err)
}

func validateUUIDListField(values []uuid.UUID, field, label string) error {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for i, value := range values {
		target := fmt.Sprintf("%s[%d]", field, i)
		if value == uuid.Nil {
			return apperror.Field(target, label+" must be a valid UUID.", nil)
		}

		if _, exists := seen[value]; exists {
			return apperror.Field(target, label+" must not be repeated.", nil)
		}

		seen[value] = struct{}{}
	}

	return nil
}

func normalizeOptionalHTTPURL(value *string, target string) error {
	if value == nil {
		return nil
	}

	return normalizeHTTPURL(value, target)
}

func normalizeHTTPURL(value *string, target string) error {
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return apperror.Field(target, "URL must not be blank.", nil)
	}

	parsed, err := url.Parse(normalized)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || strings.ContainsAny(normalized, " \t\r\n") {
		return apperror.Field(target, "Enter an absolute HTTP or HTTPS URL with a host.", err)
	}

	*value = normalized

	return nil
}
