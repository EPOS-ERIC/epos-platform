package config_test

import "testing"

func TestActiveImages(t *testing.T) {
	cfg := newConfig(t)
	if got := len(cfg.ActiveImages()); got != 10 {
		t.Fatalf("got %d active images, want 10", got)
	}
	cfg.Components.MetadataDatabase.Enabled = false
	if got := len(cfg.ActiveImages()); got != 9 {
		t.Fatalf("got %d active images, want 9", got)
	}
}
