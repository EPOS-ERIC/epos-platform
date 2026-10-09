package config

import "testing"

func TestValidateMailgunRequiresAPIKeyEnv(t *testing.T) {
	cfg := Default()
	if cfg.Notifications.Mailgun.APIKeyEnv != "MAILGUN_API_TOKEN" {
		t.Fatalf("default Mailgun API key environment variable = %q", cfg.Notifications.Mailgun.APIKeyEnv)
	}
	cfg.Services.MetadataLifecycle.URL = "metadata-lifecycle:5000"
	cfg.Services.Sharing.URL = "sharing:8200"
	cfg.Services.Auth.URL = "http://auth:8080"
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
}
