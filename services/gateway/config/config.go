package config

import (
	"fmt"
	"os"
	"path"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Server   Server             `yaml:"server"`
	Auth     Auth               `yaml:"auth"`
	Services map[string]Service `yaml:"services"`
}

type Server struct {
	Listen         string        `yaml:"listen"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
	BasePath       string        `yaml:"base_path"`
}

type Auth struct {
	Enabled bool   `yaml:"enabled"`
	AuthURL string `yaml:"auth_url"`
}

type Service struct {
	URL string `yaml:"url"`
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

	if c.Auth.Enabled && c.Auth.AuthURL == "" {
		return fmt.Errorf("auth.auth_url is required when auth.enabled is true")
	}

	if len(c.Services) == 0 {
		return fmt.Errorf("at least one service is required")
	}

	for name, svc := range c.Services {
		if name == "" {
			return fmt.Errorf("services contains an empty service name")
		}
		if svc.URL == "" {
			return fmt.Errorf("services.%s.url is required", name)
		}
	}

	c.Server.BasePath = path.Clean(c.Server.BasePath)

	return nil
}
