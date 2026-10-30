package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	darwinOS    = "darwin"
	windowsOS   = "windows"
	explorerCmd = "explorer"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Ensure TUI config is set
	if cfg.TUI.OpenURLCommand == "" {
		t.Error("OpenUrlCommand should not be empty")
	}
	if cfg.TUI.OpenDirectoryCommand == "" {
		t.Error("OpenDirectoryCommand should not be empty")
	}
	if cfg.TUI.OpenFileCommand == "" {
		t.Error("OpenFileCommand should not be empty")
	}

	// Check OS-specific values
	switch runtime.GOOS {
	case darwinOS:
		if cfg.TUI.OpenURLCommand != "open" {
			t.Errorf("Expected open, got %s", cfg.TUI.OpenURLCommand)
		}
	case windowsOS:
		if cfg.TUI.OpenURLCommand != "cmd /c start" {
			t.Errorf("Expected cmd /c start, got %s", cfg.TUI.OpenURLCommand)
		}
		if cfg.TUI.OpenDirectoryCommand != explorerCmd {
			t.Errorf("Expected %s, got %s", explorerCmd, cfg.TUI.OpenDirectoryCommand)
		}
	default: // linux and others
		if cfg.TUI.OpenURLCommand != "xdg-open" {
			t.Errorf("Expected xdg-open, got %s", cfg.TUI.OpenURLCommand)
		}
	}
}

func TestLoadConfigMalformedAndMissing(t *testing.T) {
	oldPath := configPath
	configPath = t.TempDir()
	t.Cleanup(func() { configPath = oldPath })

	if _, err := LoadConfig(); err != nil {
		t.Fatalf("missing config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configPath, "epos-cli.yaml"), []byte("tui: ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(); !errors.Is(err, ErrMalformedConfig) {
		t.Fatalf("malformed config: got %v", err)
	}
}
