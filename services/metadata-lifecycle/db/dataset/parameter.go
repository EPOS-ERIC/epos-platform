package dataset

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

// TODO: since we store json in the db we should have a test here or something that makes sure we always have the correct shape passed through to avoid inconsistencies in the data in the db

type parameterRecord struct {
	kind     domain.ParameterKind
	metadata domain.ParameterMetadata
	config   json.RawMessage
}

func encodeParameter(parameter domain.Parameter) (parameterRecord, error) {
	if parameter == nil {
		return parameterRecord{}, fmt.Errorf("parameter is required")
	}

	var (
		config json.RawMessage
		err    error
	)
	switch value := parameter.(type) {
	case domain.StringParameter:
		config, err = json.Marshal(struct {
			Default       *string  `json:"default,omitempty"`
			AllowedValues []string `json:"allowed_values,omitempty"`
			Pattern       string   `json:"pattern,omitempty"`
		}{value.Default, value.AllowedValues, value.Pattern})
	case domain.BooleanParameter:
		config, err = json.Marshal(struct {
			Default *bool `json:"default,omitempty"`
		}{value.Default})
	case domain.IntegerParameter:
		config, err = json.Marshal(struct {
			Minimum       *int64  `json:"minimum,omitempty"`
			Maximum       *int64  `json:"maximum,omitempty"`
			Default       *int64  `json:"default,omitempty"`
			AllowedValues []int64 `json:"allowed_values,omitempty"`
		}{value.Minimum, value.Maximum, value.Default, value.AllowedValues})
	case domain.FloatParameter:
		config, err = json.Marshal(struct {
			Minimum       *float64  `json:"minimum,omitempty"`
			Maximum       *float64  `json:"maximum,omitempty"`
			Default       *float64  `json:"default,omitempty"`
			AllowedValues []float64 `json:"allowed_values,omitempty"`
		}{value.Minimum, value.Maximum, value.Default, value.AllowedValues})
	case domain.DateTimeParameter:
		config, err = json.Marshal(struct {
			Minimum *time.Time `json:"minimum,omitempty"`
			Maximum *time.Time `json:"maximum,omitempty"`
			Default *time.Time `json:"default,omitempty"`
			Pattern string     `json:"pattern,omitempty"`
		}{value.Minimum, value.Maximum, value.Default, value.Pattern})
	default:
		return parameterRecord{}, fmt.Errorf("unsupported parameter type %T", parameter)
	}
	if err != nil {
		return parameterRecord{}, fmt.Errorf("marshal %s parameter config: %w", parameter.Kind(), err)
	}

	return parameterRecord{kind: parameter.Kind(), metadata: parameter.GetMetadata(), config: config}, nil
}

func decodeParameter(record parameterRecord) (domain.Parameter, error) {
	record.metadata.Kind = record.kind

	switch record.kind {
	case domain.ParameterKindString:
		var config struct {
			Default       *string  `json:"default"`
			AllowedValues []string `json:"allowed_values"`
			Pattern       string   `json:"pattern"`
		}
		if err := json.Unmarshal(record.config, &config); err != nil {
			return nil, fmt.Errorf("decode string parameter config: %w", err)
		}
		return domain.StringParameter{
			Metadata:      record.metadata,
			Default:       config.Default,
			AllowedValues: config.AllowedValues,
			Pattern:       config.Pattern,
		}, nil
	case domain.ParameterKindBoolean:
		var config struct {
			Default *bool `json:"default"`
		}
		if err := json.Unmarshal(record.config, &config); err != nil {
			return nil, fmt.Errorf("decode boolean parameter config: %w", err)
		}
		return domain.BooleanParameter{
			Metadata: record.metadata,
			Default:  config.Default,
		}, nil
	case domain.ParameterKindInteger:
		var config struct {
			Minimum       *int64  `json:"minimum"`
			Maximum       *int64  `json:"maximum"`
			Default       *int64  `json:"default"`
			AllowedValues []int64 `json:"allowed_values"`
		}
		if err := json.Unmarshal(record.config, &config); err != nil {
			return nil, fmt.Errorf("decode integer parameter config: %w", err)
		}
		return domain.IntegerParameter{
			Metadata:      record.metadata,
			Minimum:       config.Minimum,
			Maximum:       config.Maximum,
			Default:       config.Default,
			AllowedValues: config.AllowedValues,
		}, nil
	case domain.ParameterKindFloat:
		var config struct {
			Minimum       *float64  `json:"minimum"`
			Maximum       *float64  `json:"maximum"`
			Default       *float64  `json:"default"`
			AllowedValues []float64 `json:"allowed_values"`
		}
		if err := json.Unmarshal(record.config, &config); err != nil {
			return nil, fmt.Errorf("decode float parameter config: %w", err)
		}
		return domain.FloatParameter{
			Metadata:      record.metadata,
			Minimum:       config.Minimum,
			Maximum:       config.Maximum,
			Default:       config.Default,
			AllowedValues: config.AllowedValues,
		}, nil
	case domain.ParameterKindDateTime:
		var config struct {
			Minimum *time.Time `json:"minimum"`
			Maximum *time.Time `json:"maximum"`
			Default *time.Time `json:"default"`
			Pattern string     `json:"pattern"`
		}
		if err := json.Unmarshal(record.config, &config); err != nil {
			return nil, fmt.Errorf("decode datetime parameter config: %w", err)
		}
		return domain.DateTimeParameter{
			Metadata: record.metadata,
			Minimum:  config.Minimum,
			Maximum:  config.Maximum,
			Default:  config.Default,
			Pattern:  config.Pattern,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported parameter kind %q", record.kind)
	}
}
