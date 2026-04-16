package server

import (
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
)

func TestRunServerReturnsErrorForInvalidMetadataServiceConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Server: config.Server{
			Listen: ":0",
		},
		Services: config.Services{
			MetadataLifecycle: config.Service{URL: "metadata-lifecycle:8080"},
		},
	}

	err := RunServer(cfg)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "build metadata service") {
		t.Fatalf("unexpected error: %v", err)
	}
}
