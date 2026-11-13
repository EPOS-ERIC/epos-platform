package config

import (
	"testing"
	"time"
)

func TestDefaultRequestTimeout(t *testing.T) {
	if got := Default().Server.RequestTimeout; got != 60*time.Second {
		t.Fatalf("default request timeout = %s, want 60s", got)
	}

	if Default().MetadataSync.Enabled {
		t.Fatal("metadata sync enabled by default")
	}
}

func TestValidateMailgunRequiresAPIKeyEnv(t *testing.T) {
	cfg := Default()
	if cfg.Notifications.Mailgun.APIKeyEnv != "MAILGUN_API_TOKEN" {
		t.Fatalf("default Mailgun API key environment variable = %q", cfg.Notifications.Mailgun.APIKeyEnv)
	}
	cfg.Services.MetadataLifecycle.URL = "metadata-lifecycle:5000"
	cfg.Services.DataRetriever.URL = "data-retriever:8100"
	cfg.Services.Converter.URL = "converter:8200"
	cfg.Services.Sharing.URL = "sharing:8200"
	cfg.Services.Auth.URL = "http://auth:8080"
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() succeeded without public authentication root URL")
	}
	cfg.Authentication.RootURL = "http://localhost:35000"
	cfg.Notifications.Mailgun.Enable = true
	cfg.Notifications.Mailgun.Domain = "mg.example.org"
	cfg.Notifications.Mailgun.From = "EPOS <noreply@example.org>"
	cfg.Notifications.Mailgun.APIKeyEnv = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() succeeded without Mailgun API key environment variable")
	}

	cfg.Notifications.Mailgun.APIKeyEnv = "MAILGUN_API_TOKEN"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	cfg.MetadataSync.Enabled = true
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() succeeded without sync configuration")
	}
	cfg.MetadataSync = MetadataSync{
		Enabled:     true,
		BaseURL:     "https://example.org",
		Username:    "sync",
		PasswordEnv: "SYNC_PASSWORD",
		Cron:        "*/15 * * * *",
		Timeout:     time.Second,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() with sync configured: %v", err)
	}
}
