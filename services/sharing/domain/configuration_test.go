package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func validConfiguration(t *testing.T) json.RawMessage {
	t.Helper()
	return json.RawMessage(`{"configurables":[{"type":"mapZoom","value":"11"}]}`)
}

func publicInput(t *testing.T) ConfigurationInput {
	t.Helper()
	return ConfigurationInput{Configuration: validConfiguration(t), IsPublic: true}
}

func TestNewConfigurationInputAcceptsFullPayload(t *testing.T) {
	title := "  Vesuvius & Campi Flegrei volcanoes  "
	description := "Compare geodynamic activity."
	input, err := NewConfigurationInput(ConfigurationInput{
		Configuration:  validConfiguration(t),
		Title:          &title,
		Description:    &description,
		ListOfServices: []string{" GNSS Stations ", "InSAR"},
		IsPublic:       true,
	})
	if err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
	if *input.Title != "Vesuvius & Campi Flegrei volcanoes" {
		t.Errorf("title was not trimmed: %q", *input.Title)
	}
	if len(input.ListOfServices) != 2 || input.ListOfServices[0] != "GNSS Stations" {
		t.Errorf("services were not normalized: %v", input.ListOfServices)
	}
}

func TestNewConfigurationInputRequiresConfiguration(t *testing.T) {
	for _, raw := range []string{"", "   ", "null", "{invalid"} {
		_, err := NewConfigurationInput(ConfigurationInput{Configuration: json.RawMessage(raw), IsPublic: true})
		if err == nil {
			t.Errorf("expected error for configuration %q", raw)
		}
	}
}

func TestNewConfigurationInputAcceptsAnyJSONShape(t *testing.T) {
	// The payload is schemaless: objects, arrays and scalars must all pass so
	// future frontend payloads keep working without a contract change.
	for _, raw := range []string{`[1,2,3]`, `"just a string"`, `42`, `{"a":{"b":[true]}}`} {
		input, err := NewConfigurationInput(ConfigurationInput{Configuration: json.RawMessage(raw), IsPublic: true})
		if err != nil {
			t.Errorf("expected %s to be accepted, got %v", raw, err)
		}
		if !json.Valid(input.Configuration) {
			t.Errorf("expected canonical JSON for %s", raw)
		}
	}
}

func TestNewConfigurationInputBlankMetadataClears(t *testing.T) {
	blank := "   "
	input, err := NewConfigurationInput(ConfigurationInput{Configuration: validConfiguration(t), Title: &blank, IsPublic: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.Title != nil {
		t.Errorf("expected blank title to clear, got %q", *input.Title)
	}
}

func TestNewConfigurationInputOfficialRequiresPublic(t *testing.T) {
	_, err := NewConfigurationInput(ConfigurationInput{Configuration: validConfiguration(t), IsOfficial: true, IsPublic: false})
	if err == nil {
		t.Error("expected error for official non-public configuration")
	}
	input, err := NewConfigurationInput(ConfigurationInput{Configuration: validConfiguration(t), IsOfficial: true, IsPublic: true})
	if err != nil {
		t.Errorf("expected official public configuration to be accepted, got %v", err)
	}
	if !input.IsOfficial || !input.IsPublic {
		t.Errorf("expected flags to be preserved: %+v", input)
	}
}

func TestNewConfigurationInputRejectsBadMetadata(t *testing.T) {
	long := strings.Repeat("x", MaxTitleLength+1)
	if _, err := NewConfigurationInput(publicInput(t)); err != nil {
		t.Fatalf("baseline must be valid: %v", err)
	}
	baseline := publicInput(t)
	baseline.Title = &long
	if _, err := NewConfigurationInput(baseline); err == nil {
		t.Error("expected error for overlong title")
	}
	blank := " "
	baseline = publicInput(t)
	baseline.ListOfServices = []string{"ok", blank}
	if _, err := NewConfigurationInput(baseline); err == nil {
		t.Error("expected error for blank service entry")
	}
	many := make([]string, MaxServices+1)
	for i := range many {
		many[i] = "service"
	}
	baseline = publicInput(t)
	baseline.ListOfServices = many
	if _, err := NewConfigurationInput(baseline); err == nil {
		t.Error("expected error for too many services")
	}
}

func TestNewOwnerID(t *testing.T) {
	if _, err := NewOwnerID("  "); err == nil {
		t.Error("expected error for blank owner id")
	}
	owner, err := NewOwnerID("  user-123  ")
	if err != nil || owner != "user-123" {
		t.Errorf("expected trimmed owner id, got %q, %v", owner, err)
	}
}
