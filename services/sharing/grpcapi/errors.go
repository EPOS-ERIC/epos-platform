package grpcapi

import (
	"errors"

	"buf.build/go/protovalidate"
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

func ValidationAppError(err error) *apperror.AppError {
	var validationErr *protovalidate.ValidationError
	if !errors.As(err, &validationErr) {
		return apperror.Wrap(apperror.CodeInternal, "", err)
	}
	if len(validationErr.Violations) == 0 || validationErr.Violations[0] == nil || validationErr.Violations[0].Proto == nil {
		return apperror.Wrap(apperror.CodeValidationFailed, "", err)
	}
	violation := validationErr.Violations[0].Proto
	return apperror.Field(protovalidate.FieldPathString(violation.GetField()), violation.GetMessage(), err)
}
