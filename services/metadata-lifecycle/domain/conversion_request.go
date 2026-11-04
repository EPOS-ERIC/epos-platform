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

// NewConversionRequestInput validates and normalizes conversion request details.
func NewConversionRequestInput(input ConversionRequestInput) (ConversionRequestInput, error) {
	input.InputFormat = strings.TrimSpace(input.InputFormat)
	if input.InputFormat == "" {
		return ConversionRequestInput{}, apperror.Field("input_format", "Input format is required.", nil)
	}

	if len(input.Visualizations) == 0 {
		return ConversionRequestInput{}, apperror.Field("visualizations", "Select at least one visualization.", nil)
	}

	for _, visualization := range input.Visualizations {
		if !visualization.Valid() {
			return ConversionRequestInput{}, apperror.Field("visualizations", "Unknown visualization.", nil)
		}
	}

	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" {
		return ConversionRequestInput{}, apperror.Field("message", "Message is required.", nil)
	}

	return input, nil
}
