// Package config loads and validates gateway configuration.
package config

import (
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	Server         Server         `yaml:"server"`
	Services       Services       `yaml:"services"`
	Notifications  Notifications  `yaml:"notifications"`
	MetadataSync   MetadataSync   `yaml:"metadata_sync"`
	Authentication Authentication `yaml:"authentication"`
	IssueTracker   IssueTracker   `yaml:"issue_tracker"`
}

type IssueTracker struct {
	URL           string   `yaml:"url"`
	OpenNew       string   `yaml:"open_new"`
	ParsedURL     *url.URL `yaml:"-"`
	ParsedOpenNew *url.URL `yaml:"-"`
}

type Authentication struct {
	RootURL string `yaml:"root_url"`
}

type MetadataSync struct {
	Enabled     bool          `yaml:"enabled"`
	BaseURL     string        `yaml:"base_url"`
	Username    string        `yaml:"username"`
	PasswordEnv string        `yaml:"password_env"`
	Cron        string        `yaml:"cron"`
	Timeout     time.Duration `yaml:"timeout"`
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
	MetadataLifecycle Service `yaml:"metadata-lifecycle"`
	DataRetriever     Service `yaml:"data-retriever"`
	Converter         Service `yaml:"converter"`
	Sharing           Service `yaml:"sharing"`
	Auth              Service `yaml:"auth"`
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
			MetadataLifecycle: Service{URL: ""},
			DataRetriever:     Service{URL: ""},
			Converter:         Service{URL: ""},
			Sharing:           Service{URL: ""},
			Auth:              Service{URL: ""},
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
	c.IssueTracker.ParsedURL = nil
	c.IssueTracker.ParsedOpenNew = nil
	if c.IssueTracker.URL != "" || c.IssueTracker.OpenNew != "" {
		uri, err := url.Parse(c.IssueTracker.URL)
		if err != nil || !uri.IsAbs() {
			return fmt.Errorf("issue_tracker.url must be an absolute URI")
		}
		c.IssueTracker.ParsedURL = uri
		uri, err = url.Parse(c.IssueTracker.OpenNew)
		if err != nil || !uri.IsAbs() {
			return fmt.Errorf("issue_tracker.open_new must be an absolute URI")
		}
		c.IssueTracker.ParsedOpenNew = uri
	}

	if c.MetadataSync.Enabled {
		sync := c.MetadataSync
		u, err := url.Parse(sync.BaseURL)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("metadata_sync.base_url must be an HTTPS URL without credentials, query or fragment")
		}

		if sync.Username == "" || sync.PasswordEnv == "" {
			return fmt.Errorf("metadata_sync username and password_env are required")
		}
		if sync.Timeout <= 0 {
			return fmt.Errorf("metadata_sync.timeout must be greater than zero")
		}
		schedule, err := cron.ParseStandard(sync.Cron)
		if err != nil {
			return fmt.Errorf("metadata_sync.cron: %w", err)
		}
		if schedule.Next(time.Now().UTC()).IsZero() {
			return fmt.Errorf("metadata_sync.cron has no future execution time")
		}
	}

	if c.Server.Listen == "" {
		return fmt.Errorf("server.listen is required")
	}

	if c.Server.RequestTimeout <= 0 {
		return fmt.Errorf("server.request_timeout must be greater than zero")
	}

	if c.Services.MetadataLifecycle.URL == "" {
		return fmt.Errorf("services.metadata-lifecycle.url is required")
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

	authURL, err := url.Parse(c.Services.Auth.URL)
	if err != nil || authURL.Hostname() == "" || (authURL.Scheme != "http" && authURL.Scheme != "https") || authURL.User != nil || authURL.RawQuery != "" || authURL.Fragment != "" {
		return fmt.Errorf("services.auth.url must be an absolute HTTP(S) userinfo URL without credentials, query or fragment")
	}

	rootURL, err := url.Parse(c.Authentication.RootURL)
	if err != nil || rootURL.Hostname() == "" || (rootURL.Scheme != "http" && rootURL.Scheme != "https") || rootURL.User != nil || rootURL.RawQuery != "" || rootURL.Fragment != "" {
		return fmt.Errorf("authentication.root_url must be an absolute HTTP(S) URL without credentials, query or fragment")
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
