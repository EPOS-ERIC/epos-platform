package metadata

import (
	"time"

	"github.com/google/uuid"
)

type Parameter interface {
	GetMetadata() ParameterMetadata
	Kind() ParameterKind
}

type ParameterMetadata struct {
	ID             uuid.UUID
	SourceIRI      string
	Kind           ParameterKind
	Variable       string
	Property       string
	Label          string
	Required       bool
	MultipleValues bool
	ReadOnly       bool
}

type StringParameter struct {
	Metadata      ParameterMetadata
	Default       *string
	AllowedValues []string
	Pattern       string
}

func (p StringParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p StringParameter) Kind() ParameterKind            { return ParameterKindString }

type BooleanParameter struct {
	Metadata ParameterMetadata
	Default  *bool
}

func (p BooleanParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p BooleanParameter) Kind() ParameterKind            { return ParameterKindBoolean }

type IntegerParameter struct {
	Metadata      ParameterMetadata
	Minimum       *int64
	Maximum       *int64
	Default       *int64
	AllowedValues []int64
}

func (p IntegerParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p IntegerParameter) Kind() ParameterKind            { return ParameterKindInteger }

type FloatParameter struct {
	Metadata      ParameterMetadata
	Minimum       *float64
	Maximum       *float64
	Default       *float64
	AllowedValues []float64
}

func (p FloatParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p FloatParameter) Kind() ParameterKind            { return ParameterKindFloat }

type DateTimeParameter struct {
	Metadata ParameterMetadata
	Minimum  *time.Time
	Maximum  *time.Time
	Default  *time.Time
	Pattern  string
}

func (p DateTimeParameter) GetMetadata() ParameterMetadata { return p.Metadata }
func (p DateTimeParameter) Kind() ParameterKind            { return ParameterKindDateTime }
