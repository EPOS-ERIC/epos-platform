package config_test

import "testing"

func TestActiveImages(t *testing.T) {
	cfg := newConfig(t)
	if got := len(cfg.ActiveImages()); got != 12 {
		t.Fatalf("got %d active images, want 12", got)
	}
	cfg.Components.MetadataDatabase.Enabled = false
	if got := len(cfg.ActiveImages()); got != 11 {
		t.Fatalf("got %d active images, want 11", got)
	}
}
