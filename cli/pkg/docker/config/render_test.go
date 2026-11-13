package config_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDockerEnvConfig_Render(t *testing.T) {
	cfg := newConfig(t)
	files, err := cfg.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files[".env"], "GATEWAY_AUTH_URL='http://aai-service:8080/oauth2/userinfo'") {
		t.Error("managed AAI UserInfo URL missing")
	}
	if !strings.Contains(files["docker-compose.yaml"], `root_url: "http://localhost:35000"`) {
		t.Error("public AAI root URL missing from Gateway configuration")
	}
	if strings.Contains(files["docker-compose.yaml"], "      metadata_sync:") {
		t.Error("disabled metadata sync rendered in gateway config")
	}
	for _, name := range []string{"metadata-lifecycle:", "data-retriever:", "converter:", "converter-db:", "converter-plugins:/data/plugins", `url: "converter:8200"`, "sharing:", "metadata-db:", "gateway_config:"} {
		if !strings.Contains(files["docker-compose.yaml"], name) {
			t.Errorf("missing %s", name)
		}
	}
	if count := strings.Count(files["docker-compose.yaml"], "platform: linux/amd64"); count != 12 {
		t.Fatalf("got %d linux/amd64 services, want 12", count)
	}
	cfg.Components.Gateway.MetadataSync.Enabled = true
	cfg.Components.Gateway.MetadataSync.Username = "sync-user"
	cfg.Components.Gateway.MetadataSync.Password = "test-password"
	files, err = cfg.Render()
	if err != nil {
		t.Fatal(err)
	}
	var compose map[string]any
	if err := yaml.Unmarshal([]byte(files["docker-compose.yaml"]), &compose); err != nil {
		t.Fatalf("invalid rendered compose YAML: %v", err)
	}
	for _, want := range []string{`password_env: "METADATA_SYNC_PASSWORD"`, `cron: "*/15 * * * *"`, `username: "sync-user"`} {
		if !strings.Contains(files["docker-compose.yaml"], want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(files["docker-compose.yaml"], "test-password") || !strings.Contains(files[".env"], "METADATA_SYNC_PASSWORD='test-password'") {
		t.Error("metadata sync password not confined to environment file")
	}
	cfg.Components.MetadataDatabase.Enabled = false
	files, err = cfg.Render()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(files["docker-compose.yaml"], "\n  metadata-db:\n    image:") {
		t.Fatal("managed metadata database rendered when disabled")
	}
	cfg.Components.ConverterDatabase.Enabled = false
	files, err = cfg.Render()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(files["docker-compose.yaml"], "\n  converter-db:\n    image:") {
		t.Fatal("managed converter database rendered when disabled")
	}
}
