// Package domain holds the sharing service business model.
//
// A configuration persists an opaque client configuration document (raw
// JSON) plus optional descriptive metadata, visibility flags and ownership.
// The configuration payload is deliberately schemaless so frontend payloads
// can evolve without a contract change; only representation-level invariants
// (valid JSON, size and field length limits) are enforced here.
//
// Visibility rules: official configurations are curated and can only be
// public. Ownership is assigned by the transport layer from the caller's
// authenticated identity, never from client input.
package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Limits mirror the protobuf contract in configuration/v1/configuration.proto.
const (
	MaxConfigurationBytes = 8 * 1024 * 1024
	MaxTitleLength        = 500
	MaxDescriptionLength  = 10000
	MaxServiceNameLength  = 500
	MaxServices           = 100
)

// ConfigurationInput holds caller-supplied fields. Required field is T,
// nullable scalar is *T, collection is []T; absent and null both decode to
// nil and mean clear on replace. IsPublic is resolved (absent means public)
// before validation; ownership is attached separately by the caller.
type ConfigurationInput struct {
	Configuration  json.RawMessage `json:"configuration"`
	Title          *string         `json:"title"`
	Description    *string         `json:"description"`
	ListOfServices []string        `json:"listOfServices"`
	IsOfficial     bool            `json:"isOfficial"`
	IsPublic       bool            `json:"isPublic"`
}

// Configuration is a ConfigurationInput plus system-assigned identifiers,
// ownership and timestamps.
type Configuration struct {
	ConfigurationInput
	ID        uuid.UUID `json:"id"`
	OwnerID   string    `json:"ownerId"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
}

// NewConfigurationInput validates and normalizes caller-supplied fields.
func NewConfigurationInput(input ConfigurationInput) (ConfigurationInput, error) {
	configuration, err := normalizeConfiguration(input.Configuration)
	if err != nil {
		return ConfigurationInput{}, err
	}
	input.Configuration = configuration

	title, err := normalizeOptionalText("title", input.Title, MaxTitleLength)
	if err != nil {
		return ConfigurationInput{}, err
	}
	input.Title = title

	description, err := normalizeOptionalText("description", input.Description, MaxDescriptionLength)
	if err != nil {
		return ConfigurationInput{}, err
	}
	input.Description = description

	services, err := normalizeServices(input.ListOfServices)
	if err != nil {
		return ConfigurationInput{}, err
	}
	input.ListOfServices = services

	if input.IsOfficial && !input.IsPublic {
		return ConfigurationInput{}, errors.New("official configurations must be public")
	}

	return input, nil
}

// NewOwnerID validates an owner identifier supplied by the transport layer.
func NewOwnerID(ownerID string) (string, error) {
	trimmed := strings.TrimSpace(ownerID)
	if trimmed == "" {
		return "", errors.New("owner id is required")
	}
	return trimmed, nil
}

func normalizeConfiguration(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, errors.New("configuration is required")
	}
	if len(raw) > MaxConfigurationBytes {
		return nil, fmt.Errorf("configuration exceeds %d bytes", MaxConfigurationBytes)
	}
	if !json.Valid(raw) {
		return nil, errors.New("configuration must be valid JSON")
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, errors.New("configuration must be valid JSON")
	}
	canonical, err := json.Marshal(decoded)
	if err != nil {
		return nil, errors.New("configuration cannot be serialized")
	}
	return canonical, nil
}

func normalizeOptionalText(field string, value *string, maxLength int) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) > maxLength {
		return nil, fmt.Errorf("%s exceeds %d characters", field, maxLength)
	}
	return &trimmed, nil
}

func normalizeServices(values []string) ([]string, error) {
	if values == nil {
		return nil, nil
	}
	if len(values) > MaxServices {
		return nil, fmt.Errorf("listOfServices exceeds %d entries", MaxServices)
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, errors.New("listOfServices entries must not be blank")
		}
		if len(trimmed) > MaxServiceNameLength {
			return nil, fmt.Errorf("listOfServices entries exceed %d characters", MaxServiceNameLength)
		}
		normalized = append(normalized, trimmed)
	}
	return normalized, nil
}
