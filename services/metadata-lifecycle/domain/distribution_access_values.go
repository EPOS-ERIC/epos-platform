package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

// accessValues tracks validated parameter values and whether the caller supplied them.
// Template activation uses this distinction to reject partially supplied optional templates.
type accessValues struct {
	// resolved contains supplied or defaulted values by parameter UUID; scalars use one-element arrays.
	resolved map[uuid.UUID]AccessArray
	// supplied records explicit caller assignments, excluding metadata defaults.
	supplied map[uuid.UUID]bool
}

// resolveAccessValues validates assignments against top-level, template-child and body-child
// metadata, then applies defaults to omitted values. It rejects unknown, duplicate,
// read-only and direct template assignments. Required-value checks happen during rendering,
// when optional template and body activation can be determined.
func (service GenericWebServiceConfiguration) resolveAccessValues(input DistributionAccessInput) (accessValues, error) {
	parameters := make(map[uuid.UUID]Parameter)
	for _, parameter := range service.Parameters {
		parameters[parameter.ID] = parameter.Parameter
		if schema, ok := parameter.Schema.(TemplateSchema); ok {
			for _, child := range schema.Parameters {
				parameters[child.ID] = child
			}
		}
	}

	if service.Body != nil {
		for _, child := range service.Body.Parameters {
			parameters[child.ID] = child
		}
	}

	values := accessValues{
		resolved: make(map[uuid.UUID]AccessArray, len(parameters)),
		supplied: make(map[uuid.UUID]bool, len(input.Parameters)),
	}

	for i, assignment := range input.Parameters {
		target := fmt.Sprintf("parameters[%d]", i)
		parameter, exists := parameters[assignment.ID]
		if !exists {
			return accessValues{}, invalidField(target+".id", "Parameter does not exist in this distribution.", nil)
		}

		if values.supplied[assignment.ID] {
			return accessValues{}, invalidField(target+".id", "Parameter must not be assigned more than once.", nil)
		}

		if parameter.ReadOnly {
			return accessValues{}, invalidField(target+".id", "Read-only parameters cannot be assigned.", nil)
		}

		if _, template := parameter.Schema.(TemplateSchema); template {
			return accessValues{}, invalidField(target+".id", "Supply template child values instead of the template parameter itself.", nil)
		}

		resolved, err := resolveAccessValue(parameter.Schema, assignment.Value)
		if err != nil {
			return accessValues{}, apperror.PrefixTarget(err, target)
		}

		values.resolved[assignment.ID] = resolved
		values.supplied[assignment.ID] = true
	}

	for id, parameter := range parameters {
		if values.supplied[id] {
			continue
		}

		value := parameter.Schema.DefaultValue()
		if value == nil {
			continue
		}

		resolved, err := resolveAccessValue(parameter.Schema, value)
		if err != nil {
			return accessValues{}, internalValidationError(err)
		}

		values.resolved[id] = resolved
	}

	return values, nil
}

// resolveAccessValue checks scalar/array shape against schema multiplicity and
// validates each element. It rejects missing values and empty arrays, and returns scalars as
// one-element arrays so rendering can process both shapes uniformly.
func resolveAccessValue(schema ParameterSchema, value AccessValue) (AccessArray, error) {
	if value == nil {
		return nil, apperror.Field("value", "Value is required.", nil)
	}

	items, array := value.(AccessArray)
	if array != schema.AllowsMultipleValues() {
		expected := "a scalar"
		if schema.AllowsMultipleValues() {
			expected = "an array"
		}
		return nil, apperror.Field("value", fmt.Sprintf("Expected %s; received %s.", expected, value.TypeName()), nil)
	}

	if !array {
		scalar, ok := value.(AccessScalar)
		if !ok {
			return nil, apperror.Field("value", "Value must match parameter type and constraints.", nil)
		}

		items = AccessArray{scalar}
	}

	if len(items) == 0 {
		return nil, apperror.Field("value", "Arrays must contain at least one value.", nil)
	}

	result := make(AccessArray, len(items))
	for i, item := range items {
		resolved, err := validateAccessScalar(schema, item)
		if err != nil {
			return nil, err
		}

		result[i] = resolved
	}

	return result, nil
}

// validateAccessScalar checks a value against an already-validated scalar schema,
// including its type, enum, pattern and range constraints. FLOAT schemas accept
// integers; DATETIME schemas parse RFC 3339 strings and return UTC strings using
// the configured layout, or RFC 3339 when no layout is configured.
func validateAccessScalar(schema ParameterSchema, value AccessScalar) (AccessScalar, error) {
	switch schema := schema.(type) {
	case StringSchema:
		if item, ok := value.(AccessString); ok {
			text := string(item)
			if schema.Pattern != nil {
				pattern, err := regexp.Compile(*schema.Pattern)
				if err != nil {
					return nil, internalValidationError(err)
				}

				if !pattern.MatchString(text) {
					return nil, apperror.Field("value", fmt.Sprintf("Value must match pattern %q.", *schema.Pattern), nil)
				}
			}

			if len(schema.Enum) == 0 || slices.ContainsFunc(schema.Enum, func(option StringEnumOption) bool { return option.Value == text }) {
				return item, nil
			}
		}
	case BooleanSchema:
		if item, ok := value.(AccessBoolean); ok {
			return item, nil
		}
	case IntegerSchema:
		if item, ok := value.(AccessInteger); ok {
			number := int64(item)
			if integerInRange(number, schema.Minimum, schema.Maximum) &&
				(len(schema.Enum) == 0 || slices.ContainsFunc(schema.Enum, func(option IntegerEnumOption) bool { return option.Value == number })) {
				return item, nil
			}
		}
	case FloatSchema:
		if item, ok := value.(AccessInteger); ok {
			value = AccessFloat(item)
		}

		if item, ok := value.(AccessFloat); ok {
			number := float64(item)
			if floatInRange(number, schema.Minimum, schema.Maximum) &&
				(len(schema.Enum) == 0 || slices.ContainsFunc(schema.Enum, func(option FloatEnumOption) bool { return option.Value == number })) {
				return item, nil
			}
		}
	case DateTimeSchema:
		if item, ok := value.(AccessString); ok {
			instant, err := time.Parse(time.RFC3339, string(item))
			if err != nil {
				return nil, apperror.Field("value", "Value must be a valid RFC 3339 datetime.", nil)
			}

			if !timeInRange(instant, schema.Minimum, schema.Maximum) {
				var bounds []string
				if schema.Minimum != nil {
					bounds = append(bounds, "on or after "+schema.Minimum.UTC().Format(time.RFC3339Nano))
				}
				if schema.Maximum != nil {
					bounds = append(bounds, "on or before "+schema.Maximum.UTC().Format(time.RFC3339Nano))
				}

				return nil, apperror.Field("value", "Value must be "+strings.Join(bounds, " and ")+".", nil)
			}

			layout := time.RFC3339
			if schema.Layout != nil {
				layout = strings.NewReplacer("YYYY", "2006", "MM", "01", "DD", "02", "hh", "15", "mm", "04", "ss", "05").Replace(*schema.Layout)
			}
			return AccessString(instant.UTC().Format(layout)), nil
		}
	default:
		return nil, apperror.Default(apperror.CodeInternal)
	}

	return nil, apperror.Field("value", "Value must match parameter type and constraints.", nil)
}
