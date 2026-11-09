// Package config loads and validates gateway configuration.
package config

import (
	"fmt"
	"net/mail"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Server        Server        `yaml:"server"`
	Services      Services      `yaml:"services"`
	Notifications Notifications `yaml:"notifications"`
}

type Server struct {
	Listen          string        `yaml:"listen"`
	CORSAllowOrigin string        `yaml:"cors_allow_origin"`
	RequestTimeout  time.Duration `yaml:"request_timeout"`
}

type Service struct {
	URL string `yaml:"url"`
}

type Services struct {
	MetadataLifecycle  Service `yaml:"metadata-lifecycle"`
	SemanticAggregator Service `yaml:"semantic-aggregator"`
	DataRetriever      Service `yaml:"data-retriever"`
	Converter          Service `yaml:"converter"`
	Sharing            Service `yaml:"sharing"`
	Auth               Service `yaml:"auth"`
}

type Notifications struct {
	Mailgun Mailgun `yaml:"mailgun"`
}

type Mailgun struct {
	Enable             bool          `yaml:"enable"`
	Domain             string        `yaml:"domain"`
	From               string        `yaml:"from"`
	APIKeyEnv          string        `yaml:"api_key_env"`
	Timeout            time.Duration `yaml:"timeout"`
	RecipientOverrides []string      `yaml:"recipient_overrides"`
}

func Default() Config {
	return Config{
		Server: Server{
			Listen:          ":5000",
			RequestTimeout:  60 * time.Second,
			CORSAllowOrigin: "",
		},
		Services: Services{
			MetadataLifecycle:  Service{URL: ""},
			SemanticAggregator: Service{URL: ""},
			DataRetriever:      Service{URL: ""},
			Converter:          Service{URL: ""},
			Sharing:            Service{URL: ""},
			Auth:               Service{URL: ""},
		},
		Notifications: Notifications{Mailgun: Mailgun{
			Enable:    false,
			APIKeyEnv: "MAILGUN_API_TOKEN",
			Timeout:   10 * time.Second,
		}},
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

	if c.Services.MetadataLifecycle.URL == "" {
		return fmt.Errorf("services.metadata-lifecycle.url is required")
	}
	if c.Services.SemanticAggregator.URL == "" {
		return fmt.Errorf("services.semantic-aggregator.url is required")
	}
	if c.Services.DataRetriever.URL == "" {
		return fmt.Errorf("services.data-retriever.url is required")
	}
	if c.Services.Converter.URL == "" {
		return fmt.Errorf("services.converter.url is required")
	}

	if c.Services.Sharing.URL == "" {
		return fmt.Errorf("services.sharing.url is required")
	}

	if c.Services.Auth.URL == "" {
		return fmt.Errorf("services.auth.url is required")
	}

	mailgun := c.Notifications.Mailgun
	for i, recipient := range mailgun.RecipientOverrides {
		address, err := mail.ParseAddress(recipient)
		if err != nil || address.Address != recipient {
			return fmt.Errorf("notifications.mailgun.recipient_overrides[%d] must be a valid email address", i)
		}
	}
	if mailgun.Enable {
		if mailgun.Domain == "" {
			return fmt.Errorf("notifications.mailgun.domain is required when Mailgun is enabled")
		}
		if mailgun.From == "" {
			return fmt.Errorf("notifications.mailgun.from is required when Mailgun is enabled")
		}
		if mailgun.APIKeyEnv == "" {
			return fmt.Errorf("notifications.mailgun.api_key_env is required when Mailgun is enabled")
		}
		if mailgun.Timeout <= 0 {
			return fmt.Errorf("notifications.mailgun.timeout must be greater than zero")
		}
	}

	return nil
}
