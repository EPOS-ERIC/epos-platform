package grpcapi

import (
	"errors"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	apperrorgrpc "github.com/EPOS-ERIC/epos-platform/packages/go/apperror/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNormalizeErrorEncodesAppError(t *testing.T) {
	cause := errors.New("private database diagnostic")
	err := NormalizeError(apperror.Wrap(
		apperror.CodeInvalidTransition,
		"The resource cannot be submitted from its current state.",
		cause,
	))
	grpcStatus := status.Convert(err)
	if grpcStatus.Code() != codes.FailedPrecondition {
		t.Fatalf("gRPC code = %s, want %s", grpcStatus.Code(), codes.FailedPrecondition)
	}
	decoded, ok := apperrorgrpc.FromStatus(err)
	if !ok || decoded.Code != apperror.CodeInvalidTransition {
		t.Fatalf("decoded error = %+v, ok = %v", decoded, ok)
	}
}

func TestNormalizeErrorFailsClosedForUnknownCause(t *testing.T) {
	err := NormalizeError(errors.New("SQL credentials leaked"))
	grpcStatus := status.Convert(err)
	if grpcStatus.Code() != codes.Internal {
		t.Fatalf("gRPC code = %s, want Internal", grpcStatus.Code())
	}
	if grpcStatus.Message() == "SQL credentials leaked" {
		t.Fatal("internal cause leaked through gRPC")
	}
}

func TestNormalizeErrorEncodesApplicationCode(t *testing.T) {
	err := NormalizeError(apperror.Default(apperror.CodeNotFound))
	decoded, ok := apperrorgrpc.FromStatus(err)
	if !ok || decoded.Code != apperror.CodeNotFound {
		t.Fatalf("decoded error = %+v, ok = %v", decoded, ok)
	}
}

func TestNormalizeErrorPreservesValidationTargetAndDetail(t *testing.T) {
	const target = "metadata.categories[2]"
	const detail = "Category must belong to this group and appear only once."
	err := NormalizeError(apperror.Field(target, detail, nil))
	decoded, ok := apperrorgrpc.FromStatus(err)
	if !ok || decoded.Code != apperror.CodeValidationFailed || decoded.Target != target || decoded.Detail != detail {
		t.Fatalf("decoded error = %+v, ok = %v", decoded, ok)
	}
}
