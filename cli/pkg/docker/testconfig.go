package docker

import (
	"testing"

	"github.com/EPOS-ERIC/epos-platform/cli/pkg/docker/config"
)

func newTestConfig(t *testing.T, name string) *config.EnvConfig {
	t.Helper()
	cfg, err := config.GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Name = name
	return cfg
}
