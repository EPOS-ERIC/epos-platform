package dataset

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/metadata"
)

type parameterRecord struct {
	kind     metadata.ParameterKind
	metadata metadata.ParameterMetadata
	config   json.RawMessage
}

func encodeParameter(parameter metadata.Parameter) (parameterRecord, error) {
	if parameter == nil {
		return parameterRecord{}, fmt.Errorf("parameter is required")
	}

	var (
		config json.RawMessage
		err    error
	)
	switch value := parameter.(type) {
	case metadata.StringParameter:
		config, err = json.Marshal(struct {
			Default       *string  `json:"default,omitempty"`
			AllowedValues []string `json:"allowed_values,omitempty"`
			Pattern       string   `json:"pattern,omitempty"`
		}{value.Default, value.AllowedValues, value.Pattern})
	case metadata.BooleanParameter:
		config, err = json.Marshal(struct {
			Default *bool `json:"default,omitempty"`
		}{value.Default})
	case metadata.IntegerParameter:
		config, err = json.Marshal(struct {
			Minimum       *int64  `json:"minimum,omitempty"`
			Maximum       *int64  `json:"maximum,omitempty"`
			Default       *int64  `json:"default,omitempty"`
			AllowedValues []int64 `json:"allowed_values,omitempty"`
		}{value.Minimum, value.Maximum, value.Default, value.AllowedValues})
	case metadata.FloatParameter:
		config, err = json.Marshal(struct {
			Minimum       *float64  `json:"minimum,omitempty"`
			Maximum       *float64  `json:"maximum,omitempty"`
			Default       *float64  `json:"default,omitempty"`
			AllowedValues []float64 `json:"allowed_values,omitempty"`
		}{value.Minimum, value.Maximum, value.Default, value.AllowedValues})
	case metadata.DateTimeParameter:
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

func decodeParameter(record parameterRecord) (metadata.Parameter, error) {
	record.metadata.Kind = record.kind

	switch record.kind {
	case metadata.ParameterKindString:
		var config struct {
			Default       *string  `json:"default"`
			AllowedValues []string `json:"allowed_values"`
			Pattern       string   `json:"pattern"`
		}
		if err := json.Unmarshal(record.config, &config); err != nil {
			return nil, fmt.Errorf("decode string parameter config: %w", err)
		}
		return metadata.StringParameter{
			Metadata:      record.metadata,
			Default:       config.Default,
			AllowedValues: config.AllowedValues,
			Pattern:       config.Pattern,
		}, nil
	case metadata.ParameterKindBoolean:
		var config struct {
			Default *bool `json:"default"`
		}
		if err := json.Unmarshal(record.config, &config); err != nil {
			return nil, fmt.Errorf("decode boolean parameter config: %w", err)
		}
		return metadata.BooleanParameter{
			Metadata: record.metadata,
			Default:  config.Default,
		}, nil
	case metadata.ParameterKindInteger:
		var config struct {
			Minimum       *int64  `json:"minimum"`
			Maximum       *int64  `json:"maximum"`
			Default       *int64  `json:"default"`
			AllowedValues []int64 `json:"allowed_values"`
		}
		if err := json.Unmarshal(record.config, &config); err != nil {
			return nil, fmt.Errorf("decode integer parameter config: %w", err)
		}
		return metadata.IntegerParameter{
			Metadata:      record.metadata,
			Minimum:       config.Minimum,
			Maximum:       config.Maximum,
			Default:       config.Default,
			AllowedValues: config.AllowedValues,
		}, nil
	case metadata.ParameterKindFloat:
		var config struct {
			Minimum       *float64  `json:"minimum"`
			Maximum       *float64  `json:"maximum"`
			Default       *float64  `json:"default"`
			AllowedValues []float64 `json:"allowed_values"`
		}
		if err := json.Unmarshal(record.config, &config); err != nil {
			return nil, fmt.Errorf("decode float parameter config: %w", err)
		}
		return metadata.FloatParameter{
			Metadata:      record.metadata,
			Minimum:       config.Minimum,
			Maximum:       config.Maximum,
			Default:       config.Default,
			AllowedValues: config.AllowedValues,
		}, nil
	case metadata.ParameterKindDateTime:
		var config struct {
			Minimum *time.Time `json:"minimum"`
			Maximum *time.Time `json:"maximum"`
			Default *time.Time `json:"default"`
			Pattern string     `json:"pattern"`
		}
		if err := json.Unmarshal(record.config, &config); err != nil {
			return nil, fmt.Errorf("decode datetime parameter config: %w", err)
		}
		return metadata.DateTimeParameter{
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
