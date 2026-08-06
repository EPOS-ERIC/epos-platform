package main

import "time"

type Parameter interface {
	GetMetadata() ParameterMetadata
}

type ParameterMetadata struct {
	Variable       string
	Property       string // this indicates the "type" of the parameter eg: schema:startDate, we could probably use an enum here
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

type BooleanParameter struct {
	Metadata ParameterMetadata
	Default  *bool
}

func (p BooleanParameter) GetMetadata() ParameterMetadata { return p.Metadata }

type IntegerParameter struct {
	Metadata      ParameterMetadata
	Minimum       *int64
	Maximum       *int64
	Default       *int64
	AllowedValues []int64
}

func (p IntegerParameter) GetMetadata() ParameterMetadata { return p.Metadata }

type FloatParameter struct {
	Metadata      ParameterMetadata
	Minimum       *float64
	Maximum       *float64
	Default       *float64
	AllowedValues []float64
}

func (p FloatParameter) GetMetadata() ParameterMetadata { return p.Metadata }

type DateTimeParameter struct {
	Metadata ParameterMetadata
	Minimum  *time.Time
	Maximum  *time.Time
	Default  *time.Time
	Pattern  string
}

func (p DateTimeParameter) GetMetadata() ParameterMetadata { return p.Metadata }
