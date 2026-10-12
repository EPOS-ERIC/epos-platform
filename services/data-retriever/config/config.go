// Package config loads and validates Data Retriever configuration.
package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	GRPCListenAddr string        `envconfig:"GRPC_LISTEN_ADDR" default:":8100"`
	DBHost         string        `envconfig:"DB_HOST" default:"localhost"`
	DBPort         string        `envconfig:"DB_PORT" default:"5432"`
	DBName         string        `envconfig:"DB_NAME" default:"data_retriever"`
	DBUser         string        `envconfig:"DB_USER" default:"data_retriever"`
	DBPassword     string        `envconfig:"DB_PASSWORD" default:"data_retriever"`
	HTTPTimeout    time.Duration `envconfig:"HTTP_TIMEOUT" default:"30s"`
	SchemaMaxBytes int           `envconfig:"SCHEMA_MAX_BYTES" default:"2097152"`
}

// Load reads, normalizes, and validates environment configuration.
func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	if cfg.SchemaMaxBytes < 0 {
		cfg.SchemaMaxBytes = 0
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

// Validate checks required configuration values.
func (c *Config) Validate() error {
	fields := []struct {
		name  string
		value string
	}{
		{"GRPC_LISTEN_ADDR", c.GRPCListenAddr},
		{"DB_HOST", c.DBHost},
		{"DB_PORT", c.DBPort},
		{"DB_NAME", c.DBName},
		{"DB_USER", c.DBUser},
		{"DB_PASSWORD", c.DBPassword},
	}
	for _, field := range fields {
		if field.value == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}
	return nil
}
