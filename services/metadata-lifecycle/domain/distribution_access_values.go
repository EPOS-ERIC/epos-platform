package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

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
