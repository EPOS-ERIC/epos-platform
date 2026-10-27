package grpcapi

import (
	"errors"

	"buf.build/go/protovalidate"
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

func ValidationAppError(err error) error {
	var validationErr *protovalidate.ValidationError
	if !errors.As(err, &validationErr) {
		return apperror.Wrap(apperror.CodeInternal, "", err)
	}
	if len(validationErr.Violations) == 0 || validationErr.Violations[0] == nil || validationErr.Violations[0].Proto == nil {
		return apperror.Wrap(apperror.CodeValidationFailed, "The request is invalid.", err)
	}
	target := protovalidate.FieldPathString(validationErr.Violations[0].Proto.GetField())
	return apperror.Field(target, "Value is invalid.", err)
}
