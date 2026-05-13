package clients

import (
	"testing"
	"time"
)

const metadataLifecycleBaseURL = "http://metadata-lifecycle:8080"

func TestNewMetadataClient(t *testing.T) {
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
			baseURL:       metadataLifecycleBaseURL + "/",
			timeout:       0,
			wantBaseURL:   metadataLifecycleBaseURL,
			wantReqTimout: defaultMetadataRequestTimeout,
		},
		{
			name:          "trims whitespace around base URL",
			baseURL:       "  " + metadataLifecycleBaseURL + "/  ",
			timeout:       0,
			wantBaseURL:   metadataLifecycleBaseURL,
			wantReqTimout: defaultMetadataRequestTimeout,
		},
		{
			name:          "preserves path and trims trailing slash",
			baseURL:       metadataLifecycleBaseURL + "/api/v2/",
			timeout:       0,
			wantBaseURL:   metadataLifecycleBaseURL + "/api/v2",
			wantReqTimout: defaultMetadataRequestTimeout,
		},
		{
			name:          "uses provided timeout",
			baseURL:       metadataLifecycleBaseURL,
			timeout:       2 * time.Second,
			wantBaseURL:   metadataLifecycleBaseURL,
			wantReqTimout: 2 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewMetadatalifecycle(tc.baseURL, tc.timeout)
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

			if got := cfg.HTTPClient.Timeout; got != tc.wantReqTimout {
				t.Fatalf("unexpected timeout: got %s want %s", got, tc.wantReqTimout)
			}
		})
	}
}
