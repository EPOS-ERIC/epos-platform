// Package grpc maps shared application errors to gRPC transport statuses.
package grpc

import (
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

const ErrorInfoDomain = "epos-platform"

// Code returns the gRPC status code used for an application code.
func Code(code apperror.Code) codes.Code {
	switch code {
	case apperror.CodeValidationFailed, apperror.CodeUnsupportedMedia, apperror.CodeRequestTooLarge:
		return codes.InvalidArgument
	case apperror.CodeUnauthenticated:
		return codes.Unauthenticated
	case apperror.CodePermissionDenied:
		return codes.PermissionDenied
	case apperror.CodeNotFound:
		return codes.NotFound
	case apperror.CodeConflict:
		return codes.AlreadyExists
	case apperror.CodeInvalidTransition:
		return codes.FailedPrecondition
	case apperror.CodeUpstreamTimeout:
		return codes.DeadlineExceeded
	case apperror.CodeUpstreamUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

// ToStatus encodes an AppError as a sanitized gRPC status with ErrorInfo.
func ToStatus(err *apperror.AppError) error {
	code := err.Code
	if !apperror.Known(code) {
		code = apperror.CodeInternal
	}
	safe := apperror.New(code, err.Detail)
	safe.Target = err.Target
	grpcStatus := status.New(Code(code), safe.Detail)
	withDetails, detailErr := grpcStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: string(code),
		Domain: ErrorInfoDomain,
		Metadata: map[string]string{
			"detail": safe.Detail,
			"target": safe.Target,
		},
	})
	if detailErr != nil {
		return grpcStatus.Err()
	}
	return withDetails.Err()
}

// FromStatus decodes only this contract's ErrorInfo; all other errors fail closed.
func FromStatus(err error) (*apperror.AppError, bool) {
	grpcStatus, ok := status.FromError(err)
	if !ok {
		return nil, false
	}
	for _, detail := range grpcStatus.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok || info.Domain != ErrorInfoDomain {
			continue
		}
		code := apperror.Code(info.Reason)
		if !apperror.Known(code) {
			return nil, false
		}
		appErr := apperror.Wrap(code, info.Metadata["detail"], err)
		appErr.Target = info.Metadata["target"]
		return appErr, true
	}
	return nil, false
}
