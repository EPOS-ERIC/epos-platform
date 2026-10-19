// Package grpc maps shared application errors to gRPC transport statuses.
package grpc

import (
	"errors"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

const ErrorInfoDomain = "epos-platform"

// NormalizeError maps application errors to client-safe gRPC statuses and
// fails closed for unclassified errors.
func NormalizeError(err error) error {
	if err == nil {
		return nil
	}

	appErr, ok := errors.AsType[*apperror.AppError](err)
	if !ok || appErr == nil {
		appErr = apperror.Wrap(apperror.CodeInternal, "", err)
	}
	return ToStatus(appErr)
}

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
	case apperror.CodeConflict, apperror.CodeInvalidTransition:
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
	safe := apperror.New(err.Code, err.Detail)
	safe.Target = err.Target
	grpcStatus := status.New(Code(safe.Code), safe.Detail)
	withDetails, detailErr := grpcStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: string(safe.Code),
		Domain: ErrorInfoDomain,
		Metadata: map[string]string{
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
		appErr := apperror.Wrap(code, grpcStatus.Message(), err)
		appErr.Target = info.Metadata["target"]
		return appErr, true
	}
	return nil, false
}
