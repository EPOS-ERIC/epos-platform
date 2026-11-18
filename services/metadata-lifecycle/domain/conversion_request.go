package domain

import (
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

// ConversionRequestInput contains the details supplied by a requester.
type ConversionRequestInput struct {
	InputFormat    string
	Visualizations []WebServiceVisibleOn
	Message        string
}

// Normalize validates and normalizes conversion request details.
func (input *ConversionRequestInput) Normalize() error {
	input.InputFormat = strings.TrimSpace(input.InputFormat)
	if input.InputFormat == "" {
		return apperror.Field("input_format", "Input format is required.", nil)
	}

	if len(input.Visualizations) == 0 {
		return apperror.Field("visualizations", "Select at least one visualization.", nil)
	}

	for _, visualization := range input.Visualizations {
		if !visualization.Valid() {
			return apperror.Field("visualizations", "Unknown visualization.", nil)
		}
	}

	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" {
		return apperror.Field("message", "Message is required.", nil)
	}

	return nil
}
