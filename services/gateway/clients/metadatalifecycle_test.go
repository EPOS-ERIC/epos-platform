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
			wantBaseURL: metadataLifecycleBaseURL,
		},
		{
			name:        "trims whitespace around base URL",
			baseURL:     "  " + metadataLifecycleBaseURL + "/  ",
			wantBaseURL: metadataLifecycleBaseURL,
		},
		{
			name:        "preserves path and trims trailing slash",
			baseURL:     metadataLifecycleBaseURL + "/api/v2/",
			wantBaseURL: metadataLifecycleBaseURL + "/api/v2",
		},
		{
			name:    "accepts upstream configuration",
			baseURL: metadataLifecycleBaseURL,
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
			upstreamCfg := tc.upstreamCfg
			if upstreamCfg == (config.Upstream{}) {
				upstreamCfg = config.Default().Upstream
			}

			client, err := NewMetadatalifecycle(tc.baseURL, upstreamCfg)
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
