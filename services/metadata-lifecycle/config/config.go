// Package config loads and validates metadata-lifecycle configuration.
package config

import (
	"fmt"
	"net/url"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerPort       string `envconfig:"SERVER_PORT" default:"5001"`
	GatewayPublicURL string `envconfig:"GATEWAY_PUBLIC_URL"`

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

// Validate checks required values.
func (c *Config) Validate() error {
	if c.ServerPort == "" {
		return fmt.Errorf("server_port is required")
	}
	publicURL, err := url.Parse(c.GatewayPublicURL)
	if err != nil || publicURL.Host == "" || (publicURL.Scheme != "http" && publicURL.Scheme != "https") || publicURL.RawQuery != "" || publicURL.Fragment != "" {
		return fmt.Errorf("gateway_public_url must be an absolute HTTP(S) URL without a query or fragment")
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
