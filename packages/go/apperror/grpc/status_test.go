package grpc

import (
	"errors"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCodeMapping(t *testing.T) {
	tests := []struct {
		code apperror.Code
		want codes.Code
	}{
		{apperror.CodeValidationFailed, codes.InvalidArgument},
		{apperror.CodeUnauthenticated, codes.Unauthenticated},
		{apperror.CodePermissionDenied, codes.PermissionDenied},
		{apperror.CodeNotFound, codes.NotFound},
		{apperror.CodeConflict, codes.FailedPrecondition},
		{apperror.CodeInvalidTransition, codes.FailedPrecondition},
		{apperror.CodeInternal, codes.Internal},
		{apperror.CodeUpstreamTimeout, codes.DeadlineExceeded},
		{apperror.CodeUpstreamUnavailable, codes.Unavailable},
		{apperror.CodeUnsupportedMedia, codes.InvalidArgument},
		{apperror.CodeRequestTooLarge, codes.InvalidArgument},
	}
	for _, test := range tests {
		if got := Code(test.code); got != test.want {
			t.Errorf("Code(%q) = %s, want %s", test.code, got, test.want)
		}
	}
}

func TestStatusRoundTripDoesNotExposeCause(t *testing.T) {
	err := apperror.Field("metadata.title", "Must not be blank.", errors.New("private SQL diagnostic"))
	transport := ToStatus(err)
	got, ok := FromStatus(transport)
	if !ok {
		t.Fatal("FromStatus() did not recognize shared error detail")
	}
	if got.Code != apperror.CodeValidationFailed || got.Target != "metadata.title" {
		t.Fatalf("decoded AppError = %+v", got)
	}
	if status.Convert(transport).Message() == "private SQL diagnostic" {
		t.Fatal("internal cause leaked into gRPC status")
	}
}

func TestFromStatusRejectsUnknownErrorInfo(t *testing.T) {
	transport := status.New(codes.InvalidArgument, "untrusted message").Err()
	if _, ok := FromStatus(transport); ok {
		t.Fatal("FromStatus() accepted status without this contract's ErrorInfo")
	}
}
