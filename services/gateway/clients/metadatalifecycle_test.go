package clients

import (
	"testing"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/upstream"
)

const metadataLifecycleBaseURL = "http://metadata-lifecycle:8080"

func TestNewMetadataClient(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		timeout     time.Duration
		upstreamCfg config.Upstream
		wantErr     bool
		wantBaseURL string
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
			name:        "normalizes base URL and sets default timeout",
			baseURL:     metadataLifecycleBaseURL + "/",
			timeout:     0,
			wantBaseURL: metadataLifecycleBaseURL,
		},
		{
			name:        "trims whitespace around base URL",
			baseURL:     "  " + metadataLifecycleBaseURL + "/  ",
			timeout:     0,
			wantBaseURL: metadataLifecycleBaseURL,
		},
		{
			name:        "preserves path and trims trailing slash",
			baseURL:     metadataLifecycleBaseURL + "/api/v2/",
			timeout:     0,
			wantBaseURL: metadataLifecycleBaseURL + "/api/v2",
		},
		{
			name:        "uses provided timeout",
			baseURL:     metadataLifecycleBaseURL,
			timeout:     2 * time.Second,
			wantBaseURL: metadataLifecycleBaseURL,
		},
		{
			name:    "accepts upstream configuration",
			baseURL: metadataLifecycleBaseURL,
			timeout: 2 * time.Second,
			upstreamCfg: config.Upstream{
				Timeout:     3 * time.Second,
				RealTimeout: 4 * time.Second,
				CacheTTL:    10 * time.Second,
				MaxEntries:  64,
			},
			wantBaseURL: metadataLifecycleBaseURL,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewMetadatalifecycle(tc.baseURL, tc.timeout, tc.upstreamCfg)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("new metadata client: %v", err)
			}

			cfg := client.GetConfig()
			if len(cfg.Servers) != 1 {
				t.Fatalf("expected one configured server, got %d", len(cfg.Servers))
			}

			if got := cfg.Servers[0].URL; got != tc.wantBaseURL {
				t.Fatalf("unexpected base URL: got %q want %q", got, tc.wantBaseURL)
			}

			if cfg.HTTPClient == nil {
				t.Fatal("expected HTTP client to be configured")
			}

			if _, ok := cfg.HTTPClient.Transport.(*upstream.Client); !ok {
				t.Fatalf("expected upstream client transport, got %T", cfg.HTTPClient.Transport)
			}
		})
	}
}
