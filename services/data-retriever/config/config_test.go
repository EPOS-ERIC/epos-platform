package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	clearEnvironment(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GRPCListenAddr != ":8100" || cfg.DBHost != "localhost" || cfg.DBPort != "5432" || cfg.DBName != "data_retriever" || cfg.DBUser != "data_retriever" || cfg.DBPassword != "data_retriever" {
		t.Fatalf("string defaults = %+v", cfg)
	}
	if cfg.HTTPTimeout != 30*time.Second || cfg.SchemaMaxBytes != 2<<20 {
		t.Fatalf("numeric defaults = timeout %s, schema budget %d", cfg.HTTPTimeout, cfg.SchemaMaxBytes)
	}
}

func TestLoadOverridesAndClampsSchemaBudget(t *testing.T) {
	clearEnvironment(t)
	t.Setenv("HTTP_TIMEOUT", "5s")
	t.Setenv("SCHEMA_MAX_BYTES", "-1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPTimeout != 5*time.Second || cfg.SchemaMaxBytes != 0 {
		t.Fatalf("overrides = timeout %s, schema budget %d", cfg.HTTPTimeout, cfg.SchemaMaxBytes)
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{name: "invalid timeout", key: "HTTP_TIMEOUT", value: "soon", want: "HTTP_TIMEOUT"},
		{name: "invalid schema budget", key: "SCHEMA_MAX_BYTES", value: "large", want: "SCHEMA_MAX_BYTES"},
		{name: "empty database host", key: "DB_HOST", value: "", want: "DB_HOST is required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearEnvironment(t)
			t.Setenv(test.key, test.value)

			if _, err := Load(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Load() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func clearEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"GRPC_LISTEN_ADDR", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD", "HTTP_TIMEOUT", "SCHEMA_MAX_BYTES",
	} {
		value, exists := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			var err error
			if exists {
				err = os.Setenv(key, value)
			} else {
				err = os.Unsetenv(key)
			}
			if err != nil {
				t.Errorf("restore %s: %v", key, err)
			}
		})
	}
}
