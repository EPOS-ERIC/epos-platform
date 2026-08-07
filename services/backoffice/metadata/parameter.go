package metadata

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Parameter interface {
	GetMetadata() ParameterMetadata
	Kind() ParameterKind
}

type ParameterMetadata struct {
	ID             uuid.UUID     `json:"id"`
	SourceIRI      string        `json:"source_iri"`
	Kind           ParameterKind `json:"kind"`
	Variable       string        `json:"variable"`
	Property       string        `json:"property"`
	Label          string        `json:"label"`
	Required       bool          `json:"required"`
	MultipleValues bool          `json:"multiple_values"`
	ReadOnly       bool          `json:"read_only"`
}

type StringParameter struct {
	Metadata      ParameterMetadata `json:"metadata"`
	Default       *string           `json:"default"`
	AllowedValues []string          `json:"allowed_values"`
	Pattern       string            `json:"pattern"`
}

func (p StringParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p StringParameter) Kind() ParameterKind            { return ParameterKindString }

type BooleanParameter struct {
	Metadata ParameterMetadata `json:"metadata"`
	Default  *bool             `json:"default"`
}

func (p BooleanParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p BooleanParameter) Kind() ParameterKind            { return ParameterKindBoolean }

type IntegerParameter struct {
	Metadata      ParameterMetadata `json:"metadata"`
	Minimum       *int64            `json:"minimum"`
	Maximum       *int64            `json:"maximum"`
	Default       *int64            `json:"default"`
	AllowedValues []int64           `json:"allowed_values"`
}

func (p IntegerParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p IntegerParameter) Kind() ParameterKind            { return ParameterKindInteger }

type FloatParameter struct {
	Metadata      ParameterMetadata `json:"metadata"`
	Minimum       *float64          `json:"minimum"`
	Maximum       *float64          `json:"maximum"`
	Default       *float64          `json:"default"`
	AllowedValues []float64         `json:"allowed_values"`
}

func (p FloatParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p FloatParameter) Kind() ParameterKind            { return ParameterKindFloat }

type DateTimeParameter struct {
	Metadata ParameterMetadata `json:"metadata"`
	Minimum  *time.Time        `json:"minimum"`
	Maximum  *time.Time        `json:"maximum"`
	Default  *time.Time        `json:"default"`
	Pattern  string            `json:"pattern"`
}

func (p DateTimeParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p DateTimeParameter) Kind() ParameterKind            { return ParameterKindDateTime }

// ParameterRecord maps a typed parameter to web_service_parameters kind and config columns.
type ParameterRecord struct {
	Kind     ParameterKind
	Metadata ParameterMetadata
	Config   json.RawMessage
}

func EncodeParameter(parameter Parameter) (ParameterRecord, error) {
	if parameter == nil {
		return ParameterRecord{}, fmt.Errorf("parameter is required")
	}

	var (
		config json.RawMessage
		err    error
	)
	switch p := parameter.(type) {
	case StringParameter:
		config, err = json.Marshal(struct {
			Default       *string  `json:"default,omitempty"`
			AllowedValues []string `json:"allowed_values,omitempty"`
			Pattern       string   `json:"pattern,omitempty"`
		}{p.Default, p.AllowedValues, p.Pattern})
	case BooleanParameter:
		config, err = json.Marshal(struct {
			Default *bool `json:"default,omitempty"`
		}{p.Default})
	case IntegerParameter:
		config, err = json.Marshal(struct {
			Minimum       *int64  `json:"minimum,omitempty"`
			Maximum       *int64  `json:"maximum,omitempty"`
			Default       *int64  `json:"default,omitempty"`
			AllowedValues []int64 `json:"allowed_values,omitempty"`
		}{p.Minimum, p.Maximum, p.Default, p.AllowedValues})
	case FloatParameter:
		config, err = json.Marshal(struct {
			Minimum       *float64  `json:"minimum,omitempty"`
			Maximum       *float64  `json:"maximum,omitempty"`
			Default       *float64  `json:"default,omitempty"`
			AllowedValues []float64 `json:"allowed_values,omitempty"`
		}{p.Minimum, p.Maximum, p.Default, p.AllowedValues})
	case DateTimeParameter:
		config, err = json.Marshal(struct {
			Minimum *time.Time `json:"minimum,omitempty"`
			Maximum *time.Time `json:"maximum,omitempty"`
			Default *time.Time `json:"default,omitempty"`
			Pattern string     `json:"pattern,omitempty"`
		}{p.Minimum, p.Maximum, p.Default, p.Pattern})
	default:
		return ParameterRecord{}, fmt.Errorf("unsupported parameter type %T", parameter)
	}
	if err != nil {
		return ParameterRecord{}, fmt.Errorf("marshal %s parameter config: %w", parameter.Kind(), err)
	}

	return ParameterRecord{
		Kind:     parameter.Kind(),
		Metadata: parameter.GetMetadata(),
		Config:   config,
	}, nil
}

func DecodeParameter(record ParameterRecord) (Parameter, error) {
	record.Metadata.Kind = record.Kind

	switch record.Kind {
	case ParameterKindString:
		var config struct {
			Default       *string  `json:"default"`
			AllowedValues []string `json:"allowed_values"`
			Pattern       string   `json:"pattern"`
		}
		if err := json.Unmarshal(record.Config, &config); err != nil {
			return nil, fmt.Errorf("decode string parameter config: %w", err)
		}
		return StringParameter{record.Metadata, config.Default, config.AllowedValues, config.Pattern}, nil
	case ParameterKindBoolean:
		var config struct {
			Default *bool `json:"default"`
		}
		if err := json.Unmarshal(record.Config, &config); err != nil {
			return nil, fmt.Errorf("decode boolean parameter config: %w", err)
		}
		return BooleanParameter{record.Metadata, config.Default}, nil
	case ParameterKindInteger:
		var config struct {
			Minimum       *int64  `json:"minimum"`
			Maximum       *int64  `json:"maximum"`
			Default       *int64  `json:"default"`
			AllowedValues []int64 `json:"allowed_values"`
		}
		if err := json.Unmarshal(record.Config, &config); err != nil {
			return nil, fmt.Errorf("decode integer parameter config: %w", err)
		}
		return IntegerParameter{record.Metadata, config.Minimum, config.Maximum, config.Default, config.AllowedValues}, nil
	case ParameterKindFloat:
		var config struct {
			Minimum       *float64  `json:"minimum"`
			Maximum       *float64  `json:"maximum"`
			Default       *float64  `json:"default"`
			AllowedValues []float64 `json:"allowed_values"`
		}
		if err := json.Unmarshal(record.Config, &config); err != nil {
			return nil, fmt.Errorf("decode float parameter config: %w", err)
		}
		return FloatParameter{record.Metadata, config.Minimum, config.Maximum, config.Default, config.AllowedValues}, nil
	case ParameterKindDateTime:
		var config struct {
			Minimum *time.Time `json:"minimum"`
			Maximum *time.Time `json:"maximum"`
			Default *time.Time `json:"default"`
			Pattern string     `json:"pattern"`
		}
		if err := json.Unmarshal(record.Config, &config); err != nil {
			return nil, fmt.Errorf("decode datetime parameter config: %w", err)
		}
		return DateTimeParameter{record.Metadata, config.Minimum, config.Maximum, config.Default, config.Pattern}, nil
	default:
		return nil, fmt.Errorf("unsupported parameter kind %q", record.Kind)
	}
}
