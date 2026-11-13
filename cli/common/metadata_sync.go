package common

import (
	"fmt"
	"net/url"
	"time"

	"github.com/robfig/cron/v3"
)

// MetadataSync configures the gateway's remote metadata source.
type MetadataSync struct {
	Enabled  bool   `yaml:"enabled"`
	BaseURL  string `yaml:"base_url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Cron     string `yaml:"cron"`
	Timeout  string `yaml:"timeout"`
}

func (m MetadataSync) Validate() error {
	if !m.Enabled {
		return nil
	}

	u, err := url.Parse(m.BaseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("components.gateway.metadata_sync.base_url must be an HTTPS URL without credentials, query or fragment")
	}

	if m.Username == "" || m.Password == "" {
		return fmt.Errorf("components.gateway.metadata_sync username and password are required when enabled")
	}

	schedule, err := cron.ParseStandard(m.Cron)
	if err != nil {
		return fmt.Errorf("components.gateway.metadata_sync.cron: %w", err)
	}

	if schedule.Next(time.Now().UTC()).IsZero() {
		return fmt.Errorf("components.gateway.metadata_sync.cron has no future execution time")
	}

	timeout, err := time.ParseDuration(m.Timeout)
	if err != nil || timeout <= 0 {
		return fmt.Errorf("components.gateway.metadata_sync.timeout must be a positive duration")
	}

	return nil
}
