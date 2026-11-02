// Package domain defines converter records and validates their inputs.
package domain

import (
	"mime"
	"net/url"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

type Runtime string

const (
	RuntimePython Runtime = "python"
	RuntimeJava   Runtime = "java"
	RuntimeBinary Runtime = "binary"
)

type PluginInput struct {
	Name        string
	Description string
	Repository  url.URL
	Branch      string
	Runtime     Runtime
	Arguments   []string
}

type Plugin struct {
	ID uuid.UUID
	PluginInput
	Enabled   bool
	CreatedBy string
	UpdatedBy string
}

type Installation struct {
	Commit  string
	Runtime Runtime
}

type RelationInput struct {
	DistributionID uuid.UUID
	PluginID       uuid.UUID
	InputFormat    string
	OutputFormat   string
}

type Relation struct {
	ID uuid.UUID
	RelationInput
	CreatedBy string
	UpdatedBy string
}

func NewPluginInput(input PluginInput) (PluginInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Branch = strings.TrimSpace(input.Branch)
	if input.Name == "" {
		return PluginInput{}, apperror.Field("name", "Name is required.", nil)
	}
	repository := input.Repository
	if repository.Host == "" || repository.User != nil || (repository.Scheme != "https" && repository.Scheme != "http" && repository.Scheme != "git" && repository.Scheme != "ssh") {
		return PluginInput{}, apperror.Field("repository", "A public Git repository URL without credentials is required.", nil)
	}
	if input.Branch == "" || strings.HasPrefix(input.Branch, "-") {
		return PluginInput{}, apperror.Field("branch", "A Git branch is required.", nil)
	}
	switch input.Runtime {
	case RuntimePython, RuntimeJava, RuntimeBinary:
	default:
		return PluginInput{}, apperror.Field("runtime", "Runtime must be python, java, or binary.", nil)
	}
	if input.Arguments == nil {
		input.Arguments = []string{}
	}
	return input, nil
}

func NewRelationInput(input RelationInput) (RelationInput, error) {
	if input.DistributionID == uuid.Nil {
		return RelationInput{}, apperror.Field("distribution_id", "Distribution ID is required.", nil)
	}
	if input.PluginID == uuid.Nil {
		return RelationInput{}, apperror.Field("plugin_id", "Plugin ID is required.", nil)
	}
	if strings.TrimSpace(input.InputFormat) == "" {
		return RelationInput{}, apperror.Field("input_format", "Input format is required.", nil)
	}
	if _, _, err := mime.ParseMediaType(input.OutputFormat); err != nil {
		return RelationInput{}, apperror.Field("output_format", "Output format must be a valid MIME type.", err)
	}
	return input, nil
}
