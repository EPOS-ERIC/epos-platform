// Package config loads and validates Sharing configuration.
package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerPort string `envconfig:"SERVER_PORT" default:"8200"`

	DBHost     string `envconfig:"DB_HOST" default:"localhost"`
	DBPort     string `envconfig:"DB_PORT" default:"5432"`
	DBName     string `envconfig:"DB_NAME" default:"sharing"`
	DBUser     string `envconfig:"DB_USER" default:"sharing"`
	DBPassword string `envconfig:"DB_PASSWORD" default:"sharing"`
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
