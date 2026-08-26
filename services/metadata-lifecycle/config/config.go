// Package config loads and validates metadata-lifecycle configuration.
package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerPort     string        `envconfig:"PORT" default:"5000"`
	RequestTimeout time.Duration `envconfig:"REQUEST_TIMEOUT" default:"30s"`
	ReadTimeout    time.Duration `envconfig:"READ_TIMEOUT" default:"15s"`
	WriteTimeout   time.Duration `envconfig:"WRITE_TIMEOUT" default:"30s"`
	IdleTimeout    time.Duration `envconfig:"IDLE_TIMEOUT" default:"60s"`

	DBHost     string `envconfig:"DB_HOST" default:"localhost"`
	DBPort     string `envconfig:"DB_PORT" default:"5432"`
	DBName     string `envconfig:"DB_NAME" default:"metadata-lifecycle"`
	DBUser     string `envconfig:"DB_USER" default:"metadata-lifecycle"`
	DBPassword string `envconfig:"DB_PASSWORD" default:"metadata-lifecycle"`
}

// Load reads and validates configuration from environment variables.
func Load() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

// Validate checks required values and timeout settings.
func (c *Config) Validate() error {
	if c.ServerPort == "" {
		return fmt.Errorf("port is required")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be greater than zero")
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be greater than zero")
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be greater than zero")
	}
	if c.IdleTimeout <= 0 {
		return fmt.Errorf("idle_timeout must be greater than zero")
	}
	if c.DBHost == "" {
		return fmt.Errorf("db_host is required")
	}
	if c.DBPort == "" {
		return fmt.Errorf("db_port is required")
	}
	if c.DBName == "" {
		return fmt.Errorf("db_name is required")
	}
	if c.DBUser == "" {
		return fmt.Errorf("db_user is required")
	}
	if c.DBPassword == "" {
		return fmt.Errorf("db_password is required")
	}

	return nil
}
