package config_test

import (
	"strings"
	"testing"
)

func TestDockerEnvConfig_Render(t *testing.T) {
	cfg := newConfig(t)
	files, err := cfg.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"metadata-lifecycle:", "data-retriever:", "converter:", "converter-db:", "converter-plugins:/data/plugins", `url: "converter:8200"`, "sharing:", "metadata-db:", "gateway_config:"} {
		if !strings.Contains(files["docker-compose.yaml"], name) {
			t.Errorf("missing %s", name)
		}
	}
	if count := strings.Count(files["docker-compose.yaml"], "platform: linux/amd64"); count != 12 {
		t.Fatalf("got %d linux/amd64 services, want 12", count)
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
