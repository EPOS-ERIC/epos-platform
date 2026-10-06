package grpcapi

import (
	"errors"
	"strings"

	"buf.build/go/protovalidate"
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	apperrorgrpc "github.com/EPOS-ERIC/epos-platform/packages/go/apperror/grpc"
)

// NormalizeError maps a handler error to a client-safe gRPC status.
func NormalizeError(cause error) error {
	if cause == nil {
		return nil
	}

	appErr, ok := errors.AsType[*apperror.AppError](cause)
	if !ok {
		appErr = apperror.Wrap(apperror.CodeInternal, "", cause)
	}

	return apperrorgrpc.ToStatus(appErr)
}

// ValidationAppError converts the first protobuf validation violation to the
// shared client-safe application error contract.
func ValidationAppError(method string, err error) *apperror.AppError {
	var validationErr *protovalidate.ValidationError
	if !errors.As(err, &validationErr) || len(validationErr.Violations) == 0 {
		return nil
	}

	violation := validationErr.Violations[0]
	if violation == nil || violation.Proto == nil {
		return apperror.Wrap(apperror.CodeValidationFailed, "The request is invalid.", err)
	}

	target := protovalidate.FieldPathString(violation.Proto.GetField())
	detail := violation.Proto.GetMessage()
	if target == "" || detail == "" {
		return apperror.Wrap(apperror.CodeValidationFailed, "The request is invalid.", err)
	}

	// update the target for form specific resource requests
	_, name, ok := strings.CutLast(method, "/")
	if !ok {
		name = method
	}
	if name == "CreateResource" || name == "ReplaceResourceDraft" {
		resourceType, field, ok := strings.Cut(strings.TrimPrefix(target, "metadata."), ".")
		switch resourceType {
		case "dataset", "software_application", "software_source_code":
			if ok {
				target = "metadata." + field
			}
		}
	}

	appErr := apperror.Wrap(apperror.CodeValidationFailed, detail, err)
	appErr.Target = target
	return appErr
}
