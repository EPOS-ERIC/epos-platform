package domain

import (
	"errors"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
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
