package config_test

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker/config"
)

func newConfig(t *testing.T) *config.EnvConfig {
	t.Helper()
	cfg, err := config.GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Name = "test-env"
	return cfg
}
