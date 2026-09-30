package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

func TestSecurityFailureOverridesUpstreamOutcomes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code apperror.Code
		want int
	}{
		{"not found", apperror.Default(apperror.CodeNotFound), apperror.CodeUpstreamUnavailable, http.StatusServiceUnavailable},
		{"conflict", apperror.Default(apperror.CodeConflict), apperror.CodeUpstreamUnavailable, http.StatusServiceUnavailable},
		{"permission denied", apperror.Default(apperror.CodePermissionDenied), apperror.CodeUpstreamUnavailable, http.StatusServiceUnavailable},
		{"validation target", apperror.Field("private_field", "private detail", errors.New("private cause")), apperror.CodeUpstreamUnavailable, http.StatusServiceUnavailable},
		{"deadline", context.DeadlineExceeded, apperror.CodeUpstreamTimeout, http.StatusGatewayTimeout},
		{"upstream deadline", apperror.Wrap(apperror.CodeUpstreamTimeout, "", errors.New("private deadline")), apperror.CodeUpstreamTimeout, http.StatusGatewayTimeout},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := fmt.Errorf("security operation: %w", securityFailure(fmt.Errorf("upstream: %w", test.err)))
			status, problem := problemResponse(context.Background(), err)
			if status != test.want || problem.Type != string(test.code) || problem.Target.Set {
				t.Fatalf("status/problem = %d/%+v, want %d/%s without upstream target", status, problem, test.want, test.code)
			}
			if !problem.Detail.Set || problem.Detail.Value != apperror.Default(test.code).Detail {
				t.Fatalf("detail = %+v, want safe default for %s", problem.Detail, test.code)
			}
			if !errors.Is(err, test.err) {
				t.Fatal("internal cause was lost")
			}
		})
	}
}

func TestSecurityAuthAndBlockedUserKeepTheirStatus(t *testing.T) {
	for _, test := range []struct {
		err  error
		want int
	}{
		{apperror.Default(apperror.CodeUnauthenticated), http.StatusUnauthorized},
		{apperror.Default(apperror.CodePermissionDenied), http.StatusForbidden},
	} {
		status, _ := problemResponse(context.Background(), test.err)
		if status != test.want {
			t.Errorf("status for %v = %d, want %d", test.err, status, test.want)
		}
	}
}
