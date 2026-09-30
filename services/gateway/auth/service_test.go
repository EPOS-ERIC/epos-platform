package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

func TestAuthenticateClassifiesProviderErrors(t *testing.T) {
	tests := []struct {
		name string
		code int
		want apperror.Code
	}{
		{"unauthorized", http.StatusUnauthorized, apperror.CodeUnauthenticated},
		{"provider failure", http.StatusBadGateway, apperror.CodeUpstreamUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.code)
			}))
			defer server.Close()

			service, err := NewService(slog.New(slog.NewTextHandler(io.Discard, nil)), server.URL)
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.Authenticate(context.Background(), "token")
			if !apperror.IsCode(err, test.want) {
				t.Fatalf("error = %v, want code %q", err, test.want)
			}
		})
	}
}

func TestAuthenticatePreservesProviderDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	service, err := NewService(slog.New(slog.NewTextHandler(io.Discard, nil)), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = service.Authenticate(ctx, "token")
	if !apperror.IsCode(err, apperror.CodeUpstreamTimeout) {
		t.Fatalf("error = %v, want upstream timeout", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error chain does not retain deadline: %v", err)
	}
}
