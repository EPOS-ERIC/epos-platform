package config_test

import (
	"strings"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	cfg := newConfig(t)
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, password := range []string{
		cfg.Components.MetadataDatabase.Password,
		cfg.Components.DataRetrieverDatabase.Password,
		cfg.Components.SharingDatabase.Password,
		cfg.Components.AAIService.Password,
		cfg.Components.OpenTelemetry.AdminPassword,
	} {
		if len(password) != 8 || strings.Trim(password, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789") != "" {
			t.Fatal("generated password must contain eight alphanumeric characters")
		}
	}
	other := newConfig(t)
	if cfg.Components.MetadataDatabase.Password == other.Components.MetadataDatabase.Password {
		t.Fatal("credentials reused across new configs")
	}
	for _, tc := range []struct {
		name   string
		change func()
	}{
		{"missing name", func() { cfg.Name = "" }},
		{"missing domain", func() { cfg.Domain = "" }},
		{"invalid protocol", func() { cfg.Protocol = "ftp" }},
		{"invalid domain", func() { cfg.Domain = "host/path" }},
		{"missing database password", func() { cfg.Components.MetadataDatabase.Password = "" }},
		{"missing retriever password", func() { cfg.Components.DataRetrieverDatabase.Password = "" }},
		{"missing sharing password", func() { cfg.Components.SharingDatabase.Password = "" }},
		{"missing AAI password", func() { cfg.Components.AAIService.Password = "" }},
		{"missing gateway image", func() { cfg.Images.GatewayImage = "" }},
		{"invalid UI port", func() { cfg.Components.PlatformUI.Port = 65536 }},
		{"duplicate ports", func() { cfg.Components.Gateway.Port = cfg.Components.PlatformUI.Port }},
		{"invalid gateway path", func() { cfg.Components.Gateway.BaseURL = "/api/v1" }},
		{"invalid backoffice path", func() { cfg.Components.Backoffice.GUI.BaseURL = "/backoffice" }},
		{"invalid external database", func() { cfg.Components.SharingDatabase.Enabled = false; cfg.Components.SharingDatabase.Host = "" }},
		{"invalid AAI URL", func() {
			cfg.Components.AAIService.Enabled = false
			cfg.Components.Gateway.AAI.ServiceEndpoint = "https://auth.example.org"
			cfg.Components.Gateway.AAI.GatewayEndpoint = "invalid"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg = newConfig(t)
			tc.change()
			if err := cfg.Validate(); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}
