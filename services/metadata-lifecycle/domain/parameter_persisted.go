package domain

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

func validatePersistedWebServiceParameters(values []WebServiceParameter) error {
	for _, value := range values {
		if err := validatePersistedParameter(value.Parameter); err != nil {
			return err
		}
	}
	return nil
}

func applyNormalizedWebServiceParameters(values []WebServiceParameter, inputs []WebServiceParameterInput) ([]WebServiceParameter, error) {
	if values == nil {
		values = make([]WebServiceParameter, 0)
	}
	slices.SortFunc(values, func(a, b WebServiceParameter) int {
		return cmp.Or(cmp.Compare(a.Position, b.Position), strings.Compare(a.Name, b.Name))
	})
	inputs = slices.Clone(inputs)
	slices.SortFunc(inputs, func(a, b WebServiceParameterInput) int {
		return cmp.Or(cmp.Compare(a.Position, b.Position), strings.Compare(a.Name, b.Name))
	})
	for i := range values {
		if err := applyNormalizedParameter(&values[i].Parameter, inputs[i].ParameterInput); err != nil {
			return nil, err
		}
		values[i].Location = inputs[i].Location
		values[i].Explode = inputs[i].Explode
	}
	return values, nil
}

func webServiceParameterInputs(values []WebServiceParameter) []WebServiceParameterInput {
	result := make([]WebServiceParameterInput, len(values))
	for i, value := range values {
		result[i] = WebServiceParameterInput{
			ParameterInput: value.Input(),
			Location:       value.Location,
			Explode:        value.Explode,
		}
	}
	return result
}

func (value Parameter) Input() ParameterInput {
	input := value.ParameterInput
	if value.ResourceID != uuid.Nil {
		input.ResourceID = &value.ResourceID
	}
	input.Schema = parameterSchemaInput(input.Schema)
	return input
}

func parameterSchemaInput(value ParameterSchema) ParameterSchema {
	switch schema := value.(type) {
	case TemplateSchema:
		return TemplateSchemaInput{Template: schema.Template, Parameters: parameterInputs(schema.Parameters)}
	default:
		return value
	}
}

func parameterInputs(values []Parameter) []ParameterInput {
	result := make([]ParameterInput, len(values))
	for i, value := range values {
		result[i] = value.Input()
	}
	return result
}

func requestBodyInput(value *RequestBody) *RequestBodyInput {
	return &RequestBodyInput{
		Required:    new(value.Required),
		ContentType: value.ContentType,
		Template:    value.Template,
		Parameters:  parameterInputs(value.Parameters),
	}
}

func validatePersistedParameter(value Parameter) error {
	if value.ID == uuid.Nil {
		return fmt.Errorf("parameter ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if schema, ok := value.Schema.(TemplateSchema); ok {
		return validatePersistedParameters(schema.Parameters)
	}
	if _, ok := value.Schema.(TemplateSchemaInput); ok {
		return fmt.Errorf("input template schema in persisted parameter: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

func validatePersistedParameters(values []Parameter) error {
	for _, value := range values {
		if err := validatePersistedParameter(value); err != nil {
			return err
		}
	}
	return nil
}

func applyNormalizedParameter(value *Parameter, input ParameterInput) error {
	if value.ID == uuid.Nil || value.Name != input.Name {
		return fmt.Errorf("parameter identity is invalid: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if schema, ok := input.Schema.(TemplateSchemaInput); ok {
		persisted, ok := value.Schema.(TemplateSchema)
		if !ok {
			return fmt.Errorf("persisted parameter template is missing: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		parameters, err := applyNormalizedParameters(persisted.Parameters, schema.Parameters)
		if err != nil {
			return err
		}
		input.Schema = TemplateSchema{Template: schema.Template, Parameters: parameters}
	}
	value.ParameterInput = input
	return nil
}

func applyNormalizedParameters(values []Parameter, inputs []ParameterInput) ([]Parameter, error) {
	if values == nil {
		values = make([]Parameter, 0)
	}
	slices.SortFunc(values, func(a, b Parameter) int {
		return cmp.Or(cmp.Compare(a.Position, b.Position), strings.Compare(a.Name, b.Name))
	})
	inputs = slices.Clone(inputs)
	slices.SortFunc(inputs, func(a, b ParameterInput) int {
		return cmp.Or(cmp.Compare(a.Position, b.Position), strings.Compare(a.Name, b.Name))
	})
	for i := range values {
		if err := applyNormalizedParameter(&values[i], inputs[i]); err != nil {
			return nil, err
		}
	}
	return values, nil
}
