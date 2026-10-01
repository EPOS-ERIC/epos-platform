package clients

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	apperrorgrpc "github.com/EPOS-ERIC/epos-platform/packages/go/apperror/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWrapErrorPreservesSharedApplicationError(t *testing.T) {
	client := testMetadataLifecycleClient()
	transport := apperrorgrpc.ToStatus(apperror.Field("metadata.title", "Title must not be blank.", errors.New("private cause")))
	err := client.wrapError(context.Background(), "create resource", transport)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %T, want *apperror.AppError", err)
	}
	if appErr.Code != apperror.CodeValidationFailed || appErr.Target != "metadata.title" {
		t.Fatalf("AppError = %+v", appErr)
	}
}

func TestWrapErrorDoesNotInferApplicationCodeFromGRPCStatus(t *testing.T) {
	client := testMetadataLifecycleClient()
	err := client.wrapError(context.Background(), "read resource", status.Error(codes.NotFound, "untrusted server text"))
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %T, want *apperror.AppError", err)
	}
	if appErr.Code != apperror.CodeInternal {
		t.Fatalf("AppError = %+v, want generic internal error", appErr)
	}
}

func TestWrapErrorMapsLocalTransportFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code apperror.Code
	}{
		{name: "unavailable", err: status.Error(codes.Unavailable, "dial failed"), code: apperror.CodeUpstreamUnavailable},
		{name: "deadline", err: status.Error(codes.DeadlineExceeded, "deadline"), code: apperror.CodeUpstreamTimeout},
	}
	client := testMetadataLifecycleClient()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := client.wrapError(context.Background(), "operation", test.err)
			var appErr *apperror.AppError
			if !errors.As(err, &appErr) || appErr.Code != test.code {
				t.Fatalf("error = %v, want AppError code %q", err, test.code)
			}
		})
	}
}

func TestWrapErrorDistinguishesCancellationAndDeadline(t *testing.T) {
	client := testMetadataLifecycleClient()

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err := client.wrapError(canceledCtx, "operation", status.Error(codes.Canceled, "canceled"))
	if !errors.Is(err, context.Canceled) || apperror.IsCode(err, apperror.CodeUpstreamTimeout) {
		t.Fatalf("canceled request error = %v, want caller cancellation without timeout code", err)
	}

	deadlineCtx, cancelDeadline := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelDeadline()
	err = client.wrapError(deadlineCtx, "operation", status.Error(codes.Canceled, "deadline"))
	if !apperror.IsCode(err, apperror.CodeUpstreamTimeout) {
		t.Fatalf("expired request error = %v, want upstream timeout", err)
	}
}

func TestWrapErrorTreatsUnexpectedGRPCCancellationAsInternal(t *testing.T) {
	client := testMetadataLifecycleClient()
	err := client.wrapError(context.Background(), "operation", status.Error(codes.Canceled, "canceled"))
	if !apperror.IsCode(err, apperror.CodeInternal) {
		t.Fatalf("active-context gRPC cancellation error = %v, want internal", err)
	}
}

func testMetadataLifecycleClient() *MetadataLifecycle {
	return &MetadataLifecycle{log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}
