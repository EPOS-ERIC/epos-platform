package http

import (
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
)

func TestRunReturnsErrorForInvalidListenAddress(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Server: config.Server{
			Listen: "127.0.0.1",
		},
	}

	err := Run(cfg, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "listen and serve") {
		t.Fatalf("unexpected error: %v", err)
	}
}
