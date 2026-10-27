package apperror

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewUsesSafeFallbackForUnknownCode(t *testing.T) {
	err := New("not_a_shared_code", "")
	if err.Code != CodeInternal {
		t.Fatalf("Code = %q, want %q", err.Code, CodeInternal)
	}
}

func TestDefaultUsesCodeDetail(t *testing.T) {
	err := Default(CodeNotFound)
	if err.Detail != defaultDetails[CodeNotFound] {
		t.Fatalf("Detail = %q, want %q", err.Detail, defaultDetails[CodeNotFound])
	}

	upstreamErr := Default(CodeUpstreamHTTPFailure)
	if upstreamErr.Detail != "Upstream service returned an HTTP failure." {
		t.Fatalf("upstream HTTP failure detail = %q", upstreamErr.Detail)
	}
}

func TestWrapPreservesCauseAndCode(t *testing.T) {
	cause := errors.New("database password leaked")
	err := Wrap(CodeNotFound, "", cause)
	if err.Code != CodeNotFound {
		t.Fatalf("Code = %q, want %q", err.Code, CodeNotFound)
	}
	if !errors.Is(err, cause) {
		t.Fatal("wrapped cause is not discoverable")
	}
	if err.Error() != cause.Error() {
		t.Fatalf("Error() = %q, want internal cause for logs", err.Error())
	}
}

func TestIsCodeFindsWrappedApplicationError(t *testing.T) {
	err := errors.New("context")
	wrapped := fmt.Errorf("operation: %w", Wrap(CodeNotFound, "", err))
	if !IsCode(wrapped, CodeNotFound) {
		t.Fatal("IsCode did not find wrapped application error")
	}
	if IsCode(wrapped, CodeConflict) {
		t.Fatal("IsCode matched wrong application code")
	}
}

func TestPrefixTargetDoesNotNestAppErrors(t *testing.T) {
	cause := errors.New("private diagnostic")
	first := Field("title", "Must not be blank.", cause)
	wrapped := PrefixTarget(first, "metadata")
	var got *AppError
	if !errors.As(wrapped, &got) {
		t.Fatal("prefixed error is not an AppError")
	}
	if got.Target != "metadata.title" {
		t.Fatalf("Target = %q, want metadata.title", got.Target)
	}
	if errors.Unwrap(got) != cause {
		t.Fatal("prefixing nested the original AppError instead of preserving its cause")
	}
}

func TestHTTPStatusMapping(t *testing.T) {
	tests := []struct {
		code Code
		want int
	}{
		{CodeValidationFailed, 400},
		{CodeUnauthenticated, 401},
		{CodePermissionDenied, 403},
		{CodeNotFound, 404},
		{CodeConflict, 409},
		{CodeInvalidTransition, 409},
		{CodeUpstreamTimeout, 504},
		{CodeUpstreamUnavailable, 503},
		{CodeUpstreamHTTPFailure, 503},
		{CodeUnsupportedMedia, 415},
		{CodeRequestTooLarge, 413},
		{Code("unknown"), 500},
	}
	for _, test := range tests {
		if got := HTTPStatus(test.code); got != test.want {
			t.Errorf("HTTPStatus(%q) = %d, want %d", test.code, got, test.want)
		}
	}
}
