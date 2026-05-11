// Package config loads and validates gateway configuration.
package config

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Server   Server   `yaml:"server"`
	Services Services `yaml:"services"`
}

type Server struct {
	Listen         string        `yaml:"listen"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

type Service struct {
	URL string `yaml:"url"`
}

type Services struct {
	MetadataLifecycle Service `yaml:"metadata-lifecycle"`
}

// Load loads configuration from file
func Load(cfgPath string) (*Config, error) {
	// Load from file if it exists
	if _, err := os.Stat(cfgPath); err != nil {
		return nil, fmt.Errorf("load config file %s: %w", cfgPath, err)
	}

	rawCfg, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", cfgPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(rawCfg, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config file %s: %w", cfgPath, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Listen == "" {
		return fmt.Errorf("server.listen is required")
	}

	if c.Services.MetadataLifecycle.URL == "" {
		return fmt.Errorf("services.metadata-lifecycle.url is required")
	}

	return nil
}
