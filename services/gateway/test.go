package main

type Dataset struct {
	Title         string
	Description   string
	Distributions []Distribution
}

type Distribution struct {
	Title       string
	Description string
	WebService  WebService
}

type WebService struct {
	Name        string
	Description string
	TemplateURL string
	Parameters  []Parameter
}

type Parameter interface {
	// GetVariable() string
	// GetProperty() string
	// GetLabel() string
	// IsRequired() bool
	// IsMultipleValues() bool
	// IsReadOnly() bool
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
