// Package config loads and validates gateway configuration.
package config

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Server    Server    `yaml:"server"`
	Upstream  Upstream  `yaml:"upstream"`
	Discovery Discovery `yaml:"discovery"`
	Services  Services  `yaml:"services"`
}

type Server struct {
	Listen         string        `yaml:"listen"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	IdleTimeout    time.Duration `yaml:"idle_timeout"`
}

type Service struct {
	URL string `yaml:"url"`
}

type Upstream struct {
	Timeout     time.Duration `yaml:"timeout"`
	RealTimeout time.Duration `yaml:"real_timeout"`
	CacheTTL    time.Duration `yaml:"cache_ttl"`
	MaxEntries  int           `yaml:"max_entries"`
}

type Discovery struct {
	FanOutLimit int `yaml:"fan_out_limit"`
}

type Services struct {
	MetadataLifecycle Service `yaml:"metadata-lifecycle"`
}

func Default() Config {
	return Config{
		Server: Server{
			Listen:         ":5000",
			RequestTimeout: 30 * time.Second,
			ReadTimeout:    15 * time.Second,
			WriteTimeout:   30 * time.Second,
			IdleTimeout:    60 * time.Second,
		},
		Upstream: Upstream{
			Timeout:     5 * time.Second,
			RealTimeout: 15 * time.Second,
			CacheTTL:    0 * time.Second,
			MaxEntries:  128,
		},
		Discovery: Discovery{FanOutLimit: 64},
		Services: Services{
			MetadataLifecycle: Service{URL: ""},
		},
	}
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

	cfg := Default()
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

	if c.Server.RequestTimeout <= 0 {
		return fmt.Errorf("server.request_timeout must be greater than zero")
	}

	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("server.read_timeout must be greater than zero")
	}

	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server.write_timeout must be greater than zero")
	}

	if c.Server.IdleTimeout <= 0 {
		return fmt.Errorf("server.idle_timeout must be greater than zero")
	}

	if c.Services.MetadataLifecycle.URL == "" {
		return fmt.Errorf("services.metadata-lifecycle.url is required")
	}

	if c.Upstream.Timeout <= 0 {
		return fmt.Errorf("upstream.timeout must be greater than zero")
	}

	if c.Upstream.RealTimeout <= 0 {
		return fmt.Errorf("upstream.real_timeout must be greater than zero")
	}

	if c.Upstream.CacheTTL < 0 {
		return fmt.Errorf("upstream.cache_ttl must be zero or greater")
	}

	if c.Upstream.MaxEntries <= 0 {
		return fmt.Errorf("upstream.max_entries must be greater than zero")
	}

	if c.Discovery.FanOutLimit <= 0 {
		return fmt.Errorf("discovery.fan_out_limit must be greater than zero")
	}

	return nil
}
