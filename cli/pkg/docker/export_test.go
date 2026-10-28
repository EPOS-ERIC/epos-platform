package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker/config"
)

func TestExport(t *testing.T) {
	tests := []struct {
		name      string
		setupPath func(t *testing.T) string
		wantErr   bool
	}{
		{
			name: "success - exports docker config filename",
			setupPath: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
		{
			name: "failure - path is a file",
			setupPath: func(t *testing.T) string {
				tmpFile, err := os.CreateTemp(t.TempDir(), "not-a-dir")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				_ = tmpFile.Close()
				return tmpFile.Name()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exportPath := tt.setupPath(t)

			err := Export(ExportOpts{Path: exportPath})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Export() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			dockerConfigPath := filepath.Join(exportPath, "docker-config.yaml")
			if _, err := os.Stat(dockerConfigPath); err != nil {
				t.Fatalf("Export() did not create docker-config.yaml: %v", err)
			}

			legacyPath := filepath.Join(exportPath, "config.yaml")
			if _, err := os.Stat(legacyPath); err == nil {
				t.Fatalf("Export() unexpectedly created legacy config.yaml")
			} else if !os.IsNotExist(err) {
				t.Fatalf("stat legacy config.yaml: %v", err)
			}

			content, err := os.ReadFile(dockerConfigPath)
			if err != nil {
				t.Fatalf("failed to read exported file: %v", err)
			}

			cfg, err := config.LoadConfigFromBytes(content)
			if err != nil {
				t.Fatalf("parse exported config: %v", err)
			}
			cfg.Name = "test-env"
			if err := cfg.Validate(); err != nil {
				t.Fatalf("exported config invalid: %v", err)
			}
		})
	}
}
