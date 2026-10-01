package domain

import (
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

func invalidField(target, detail string, cause error) error {
	return apperror.Field(target, detail, cause)
}

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
	return invalidField(target, detail, err)
}

func validateUUIDListField(values []uuid.UUID, field, label string) error {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for i, value := range values {
		target := fmt.Sprintf("%s[%d]", field, i)
		if value == uuid.Nil {
			return invalidField(target, label+" must be a valid UUID.", nil)
		}
		if _, exists := seen[value]; exists {
			return invalidField(target, label+" must not be repeated.", nil)
		}
		seen[value] = struct{}{}
	}
	return nil
}
