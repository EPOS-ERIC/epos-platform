package k8s

import (
	"sort"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/cli/pkg/k8s/config"
)

func TestRenderOpts_Validate(t *testing.T) {
	tests := []struct {
		name         string
		opts         RenderOpts
		wantErr      bool
		errContains  string
		wantName     string
		wantConfig   bool
		checkNameSet bool
	}{
		{
			name:        "defaults config when nil and still requires name",
			opts:        RenderOpts{},
			wantErr:     true,
			errContains: "environment name is required",
			wantConfig:  true,
		},
		{
			name:       "uses name from config when name is empty",
			opts:       RenderOpts{Config: &config.Config{Name: "from-config"}},
			wantErr:    false,
			wantName:   "from-config",
			wantConfig: true,
		},
		{
			name:       "name argument overrides config name",
			opts:       RenderOpts{Name: "from-arg", Config: &config.Config{Name: "from-config"}},
			wantErr:    false,
			wantName:   "from-arg",
			wantConfig: true,
		},
		{
			name:        "returns error when resolved name is empty",
			opts:        RenderOpts{Config: &config.Config{}},
			wantErr:     true,
			errContains: "environment name is required",
		},
		{
			name:        "returns error for invalid name",
			opts:        RenderOpts{Name: "invalid/name", Config: &config.Config{}},
			wantErr:     true,
			errContains: "invalid name for environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.errContains != "" && (err == nil || !strings.Contains(err.Error(), tt.errContains)) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tt.errContains)
			}

			if tt.wantConfig && tt.opts.Config == nil {
				t.Fatal("Validate() did not initialize config")
			}

			if tt.checkNameSet && tt.opts.Name == "" {
				t.Fatal("Validate() did not resolve a name")
			}

			if tt.wantName != "" {
				if tt.opts.Name != tt.wantName {
					t.Fatalf("Validate() opts.Name = %q, want %q", tt.opts.Name, tt.wantName)
				}
				if tt.opts.Config == nil || tt.opts.Config.Name != tt.wantName {
					t.Fatalf("Validate() opts.Config.Name = %q, want %q", tt.opts.Config.Name, tt.wantName)
				}
			}
		})
	}
}

func TestRenderOpts_ConfigRender_OptionalComponents(t *testing.T) {
	tests := []struct {
		name          string
		envName       string
		mutate        func(cfg *config.Config)
		wantGateway   []string
		wantTemplates map[string]string
	}{
		{
			name:    "managed telemetry renders collector",
			envName: "test-otel",
			mutate: func(cfg *config.Config) {
				cfg.Components.OpenTelemetry.Enabled = true
			},
			wantGateway: []string{`value: "http://otel-lgtm:4318"`},
			wantTemplates: map[string]string{
				"templates/otel.yaml": "path: /test-otel/grafana(/|$)(.*)",
			},
		},
		{
			name:    "sharing renders current service",
			envName: "test-sharing",
			mutate: func(cfg *config.Config) {
				cfg.Components.SharingDatabase.Enabled = true
			},
			wantGateway: []string{`url: "sharing:8200"`},
			wantTemplates: map[string]string{
				"templates/sharing.yaml": "name: sharing",
			},
		},
		{
			name:    "backoffice enabled renders UI",
			envName: "test-backoffice",
			mutate: func(cfg *config.Config) {
				cfg.Components.Backoffice.Enabled = true
			},
			wantGateway: []string{`url: "metadata-lifecycle:5000"`},
			wantTemplates: map[string]string{
				"templates/backoffice-ui.yaml": "name: backoffice-ui",
			},
		},
		{
			name:    "external telemetry configures gateway",
			envName: "test-external-otel",
			mutate: func(cfg *config.Config) {
				cfg.Components.OpenTelemetry.Enabled = true
				cfg.Components.OpenTelemetry.Endpoint = "https://otel.example"
			},
			wantGateway: []string{`value: "https://otel.example"`},
			wantTemplates: map[string]string{
				"templates/metadata-lifecycle.yaml": "name: metadata-lifecycle",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.GetDefaultConfig()
			cfg.Name = tt.envName
			tt.mutate(cfg)

			opts := RenderOpts{Config: cfg}
			if err := opts.Validate(); err != nil {
				t.Fatalf("Validate() returned error: %v", err)
			}

			files, err := opts.Config.Render()
			if err != nil {
				t.Fatalf("Config.Render() returned error: %v", err)
			}

			gatewayManifest := mustGetRenderedFileBySuffix(t, files, "templates/gateway.yaml")
			for _, want := range tt.wantGateway {
				if !strings.Contains(gatewayManifest, want) {
					t.Fatalf("gateway manifest missing expected content %q", want)
				}
			}

			for suffix, want := range tt.wantTemplates {
				manifest := mustGetRenderedFileBySuffix(t, files, suffix)
				if !strings.Contains(manifest, want) {
					t.Fatalf("rendered file %q missing expected content %q", suffix, want)
				}
			}
		})
	}
}

func mustGetRenderedFileBySuffix(t *testing.T, files map[string]string, suffix string) string {
	t.Helper()

	for path, content := range files {
		if strings.HasSuffix(path, suffix) {
			return content
		}
	}

	keys := make([]string, 0, len(files))
	for path := range files {
		keys = append(keys, path)
	}
	sort.Strings(keys)

	t.Fatalf("rendered file with suffix %q not found; got files: %v", suffix, keys)

	return ""
}
