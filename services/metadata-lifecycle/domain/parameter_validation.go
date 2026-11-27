package domain

import (
	"errors"
	"fmt"
	"math"
	"net/textproto"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
	"golang.org/x/net/http/httpguts"
)

var (
	pathPlaceholderPattern       = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_.-]*)\}`)
	templateReferencePattern     = regexp.MustCompile(`\{\{[^{}]*\.\s*([A-Za-z_][A-Za-z0-9_]*)`)
	templateParameterNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

func reservedParameterNames(names ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(names))
	for _, name := range names {
		result[name] = struct{}{}
	}
	return result
}

func normalizeWebServiceVisibleOn(values []WebServiceVisibleOn) ([]WebServiceVisibleOn, error) {
	// TODO: add normalie function to the type
	if len(values) == 0 {
		return nil, nil
	}

	result := make([]WebServiceVisibleOn, 0, len(values))
	seen := make(map[WebServiceVisibleOn]struct{}, len(values))
	for _, value := range values {
		value = WebServiceVisibleOn(strings.ToUpper(strings.TrimSpace(string(value))))
		if !value.Valid() {
			return nil, fmt.Errorf("visible on value %q is invalid: %w", value, apperror.Default(apperror.CodeValidationFailed))
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func parseTemplateDefinition(source string, values []ParameterInput) (string, []ParameterInput, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", nil, fmt.Errorf("template is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if _, err := template.New("parameter").Option("missingkey=error").Parse(source); err != nil {
		return "", nil, fmt.Errorf("parse template: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	parameters, err := parseParameters(values)
	if err != nil {
		return "", nil, fmt.Errorf("template parameters: %w", err)
	}
	if err := validateTemplateReferences(source, parameters); err != nil {
		return "", nil, fmt.Errorf("template references: %w", err)
	}
	return source, parameters, nil
}

func parseWebServiceParameterInputs(values []WebServiceParameterInput, field string) ([]WebServiceParameterInput, error) {
	result := make([]WebServiceParameterInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	seenHeaders := make(map[string]struct{})
	for i, value := range values {
		parsed := value
		if err := parsed.Normalize(); err != nil {
			return nil, fmt.Errorf("validate web service parameter: %w", apperror.PrefixTarget(err, fmt.Sprintf("%s[%d]", field, i)))
		}
		if _, exists := seen[parsed.Name]; exists {
			return nil, apperror.Field(fmt.Sprintf("%s[%d].name", field, i), "Parameter names must be unique.", nil)
		}
		if parsed.Location == ParameterLocationHeader {
			name := textproto.CanonicalMIMEHeaderKey(parsed.Name)
			if _, exists := seenHeaders[name]; exists {
				return nil, apperror.Field(fmt.Sprintf("%s[%d].name", field, i), "Header names must be unique ignoring case.", nil)
			}
			seenHeaders[name] = struct{}{}
		}
		seen[parsed.Name] = struct{}{}
		result = append(result, parsed)
	}
	return result, nil
}

func parseParameter(value ParameterInput, allowTemplate, allowMultipleValues bool) (ParameterInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	value.Label = strings.TrimSpace(value.Label)
	value.Description = optionalTrimmedString(value.Description)
	if value.Role != nil {
		role := WebServiceParameterRole(strings.ToUpper(strings.TrimSpace(string(*value.Role))))
		if !role.Valid() {
			return ParameterInput{}, apperror.Field("role", "Role is not supported.", nil)
		}
		value.Role = &role
	}
	if value.Name == "" {
		return ParameterInput{}, apperror.Field("name", "Name must not be blank.", nil)
	}
	if value.Label == "" {
		return ParameterInput{}, apperror.Field("label", "Label must not be blank.", nil)
	}
	if value.Schema == nil {
		return ParameterInput{}, apperror.Field("kind", "Parameter kind is required.", nil)
	}
	schema, err := parseParameterSchema(value.Schema, allowTemplate)
	if err != nil {
		return ParameterInput{}, err
	}
	if !allowMultipleValues && schema.AllowsMultipleValues() {
		return ParameterInput{}, apperror.Field("multiple_values", "Multiple values are not allowed here.", nil)
	}
	value.Schema = schema
	return value, nil
}

func validateRequiredParameterDefault(parameter ParameterInput) error {
	if schema, ok := parameter.Schema.(TemplateSchemaInput); ok {
		return validateRequiredParameterDefaults("parameters", schema.Parameters)
	}

	if parameter.Required && parameter.Schema.DefaultValue() == nil {
		return apperror.Field("default", "Required parameters must have a default.", nil)
	}

	return nil
}

func validateRequiredParameterDefaults(field string, parameters []ParameterInput) error {
	for i, parameter := range parameters {
		if err := validateRequiredParameterDefault(parameter); err != nil {
			return apperror.PrefixTarget(err, fmt.Sprintf("%s[%d]", field, i))
		}
	}

	return nil
}

// validateTemplateDefaults checks required defaults and renders active templates
// without caller-supplied values, reporting errors against metadata fields.
func validateTemplateDefaults(field string, parameters []WebServiceParameterInput, body *RequestBodyInput) error {
	for i, parameter := range parameters {
		if err := validateRequiredParameterDefault(parameter.ParameterInput); err != nil {
			return apperror.PrefixTarget(err, fmt.Sprintf("%s[%d]", field, i))
		}

		if err := parameter.validateHeaderDefault(); err != nil {
			return apperror.PrefixTarget(err, fmt.Sprintf("%s[%d]", field, i))
		}
	}

	if body != nil {
		if err := validateRequiredParameterDefaults("body_parameters", body.Parameters); err != nil {
			return err
		}
	}

	for i, parameter := range parameters {
		schema, ok := parameter.Schema.(TemplateSchemaInput)
		if !ok {
			continue
		}

		target := fmt.Sprintf("%s[%d].template", field, i)

		rendered, err := validateTemplateDefault(target, schema.Template, schema.Parameters, parameter.Required)
		if err != nil {
			return err
		}

		if rendered != nil && parameter.Location == ParameterLocationHeader && !httpguts.ValidHeaderFieldValue(*rendered) {
			return apperror.Field(target, "Template defaults produce an invalid HTTP header value.", nil)
		}
	}

	if body != nil {
		_, err := validateTemplateDefault("body_template", body.Template, body.Parameters, *body.Required)
		return err
	}

	return nil
}

func validateTemplateDefault(target, source string, inputs []ParameterInput, required bool) (*string, error) {
	parameters := renderParameterInputs(inputs)

	index := make(map[uuid.UUID]Parameter, len(parameters))
	for _, parameter := range parameters {
		index[parameter.ID] = parameter
	}

	values := accessValues{
		resolved: make(map[uuid.UUID]AccessArray),
		supplied: make(map[uuid.UUID]bool),
	}

	if err := values.resolveDefaults(index); err != nil {
		return nil, apperror.Field(target, "Template has an invalid input default.", err)
	}

	rendered, err := values.renderTemplate(source, parameters, required)
	if err == nil {
		return rendered, nil
	}

	detail := "Template could not be rendered with defaults. Check the template expressions and input defaults."
	if execution, ok := errors.AsType[template.ExecError](err); ok {
		// return the actual template error so that the user can debug the template
		detail = execution.Error()
	}

	return nil, apperror.Field(target, detail, err)
}

func parseParameters(values []ParameterInput) ([]ParameterInput, error) {
	result := make([]ParameterInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for i, value := range values {
		parsed, err := parseParameter(value, false, false)
		if err != nil {
			return nil, fmt.Errorf("validate nested parameter: %w", apperror.PrefixTarget(err, fmt.Sprintf("[%d]", i)))
		}
		if !templateParameterNamePattern.MatchString(parsed.Name) {
			return nil, apperror.Field(fmt.Sprintf("[%d].name", i), "Name must start with a letter or underscore and contain only letters, numbers, and underscores.", nil)
		}
		if _, exists := seen[parsed.Name]; exists {
			return nil, apperror.Field(fmt.Sprintf("[%d].name", i), "Parameter names must be unique.", nil)
		}
		seen[parsed.Name] = struct{}{}
		result = append(result, parsed)
	}
	return result, nil
}

func parseParameterSchema(value ParameterSchema, allowTemplate bool) (ParameterSchema, error) {
	switch schema := value.(type) {
	case StringSchema:
		return parseStringSchema(schema)
	case BooleanSchema:
		return schema, nil
	case IntegerSchema:
		return parseIntegerSchema(schema)
	case FloatSchema:
		return parseFloatSchema(schema)
	case DateTimeSchema:
		return parseDateTimeSchema(schema)
	case TemplateSchemaInput:
		if !allowTemplate {
			return nil, apperror.Field("kind", "Template parameters are not allowed here.", nil)
		}
		return parseTemplateSchema(schema)
	default:
		return nil, apperror.Field("kind", "Parameter kind is not supported.", nil)
	}
}

func parseStringSchema(value StringSchema) (StringSchema, error) {
	value.Pattern = optionalTrimmedString(value.Pattern)
	if value.Pattern != nil {
		if _, err := regexp.Compile(*value.Pattern); err != nil {
			return StringSchema{}, apperror.Field("pattern", "Pattern must be a valid regular expression.", err)
		}
	}
	if len(value.Enum) == 0 {
		value.Enum = nil
	}
	if len(value.Default) == 0 {
		value.Default = nil
	}
	seen := make(map[string]struct{}, len(value.Enum))
	for i := range value.Enum {
		item := &value.Enum[i]
		if item.Value == "" {
			return StringSchema{}, apperror.Field(fmt.Sprintf("enum[%d].value", i), "Enum value must not be empty.", nil)
		}
		if _, exists := seen[item.Value]; exists {
			return StringSchema{}, apperror.Field(fmt.Sprintf("enum[%d].value", i), "Enum values must be unique.", nil)
		}
		seen[item.Value] = struct{}{}
		if item.Label != nil {
			label := strings.TrimSpace(*item.Label)
			if label == "" {
				return StringSchema{}, apperror.Field(fmt.Sprintf("enum[%d].label", i), "Enum label must not be blank.", nil)
			}
			item.Label = &label
		}
		if value.Pattern != nil && !regexp.MustCompile(*value.Pattern).MatchString(item.Value) {
			return StringSchema{}, apperror.Field(fmt.Sprintf("enum[%d].value", i), "Enum value must match the pattern.", nil)
		}
	}
	if value.Default != nil {
		if !value.MultipleValues && len(value.Default) > 1 {
			return StringSchema{}, apperror.Field("default", "Single-value parameters allow at most one default.", nil)
		}
		for _, defaultValue := range value.Default {
			if strings.TrimSpace(defaultValue) == "" {
				return StringSchema{}, apperror.Field("default", "Default must not be blank.", nil)
			}
			if value.Pattern != nil && !regexp.MustCompile(*value.Pattern).MatchString(defaultValue) {
				return StringSchema{}, apperror.Field("default", "Default must match the pattern.", nil)
			}
			if value.Enum != nil && !slices.ContainsFunc(value.Enum, func(item StringEnumOption) bool {
				return item.Value == defaultValue
			}) {
				return StringSchema{}, apperror.Field("default", "Default must be an allowed value.", nil)
			}
		}
	}
	return value, nil
}

func parseIntegerSchema(value IntegerSchema) (IntegerSchema, error) {
	if len(value.Enum) == 0 {
		value.Enum = nil
	}
	if len(value.Default) == 0 {
		value.Default = nil
	}
	if value.Minimum != nil && value.Maximum != nil && *value.Minimum > *value.Maximum {
		return IntegerSchema{}, apperror.Field("minimum", "Minimum must not exceed maximum.", nil)
	}
	seen := make(map[int64]struct{}, len(value.Enum))
	for i := range value.Enum {
		item := &value.Enum[i]
		if _, exists := seen[item.Value]; exists {
			return IntegerSchema{}, apperror.Field(fmt.Sprintf("enum[%d].value", i), "Enum values must be unique.", nil)
		}
		seen[item.Value] = struct{}{}
		if item.Label != nil {
			label := strings.TrimSpace(*item.Label)
			if label == "" {
				return IntegerSchema{}, apperror.Field(fmt.Sprintf("enum[%d].label", i), "Enum label must not be blank.", nil)
			}
			item.Label = &label
		}
		if !integerInRange(item.Value, value.Minimum, value.Maximum) {
			return IntegerSchema{}, apperror.Field(fmt.Sprintf("enum[%d].value", i), "Enum value must be within the allowed range.", nil)
		}
	}
	if value.Default != nil {
		if !value.MultipleValues && len(value.Default) > 1 {
			return IntegerSchema{}, apperror.Field("default", "Single-value parameters allow at most one default.", nil)
		}
		for _, defaultValue := range value.Default {
			if !integerInRange(defaultValue, value.Minimum, value.Maximum) {
				return IntegerSchema{}, apperror.Field("default", "Default must be within the allowed range.", nil)
			}
			if value.Enum != nil && !slices.ContainsFunc(value.Enum, func(item IntegerEnumOption) bool {
				return item.Value == defaultValue
			}) {
				return IntegerSchema{}, apperror.Field("default", "Default must be an allowed value.", nil)
			}
		}
	}
	return value, nil
}

func parseFloatSchema(value FloatSchema) (FloatSchema, error) {
	if len(value.Enum) == 0 {
		value.Enum = nil
	}
	if len(value.Default) == 0 {
		value.Default = nil
	}
	seen := make(map[float64]struct{}, len(value.Enum))
	for i := range value.Enum {
		item := &value.Enum[i]
		if _, exists := seen[item.Value]; exists {
			return FloatSchema{}, apperror.Field(fmt.Sprintf("enum[%d].value", i), "Enum values must be unique.", nil)
		}
		seen[item.Value] = struct{}{}
		if item.Label != nil {
			label := strings.TrimSpace(*item.Label)
			if label == "" {
				return FloatSchema{}, apperror.Field(fmt.Sprintf("enum[%d].label", i), "Enum label must not be blank.", nil)
			}
			item.Label = &label
		}
		if math.IsNaN(item.Value) || math.IsInf(item.Value, 0) || !floatInRange(item.Value, value.Minimum, value.Maximum) {
			return FloatSchema{}, apperror.Field(fmt.Sprintf("enum[%d].value", i), "Enum value must be finite and within the allowed range.", nil)
		}
	}
	if value.Minimum != nil && value.Maximum != nil && *value.Minimum > *value.Maximum {
		return FloatSchema{}, apperror.Field("minimum", "Minimum must not exceed maximum.", nil)
	}
	if value.Default != nil {
		if !value.MultipleValues && len(value.Default) > 1 {
			return FloatSchema{}, apperror.Field("default", "Single-value parameters allow at most one default.", nil)
		}
		for _, defaultValue := range value.Default {
			if math.IsNaN(defaultValue) || math.IsInf(defaultValue, 0) || !floatInRange(defaultValue, value.Minimum, value.Maximum) {
				return FloatSchema{}, apperror.Field("default", "Default must be finite and within the allowed range.", nil)
			}
			if value.Enum != nil && !slices.ContainsFunc(value.Enum, func(item FloatEnumOption) bool {
				return item.Value == defaultValue
			}) {
				return FloatSchema{}, apperror.Field("default", "Default must be an allowed value.", nil)
			}
		}
	}
	return value, nil
}

func parseDateTimeSchema(value DateTimeSchema) (DateTimeSchema, error) {
	if value.Layout != nil {
		layout := strings.TrimSpace(*value.Layout)
		value.Layout = &layout
		if err := validateDateTimeLayout(layout); err != nil {
			return DateTimeSchema{}, apperror.Field("layout", "Layout contains unsupported date-time tokens.", err)
		}
	}
	if value.Minimum != nil && value.Maximum != nil && value.Minimum.After(*value.Maximum) {
		return DateTimeSchema{}, apperror.Field("minimum", "Minimum must not be after maximum.", nil)
	}
	if value.Default != nil && !timeInRange(*value.Default, value.Minimum, value.Maximum) {
		return DateTimeSchema{}, apperror.Field("default", "Default must be within the allowed range.", nil)
	}
	return value, nil
}

func parseTemplateSchema(value TemplateSchemaInput) (TemplateSchemaInput, error) {
	templateValue, parameters, err := parseTemplateDefinition(value.Template, value.Parameters)
	if err != nil {
		return TemplateSchemaInput{}, prefixOrField(err, "parameters", "template", "Template must be valid and reference declared parameters.")
	}
	value.Template = templateValue
	value.Parameters = parameters
	return value, nil
}

func validateTemplateReferences(source string, parameters []ParameterInput) error {
	allowed := make(map[string]struct{}, len(parameters))
	for _, parameter := range parameters {
		allowed[parameter.Name] = struct{}{}
	}
	for _, match := range templateReferencePattern.FindAllStringSubmatch(source, -1) {
		if _, exists := allowed[match[1]]; !exists {
			return fmt.Errorf("template references undeclared parameter %q: %w", match[1], apperror.Default(apperror.CodeValidationFailed))
		}
	}
	return nil
}

func validatePathParameters(templateURL string, pathParameters map[string]struct{}) error {
	placeholders := make(map[string]struct{})
	for _, match := range pathPlaceholderPattern.FindAllStringSubmatch(templateURL, -1) {
		placeholders[match[1]] = struct{}{}
	}
	for name := range pathParameters {
		if _, exists := placeholders[name]; !exists {
			return fmt.Errorf("path parameter %q has no URL placeholder: %w", name, apperror.Default(apperror.CodeValidationFailed))
		}
		delete(placeholders, name)
	}
	for name := range placeholders {
		return fmt.Errorf("URL placeholder %q has no path parameter: %w", name, apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

func pathParameterNames(parameters []WebServiceParameterInput) map[string]struct{} {
	result := make(map[string]struct{})
	for _, parameter := range parameters {
		if parameter.Location == ParameterLocationPath {
			result[parameter.Name] = struct{}{}
		}
	}
	return result
}

func validateMethodBody(method HTTPMethod, body *RequestBodyInput) error {
	if body != nil && (method == HTTPMethodGet || method == HTTPMethodDelete) {
		return fmt.Errorf("body is not allowed for %s: %w", method, apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

func validateDateTimeLayout(layout string) error {
	if strings.TrimSpace(layout) == "" {
		return fmt.Errorf("date-time layout cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	tokens := []string{"YYYY", "MM", "DD", "hh", "mm", "ss"}
	for index := 0; index < len(layout); {
		matched := false
		for _, token := range tokens {
			if strings.HasPrefix(layout[index:], token) {
				index += len(token)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		if strings.ContainsRune("YyMDHhms", rune(layout[index])) {
			return fmt.Errorf("unsupported date-time layout token near %q: %w", layout[index:], apperror.Default(apperror.CodeValidationFailed))
		}
		index++
	}
	return nil
}
