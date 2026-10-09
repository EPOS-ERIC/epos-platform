package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

const (
	MaxConfigurationBytes                = 8 * 1024 * 1024
	MaxTitleLength                       = 500
	MaxDescriptionLength                 = 10000
	MaxServiceNameLength                 = 500
	MaxServices                          = 100
	MaxNonOfficialConfigurationsPerOwner = 50
)

type ConfigurationInput struct {
	Configuration  json.RawMessage
	Title          string
	Description    *string
	ListOfServices []string
	IsPublic       *bool
}

type Configuration struct {
	ID             uuid.UUID
	Configuration  json.RawMessage
	Title          string
	Description    *string
	ListOfServices []string
	IsOfficial     bool
	IsPublic       bool
	OwnerID        Actor
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ConfigurationSummary struct {
	ID             uuid.UUID
	Title          string
	Description    *string
	ListOfServices []string
	IsOfficial     bool
	IsPublic       bool
	OwnerID        Actor
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewConfigurationInput(input ConfigurationInput) (ConfigurationInput, error) {
	configuration, err := normalizeConfiguration(input.Configuration)
	if err != nil {
		return ConfigurationInput{}, err
	}
	input.Configuration = configuration

	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return ConfigurationInput{}, apperror.Field("title", "Title must not be blank.", nil)
	}
	if utf8.RuneCountInString(input.Title) > MaxTitleLength {
		return ConfigurationInput{}, apperror.Field("title", "Title must contain no more than 500 characters.", nil)
	}

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

	return input, nil
}

func NewConfiguration(id uuid.UUID, input ConfigurationInput, isOfficial bool, ownerID string, createdAt, updatedAt time.Time) (Configuration, error) {
	input, err := NewConfigurationInput(input)
	if err != nil {
		return Configuration{}, apperror.Wrap(apperror.CodeInternal, "", err)
	}
	owner, err := NewActor(ownerID)
	if err != nil {
		return Configuration{}, apperror.Wrap(apperror.CodeInternal, "", err)
	}
	if id == uuid.Nil || createdAt.IsZero() || updatedAt.IsZero() || createdAt.Year() < 1 || createdAt.Year() > 9999 || updatedAt.Year() < 1 || updatedAt.Year() > 9999 {
		return Configuration{}, apperror.Default(apperror.CodeInternal)
	}
	isPublic := input.IsPublic != nil && *input.IsPublic
	if isOfficial && !isPublic {
		return Configuration{}, apperror.Default(apperror.CodeInternal)
	}
	return Configuration{
		ID:             id,
		Configuration:  input.Configuration,
		Title:          input.Title,
		Description:    input.Description,
		ListOfServices: input.ListOfServices,
		IsOfficial:     isOfficial,
		IsPublic:       isPublic,
		OwnerID:        owner,
		CreatedAt:      createdAt.UTC(),
		UpdatedAt:      updatedAt.UTC(),
	}, nil
}

func NewConfigurationSummary(id uuid.UUID, title string, description *string, services []string, isOfficial, isPublic bool, ownerID string, createdAt, updatedAt time.Time) (ConfigurationSummary, error) {
	if id == uuid.Nil || createdAt.IsZero() || updatedAt.IsZero() || createdAt.Year() < 1 || createdAt.Year() > 9999 || updatedAt.Year() < 1 || updatedAt.Year() > 9999 {
		return ConfigurationSummary{}, apperror.Default(apperror.CodeInternal)
	}
	if isOfficial && !isPublic {
		return ConfigurationSummary{}, apperror.Default(apperror.CodeInternal)
	}
	owner, err := NewActor(ownerID)
	if err != nil {
		return ConfigurationSummary{}, apperror.Wrap(apperror.CodeInternal, "", err)
	}
	input, err := NewConfigurationInput(ConfigurationInput{
		Configuration:  json.RawMessage(`{}`),
		Title:          title,
		Description:    description,
		ListOfServices: services,
		IsPublic:       new(isPublic),
	})
	if err != nil {
		return ConfigurationSummary{}, apperror.Wrap(apperror.CodeInternal, "", err)
	}
	return ConfigurationSummary{
		ID:             id,
		Title:          input.Title,
		Description:    input.Description,
		ListOfServices: input.ListOfServices,
		IsOfficial:     isOfficial,
		IsPublic:       isPublic,
		OwnerID:        owner,
		CreatedAt:      createdAt.UTC(),
		UpdatedAt:      updatedAt.UTC(),
	}, nil
}

func normalizeConfiguration(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, apperror.Field("configuration", "Configuration is required.", nil)
	}
	if len(raw) > MaxConfigurationBytes {
		return nil, apperror.Field("configuration", fmt.Sprintf("Configuration exceeds %d bytes.", MaxConfigurationBytes), nil)
	}
	if !json.Valid(raw) {
		return nil, apperror.Field("configuration", "Configuration must be valid JSON.", nil)
	}
	return raw, nil
}

func normalizeOptionalText(field string, value *string, maxLength int) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > maxLength {
		return nil, apperror.Field(field, fmt.Sprintf("Must contain no more than %d characters.", maxLength), nil)
	}
	return &trimmed, nil
}

func normalizeServices(values []string) ([]string, error) {
	if values == nil {
		return nil, nil
	}
	if len(values) > MaxServices {
		return nil, apperror.Field("list_of_services", fmt.Sprintf("Must contain no more than %d services.", MaxServices), nil)
	}
	normalized := make([]string, 0, len(values))
	for i, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, apperror.Field(fmt.Sprintf("list_of_services[%d]", i), "Service must not be blank.", nil)
		}
		if utf8.RuneCountInString(trimmed) > MaxServiceNameLength {
			return nil, apperror.Field(fmt.Sprintf("list_of_services[%d]", i), fmt.Sprintf("Must contain no more than %d characters.", MaxServiceNameLength), nil)
		}
		normalized = append(normalized, trimmed)
	}
	return normalized, nil
}
