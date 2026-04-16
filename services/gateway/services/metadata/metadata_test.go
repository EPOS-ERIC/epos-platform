package metadatalifecycle

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name          string
		baseURL       string
		timeout       time.Duration
		wantErr       bool
		wantBaseURL   string
		wantReqTimout time.Duration
	}{
		{
			name:    "fails on empty base URL",
			baseURL: "",
			wantErr: true,
		},
		{
			name:    "fails on non absolute base URL",
			baseURL: "metadata-lifecycle:8080",
			wantErr: true,
		},
		{
			name:          "normalizes base URL and sets default timeout",
			baseURL:       "http://metadata-lifecycle:8080/",
			timeout:       0,
			wantBaseURL:   "http://metadata-lifecycle:8080",
			wantReqTimout: defaultRequestTimeout,
		},
		{
			name:          "trims whitespace around base URL",
			baseURL:       "  http://metadata-lifecycle:8080/  ",
			timeout:       0,
			wantBaseURL:   "http://metadata-lifecycle:8080",
			wantReqTimout: defaultRequestTimeout,
		},
		{
			name:          "preserves path and trims trailing slash",
			baseURL:       "http://metadata-lifecycle:8080/api/v2/",
			timeout:       0,
			wantBaseURL:   "http://metadata-lifecycle:8080/api/v2",
			wantReqTimout: defaultRequestTimeout,
		},
		{
			name:          "uses provided timeout",
			baseURL:       "http://metadata-lifecycle:8080",
			timeout:       2 * time.Second,
			wantBaseURL:   "http://metadata-lifecycle:8080",
			wantReqTimout: 2 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, err := NewService(tc.baseURL, tc.timeout)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("new service: %v", err)
			}

			cfg := svc.client.GetConfig()
			if len(cfg.Servers) != 1 {
				t.Fatalf("expected one configured server, got %d", len(cfg.Servers))
			}

			if got := cfg.Servers[0].URL; got != tc.wantBaseURL {
				t.Fatalf("unexpected base URL: got %q want %q", got, tc.wantBaseURL)
			}

			if got := cfg.HTTPClient.Timeout; got != tc.wantReqTimout {
				t.Fatalf("unexpected timeout: got %s want %s", got, tc.wantReqTimout)
			}
		})
	}
}

func TestSearchErrorMapping(t *testing.T) {
	tests := []struct {
		name      string
		newCall   func(t *testing.T) (*Service, context.Context)
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "maps upstream non 2xx to UpstreamError",
			newCall: func(t *testing.T) (*Service, context.Context) {
				t.Helper()

				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadGateway)
					_, _ = w.Write([]byte(`{"error":"upstream broken"}`))
				}))
				t.Cleanup(srv.Close)

				svc, err := NewService(srv.URL, time.Second)
				if err != nil {
					t.Fatalf("new service: %v", err)
				}

				return svc, context.Background()
			},
			assertErr: func(t *testing.T, err error) {
				t.Helper()

				var upstreamErr *UpstreamError
				if !errors.As(err, &upstreamErr) {
					t.Fatalf("expected UpstreamError, got %T", err)
				}

				if upstreamErr.StatusCode != http.StatusBadGateway {
					t.Fatalf("unexpected status code: got %d want %d", upstreamErr.StatusCode, http.StatusBadGateway)
				}

				if !strings.Contains(upstreamErr.Body, "upstream broken") {
					t.Fatalf("unexpected error body: %q", upstreamErr.Body)
				}
			},
		},
		{
			name: "maps request timeout to TimeoutError",
			newCall: func(t *testing.T) (*Service, context.Context) {
				t.Helper()

				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					time.Sleep(100 * time.Millisecond)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"results":[],"total":0,"page":1,"pageSize":20,"totalPages":0}`))
				}))
				t.Cleanup(srv.Close)

				svc, err := NewService(srv.URL, 5*time.Millisecond)
				if err != nil {
					t.Fatalf("new service: %v", err)
				}

				return svc, context.Background()
			},
			assertErr: func(t *testing.T, err error) {
				t.Helper()

				var timeoutErr *TimeoutError
				if !errors.As(err, &timeoutErr) {
					t.Fatalf("expected TimeoutError, got %T", err)
				}
			},
		},
		{
			name: "maps canceled context to TimeoutError",
			newCall: func(t *testing.T) (*Service, context.Context) {
				t.Helper()

				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"results":[],"total":0,"page":1,"pageSize":20,"totalPages":0}`))
				}))
				t.Cleanup(srv.Close)

				svc, err := NewService(srv.URL, time.Second)
				if err != nil {
					t.Fatalf("new service: %v", err)
				}

				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return svc, ctx
			},
			assertErr: func(t *testing.T, err error) {
				t.Helper()

				var timeoutErr *TimeoutError
				if !errors.As(err, &timeoutErr) {
					t.Fatalf("expected TimeoutError, got %T", err)
				}
			},
		},
		{
			name: "maps connection failures to TransportError",
			newCall: func(t *testing.T) (*Service, context.Context) {
				t.Helper()

				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("listen on loopback: %v", err)
				}

				addr := ln.Addr().String()
				if err := ln.Close(); err != nil {
					t.Fatalf("close listener: %v", err)
				}

				svc, err := NewService("http://"+addr, 50*time.Millisecond)
				if err != nil {
					t.Fatalf("new service: %v", err)
				}

				return svc, context.Background()
			},
			assertErr: func(t *testing.T, err error) {
				t.Helper()

				var transportErr *TransportError
				if !errors.As(err, &transportErr) {
					t.Fatalf("expected TransportError, got %T", err)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, ctx := tc.newCall(t)

			_, err := svc.Search(ctx, SearchParams{})
			if err == nil {
				t.Fatal("expected error")
			}

			tc.assertErr(t, err)
		})
	}
}
