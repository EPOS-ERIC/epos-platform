package domain

import (
	"fmt"
	"mime"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

type RequestBodyInput struct {
	Required    *bool
	ContentType string
	Template    string
	Parameters  []ParameterInput
}

type RequestBody struct {
	Required    bool
	ContentType string
	Template    string
	Parameters  []Parameter
}

// Normalize validates caller-controlled request-body fields.
func (value *RequestBodyInput) Normalize() error {
	if value.Required == nil {
		value.Required = new(true)
	}
	value.ContentType = strings.TrimSpace(value.ContentType)
	if value.ContentType == "" {
		return apperror.Field("body_content_type", "Content type must not be blank.", nil)
	}
	contentType, err := normalizeMediaType(value.ContentType)
	if err != nil {
		return apperror.Field("body_content_type", "Content type must be a valid media type.", err)
	}
	value.ContentType = contentType
	templateValue, parameters, err := parseTemplateDefinition(value.Template, value.Parameters)
	if err != nil {
		return prefixOrField(err, "body_parameters", "body_template", "Template must be valid and reference declared parameters.")
	}
	value.Template = templateValue
	value.Parameters = parameters
	return nil
}

func normalizeMediaType(value string) (string, error) {
	mediaType, parameters, err := mime.ParseMediaType(value)
	if err != nil {
		return "", fmt.Errorf("parse MIME media type %q: %w: %w", value, err, apperror.Default(apperror.CodeValidationFailed))
	}
	if !strings.Contains(mediaType, "/") {
		return "", fmt.Errorf("MIME media type %q must contain type and subtype: %w", value, apperror.Default(apperror.CodeValidationFailed))
	}
	normalized := mime.FormatMediaType(mediaType, parameters)
	if normalized == "" {
		return "", fmt.Errorf("format MIME media type %q: %w", value, apperror.Default(apperror.CodeValidationFailed))
	}
	return normalized, nil
}
