package domain

import (
	"cmp"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
)

var (
	pathPlaceholderPattern       = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_.-]*)\}`)
	templateReferencePattern     = regexp.MustCompile(`\{\{[^{}]*\.\s*([A-Za-z_][A-Za-z0-9_]*)`)
	templateParameterNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

func NewWebServiceInput(value WebServiceInput) (WebServiceInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	description, err := optionalWebServiceDescription(value.Description)
	if err != nil {
		return WebServiceInput{}, err
	}
	value.Description = description
	value.TemplateURL = strings.TrimSpace(value.TemplateURL)
	value.Format = strings.TrimSpace(value.Format)
	value.Method = HTTPMethod(strings.ToUpper(strings.TrimSpace(string(value.Method))))
	value.Type = WebServiceType(strings.ToUpper(strings.TrimSpace(string(value.Type))))
	if value.Name == "" || value.TemplateURL == "" || value.Format == "" {
		return WebServiceInput{}, fmt.Errorf("name, template URL, and format are required: %w", ErrInvalidDataset)
	}
	if !value.Method.Valid() {
		return WebServiceInput{}, fmt.Errorf("web service method is invalid: %w", ErrInvalidDataset)
	}
	if !value.Type.Valid() {
		return WebServiceInput{}, fmt.Errorf("web service type is invalid: %w", ErrInvalidDataset)
	}
	visibleOn, err := normalizeWebServiceVisibleOn(value.VisibleOn)
	if err != nil {
		return WebServiceInput{}, fmt.Errorf("web service visible on: %w", err)
	}
	value.VisibleOn = visibleOn
	if err := validateTemporal(value.TemporalCoverage); err != nil {
		return WebServiceInput{}, fmt.Errorf("web service temporal coverage: %w", err)
	}
	if value.ProviderID != nil && *value.ProviderID == uuid.Nil {
		return WebServiceInput{}, fmt.Errorf("web service provider ID is required: %w", ErrInvalidDataset)
	}
	if err := validateUUIDList(value.ContactPointIDs, "web service contact point"); err != nil {
		return WebServiceInput{}, fmt.Errorf("web service contact points: %w: %w", err, ErrInvalidDataset)
	}
	if value.Documentation != nil {
		value.Documentation.Title = optionalTrimmedString(value.Documentation.Title)
		value.Documentation.Description = optionalTrimmedString(value.Documentation.Description)
		value.Documentation.URI = strings.TrimSpace(value.Documentation.URI)
		if value.Documentation.URI == "" {
			return WebServiceInput{}, fmt.Errorf("documentation URI is required: %w", ErrInvalidDataset)
		}
	}

	parameters, err := parseWebServiceParameterInputs(value.Parameters)
	if err != nil {
		return WebServiceInput{}, fmt.Errorf("web service parameters: %w", err)
	}
	value.Parameters = parameters
	if value.Body != nil {
		body, err := NewRequestBody(*value.Body)
		if err != nil {
			return WebServiceInput{}, fmt.Errorf("web service body: %w", err)
		}
		value.Body = &body
	}
	if err := validateMethodBody(value.Method, value.Body); err != nil {
		return WebServiceInput{}, fmt.Errorf("web service body: %w", err)
	}
	if err := validatePathParameters(value.TemplateURL, pathParameterNames(parameters)); err != nil {
		return WebServiceInput{}, fmt.Errorf("web service path parameters: %w", err)
	}
	return value, nil
}

// NewWebService parses and validates persisted web-service metadata.
func NewWebService(value WebService) (WebService, error) {
	if value.ID == uuid.Nil {
		return WebService{}, fmt.Errorf("web service ID is required: %w", ErrInvalidDataset)
	}
	parameterIDs := make(map[string]uuid.UUID, len(value.Parameters))
	for _, parameter := range value.Parameters {
		if parameter.ID == uuid.Nil {
			return WebService{}, fmt.Errorf("parameter ID is required: %w", ErrInvalidDataset)
		}
		parameterIDs[strings.TrimSpace(parameter.Name)] = parameter.ID
	}

	input, err := NewWebServiceInput(value.Input())
	if err != nil {
		return WebService{}, err
	}
	value.Name = input.Name
	value.Description = input.Description
	value.TemplateURL = input.TemplateURL
	value.Method = input.Method
	value.Type = input.Type
	value.VisibleOn = input.VisibleOn
	value.Body = input.Body
	value.Parameters = make([]WebServiceParameter, len(input.Parameters))
	for i, parameter := range input.Parameters {
		value.Parameters[i] = WebServiceParameter{
			ID:        parameterIDs[parameter.Name],
			Parameter: parameter.Parameter,
			Property:  parameter.Property,
			Location:  parameter.Location,
			Explode:   parameter.Explode,
		}
	}
	if value.Provider != nil {
		if err := validateOrganization(*value.Provider); err != nil {
			return WebService{}, fmt.Errorf("provider: %w: %w", err, ErrInvalidDataset)
		}
	}
	if err := validateContactPoints(value.ContactPoints, "web service contact point"); err != nil {
		return WebService{}, fmt.Errorf("contact points: %w: %w", err, ErrInvalidDataset)
	}
	return value, nil
}

func normalizeWebServiceVisibleOn(values []WebServiceVisibleOn) ([]WebServiceVisibleOn, error) {
	if values == nil {
		return nil, nil
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("visible on must contain at least one value: %w", ErrInvalidDataset)
	}

	result := make([]WebServiceVisibleOn, 0, len(values))
	seen := make(map[WebServiceVisibleOn]struct{}, len(values))
	for _, value := range values {
		value = WebServiceVisibleOn(strings.ToUpper(strings.TrimSpace(string(value))))
		if !value.Valid() {
			return nil, fmt.Errorf("visible on value %q is invalid: %w", value, ErrInvalidDataset)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

// NewWebServiceParameterInput parses a caller-controlled top-level parameter.
func NewWebServiceParameterInput(value WebServiceParameterInput) (WebServiceParameterInput, error) {
	parameter, err := parseParameter(value.Parameter, true, true)
	if err != nil {
		return WebServiceParameterInput{}, err
	}
	value.Parameter = parameter
	value.Property = optionalTrimmedString(value.Property)
	value.Location = ParameterLocation(strings.ToUpper(strings.TrimSpace(string(value.Location))))
	if !value.Location.Valid() {
		return WebServiceParameterInput{}, fmt.Errorf("parameter location is invalid: %w", ErrInvalidDataset)
	}
	if value.Explode && (value.Location != ParameterLocationQuery || !schemaHasMultipleValues(value.Schema)) {
		return WebServiceParameterInput{}, fmt.Errorf("parameter explode requires query location and multiple values: %w", ErrInvalidDataset)
	}
	return value, nil
}

// NewWebServiceParameter parses persisted top-level parameter metadata.
func NewWebServiceParameter(value WebServiceParameter) (WebServiceParameter, error) {
	if value.ID == uuid.Nil {
		return WebServiceParameter{}, fmt.Errorf("parameter ID is required: %w", ErrInvalidDataset)
	}
	parsed, err := NewWebServiceParameterInput(WebServiceParameterInput{
		Parameter: value.Parameter,
		Property:  value.Property,
		Location:  value.Location,
		Explode:   value.Explode,
	})
	if err != nil {
		return WebServiceParameter{}, err
	}
	value.Parameter = parsed.Parameter
	value.Property = parsed.Property
	value.Location = parsed.Location
	value.Explode = parsed.Explode
	return value, nil
}

// NewRequestBody parses and validates an optional request-body definition.
func NewRequestBody(value RequestBody) (RequestBody, error) {
	value.ContentType = strings.TrimSpace(value.ContentType)
	value.Template = strings.TrimSpace(value.Template)
	if value.ContentType == "" {
		return RequestBody{}, fmt.Errorf("body content type and template are required: %w", ErrInvalidDataset)
	}
	templateValue, parameters, err := parseTemplateDefinition(value.Template, value.Parameters)
	if err != nil {
		return RequestBody{}, fmt.Errorf("body template: %w", err)
	}
	value.Template = templateValue
	value.Parameters = parameters
	return value, nil
}

func parseTemplateDefinition(source string, values []Parameter) (string, []Parameter, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", nil, fmt.Errorf("template is required: %w", ErrInvalidDataset)
	}
	if _, err := template.New("parameter").Option("missingkey=error").Parse(source); err != nil {
		return "", nil, fmt.Errorf("parse template: %w: %w", err, ErrInvalidDataset)
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

func parseWebServiceParameterInputs(values []WebServiceParameterInput) ([]WebServiceParameterInput, error) {
	result := make([]WebServiceParameterInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		parsed, err := NewWebServiceParameterInput(value)
		if err != nil {
			return nil, fmt.Errorf("parse parameter: %w", err)
		}
		if _, exists := seen[parsed.Name]; exists {
			return nil, fmt.Errorf("duplicate parameter name %q: %w", parsed.Name, ErrInvalidDataset)
		}
		seen[parsed.Name] = struct{}{}
		result = append(result, parsed)
	}
	slices.SortFunc(result, func(a, b WebServiceParameterInput) int {
		if order := cmp.Compare(a.Position, b.Position); order != 0 {
			return order
		}
		return strings.Compare(a.Name, b.Name)
	})
	return result, nil
}

func webServiceParameterInputs(values []WebServiceParameter) []WebServiceParameterInput {
	result := make([]WebServiceParameterInput, len(values))
	for i, value := range values {
		result[i] = WebServiceParameterInput{
			Parameter: value.Parameter,
			Property:  value.Property,
			Location:  value.Location,
			Explode:   value.Explode,
		}
	}
	return result
}

func parseParameter(value Parameter, allowTemplate, allowMultipleValues bool) (Parameter, error) {
	value.Name = strings.TrimSpace(value.Name)
	value.Label = strings.TrimSpace(value.Label)
	value.Description = optionalTrimmedString(value.Description)
	if value.Name == "" || value.Label == "" {
		return Parameter{}, fmt.Errorf("parameter name and label are required: %w", ErrInvalidDataset)
	}
	if value.Schema == nil {
		return Parameter{}, fmt.Errorf("parameter schema is required: %w", ErrInvalidDataset)
	}
	schema, err := parseParameterSchema(value.Schema, allowTemplate)
	if err != nil {
		return Parameter{}, err
	}
	if !allowMultipleValues && schemaHasMultipleValues(schema) {
		return Parameter{}, fmt.Errorf("multiple values are not allowed for scalar parameters: %w", ErrInvalidDataset)
	}
	if schemaHasMultipleValues(schema) && !parameterSchemaHasEnum(schema) {
		return Parameter{}, fmt.Errorf("multiple values require a non-empty enum schema: %w", ErrInvalidDataset)
	}
	value.Schema = schema
	return value, nil
}

func parseParameters(values []Parameter) ([]Parameter, error) {
	result := make([]Parameter, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		parsed, err := parseParameter(value, false, false)
		if err != nil {
			return nil, fmt.Errorf("parse parameter: %w", err)
		}
		if !templateParameterNamePattern.MatchString(parsed.Name) {
			return nil, fmt.Errorf("nested parameter name %q is invalid: %w", parsed.Name, ErrInvalidDataset)
		}
		if _, exists := seen[parsed.Name]; exists {
			return nil, fmt.Errorf("duplicate parameter name %q: %w", parsed.Name, ErrInvalidDataset)
		}
		seen[parsed.Name] = struct{}{}
		result = append(result, parsed)
	}
	slices.SortFunc(result, func(a, b Parameter) int {
		if order := cmp.Compare(a.Position, b.Position); order != 0 {
			return order
		}
		return strings.Compare(a.Name, b.Name)
	})
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
	case TemplateSchema:
		if !allowTemplate {
			return nil, fmt.Errorf("template schema is not allowed here: %w", ErrInvalidDataset)
		}
		return parseTemplateSchema(schema)
	default:
		return nil, fmt.Errorf("unsupported parameter schema %T: %w", value, ErrInvalidDataset)
	}
}

func parameterSchemaHasEnum(value ParameterSchema) bool {
	switch schema := value.(type) {
	case StringSchema:
		return len(schema.Enum) > 0
	case IntegerSchema:
		return len(schema.Enum) > 0
	case FloatSchema:
		return len(schema.Enum) > 0
	default:
		return false
	}
}

func schemaHasMultipleValues(value ParameterSchema) bool {
	switch schema := value.(type) {
	case StringSchema:
		return schema.MultipleValues
	case IntegerSchema:
		return schema.MultipleValues
	case FloatSchema:
		return schema.MultipleValues
	default:
		return false
	}
}

func parseStringSchema(value StringSchema) (StringSchema, error) {
	value.Pattern = optionalTrimmedString(value.Pattern)
	if value.Pattern != nil {
		if _, err := regexp.Compile(*value.Pattern); err != nil {
			return StringSchema{}, fmt.Errorf("invalid string pattern: %w: %w", err, ErrInvalidDataset)
		}
	}
	if value.Enum != nil && len(value.Enum) == 0 {
		return StringSchema{}, fmt.Errorf("string enum cannot be empty: %w", ErrInvalidDataset)
	}
	seen := make(map[string]struct{}, len(value.Enum))
	for i := range value.Enum {
		item := &value.Enum[i]
		if item.Value == "" {
			return StringSchema{}, fmt.Errorf("string enum values cannot be empty: %w", ErrInvalidDataset)
		}
		if _, exists := seen[item.Value]; exists {
			return StringSchema{}, fmt.Errorf("string enum value %q is duplicated: %w", item.Value, ErrInvalidDataset)
		}
		seen[item.Value] = struct{}{}
		if item.Label != nil {
			label := strings.TrimSpace(*item.Label)
			if label == "" {
				return StringSchema{}, fmt.Errorf("string enum label cannot be empty: %w", ErrInvalidDataset)
			}
			item.Label = &label
		}
		if value.Pattern != nil && !regexp.MustCompile(*value.Pattern).MatchString(item.Value) {
			return StringSchema{}, fmt.Errorf("string enum value %q does not match pattern: %w", item.Value, ErrInvalidDataset)
		}
	}
	if value.Default != nil {
		if strings.TrimSpace(*value.Default) == "" {
			return StringSchema{}, fmt.Errorf("string default cannot be empty: %w", ErrInvalidDataset)
		}
		if value.Pattern != nil && !regexp.MustCompile(*value.Pattern).MatchString(*value.Default) {
			return StringSchema{}, fmt.Errorf("string default does not match pattern: %w", ErrInvalidDataset)
		}
		if value.Enum != nil && !slices.ContainsFunc(value.Enum, func(item StringEnumOption) bool {
			return item.Value == *value.Default
		}) {
			return StringSchema{}, fmt.Errorf("string default is not an allowed value: %w", ErrInvalidDataset)
		}
	}
	return value, nil
}

func parseIntegerSchema(value IntegerSchema) (IntegerSchema, error) {
	if value.Enum != nil && len(value.Enum) == 0 {
		return IntegerSchema{}, fmt.Errorf("integer enum cannot be empty: %w", ErrInvalidDataset)
	}
	if value.Minimum != nil && value.Maximum != nil && *value.Minimum > *value.Maximum {
		return IntegerSchema{}, fmt.Errorf("integer minimum must not exceed maximum: %w", ErrInvalidDataset)
	}
	seen := make(map[int64]struct{}, len(value.Enum))
	for i := range value.Enum {
		item := &value.Enum[i]
		if _, exists := seen[item.Value]; exists {
			return IntegerSchema{}, fmt.Errorf("integer enum value %d is duplicated: %w", item.Value, ErrInvalidDataset)
		}
		seen[item.Value] = struct{}{}
		if item.Label != nil {
			label := strings.TrimSpace(*item.Label)
			if label == "" {
				return IntegerSchema{}, fmt.Errorf("integer enum label cannot be empty: %w", ErrInvalidDataset)
			}
			item.Label = &label
		}
		if !integerInRange(item.Value, value.Minimum, value.Maximum) {
			return IntegerSchema{}, fmt.Errorf("integer enum value is outside allowed range: %w", ErrInvalidDataset)
		}
	}
	if value.Default != nil {
		if !integerInRange(*value.Default, value.Minimum, value.Maximum) {
			return IntegerSchema{}, fmt.Errorf("integer default is outside allowed range: %w", ErrInvalidDataset)
		}
		if value.Enum != nil && !slices.ContainsFunc(value.Enum, func(item IntegerEnumOption) bool {
			return item.Value == *value.Default
		}) {
			return IntegerSchema{}, fmt.Errorf("integer default is not an allowed value: %w", ErrInvalidDataset)
		}
	}
	return value, nil
}

func parseFloatSchema(value FloatSchema) (FloatSchema, error) {
	if value.Enum != nil && len(value.Enum) == 0 {
		return FloatSchema{}, fmt.Errorf("float enum cannot be empty: %w", ErrInvalidDataset)
	}
	if !finiteFloat(value.Minimum) || !finiteFloat(value.Maximum) || !finiteFloat(value.Default) {
		return FloatSchema{}, fmt.Errorf("float values must be finite: %w", ErrInvalidDataset)
	}
	seen := make(map[float64]struct{}, len(value.Enum))
	for i := range value.Enum {
		item := &value.Enum[i]
		if _, exists := seen[item.Value]; exists {
			return FloatSchema{}, fmt.Errorf("float enum value %v is duplicated: %w", item.Value, ErrInvalidDataset)
		}
		seen[item.Value] = struct{}{}
		if item.Label != nil {
			label := strings.TrimSpace(*item.Label)
			if label == "" {
				return FloatSchema{}, fmt.Errorf("float enum label cannot be empty: %w", ErrInvalidDataset)
			}
			item.Label = &label
		}
		if math.IsNaN(item.Value) || math.IsInf(item.Value, 0) || !floatInRange(item.Value, value.Minimum, value.Maximum) {
			return FloatSchema{}, fmt.Errorf("float enum value is invalid or outside allowed range: %w", ErrInvalidDataset)
		}
	}
	if value.Minimum != nil && value.Maximum != nil && *value.Minimum > *value.Maximum {
		return FloatSchema{}, fmt.Errorf("float minimum must not exceed maximum: %w", ErrInvalidDataset)
	}
	if value.Default != nil {
		if !floatInRange(*value.Default, value.Minimum, value.Maximum) {
			return FloatSchema{}, fmt.Errorf("float default is outside allowed range: %w", ErrInvalidDataset)
		}
		if value.Enum != nil && !slices.ContainsFunc(value.Enum, func(item FloatEnumOption) bool {
			return item.Value == *value.Default
		}) {
			return FloatSchema{}, fmt.Errorf("float default is not an allowed value: %w", ErrInvalidDataset)
		}
	}
	return value, nil
}

func parseDateTimeSchema(value DateTimeSchema) (DateTimeSchema, error) {
	if value.Layout != nil {
		layout := strings.TrimSpace(*value.Layout)
		value.Layout = &layout
		if err := validateDateTimeLayout(layout); err != nil {
			return DateTimeSchema{}, err
		}
	}
	if value.Minimum != nil && value.Maximum != nil && value.Minimum.After(*value.Maximum) {
		return DateTimeSchema{}, fmt.Errorf("date-time minimum must not be after maximum: %w", ErrInvalidDataset)
	}
	if value.Default != nil && !timeInRange(*value.Default, value.Minimum, value.Maximum) {
		return DateTimeSchema{}, fmt.Errorf("date-time default is outside allowed range: %w", ErrInvalidDataset)
	}
	return value, nil
}

func parseTemplateSchema(value TemplateSchema) (TemplateSchema, error) {
	templateValue, parameters, err := parseTemplateDefinition(value.Template, value.Parameters)
	if err != nil {
		return TemplateSchema{}, err
	}
	value.Template = templateValue
	value.Parameters = parameters
	return value, nil
}

func validateTemplateReferences(source string, parameters []Parameter) error {
	allowed := make(map[string]struct{}, len(parameters))
	for _, parameter := range parameters {
		allowed[parameter.Name] = struct{}{}
	}
	for _, match := range templateReferencePattern.FindAllStringSubmatch(source, -1) {
		if _, exists := allowed[match[1]]; !exists {
			return fmt.Errorf("template references undeclared parameter %q: %w", match[1], ErrInvalidDataset)
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
			return fmt.Errorf("path parameter %q has no URL placeholder: %w", name, ErrInvalidDataset)
		}
		delete(placeholders, name)
	}
	if len(placeholders) != 0 {
		for name := range placeholders {
			return fmt.Errorf("URL placeholder %q has no path parameter: %w", name, ErrInvalidDataset)
		}
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

func validateMethodBody(method HTTPMethod, body *RequestBody) error {
	if body != nil && (method == HTTPMethodGet || method == HTTPMethodDelete) {
		return fmt.Errorf("body is not allowed for %s: %w", method, ErrInvalidDataset)
	}
	return nil
}

func validateDateTimeLayout(layout string) error {
	if strings.TrimSpace(layout) == "" {
		return fmt.Errorf("date-time layout cannot be empty: %w", ErrInvalidDataset)
	}
	tokens := []struct {
		name    string
		goValue string
	}{
		{"YYYY", "2006"},
		{"MM", "01"},
		{"DD", "02"},
		{"hh", "15"},
		{"mm", "04"},
		{"ss", "05"},
	}
	for index := 0; index < len(layout); {
		matched := false
		for _, token := range tokens {
			if strings.HasPrefix(layout[index:], token.name) {
				index += len(token.name)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		if strings.ContainsRune("YyMDHhms", rune(layout[index])) {
			return fmt.Errorf("unsupported date-time layout token near %q: %w", layout[index:], ErrInvalidDataset)
		}
		index++
	}
	return nil
}

func optionalTrimmedString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalWebServiceDescription(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, fmt.Errorf("web service description cannot be empty: %w", ErrInvalidDataset)
	}
	return &trimmed, nil
}

func finiteFloat(value *float64) bool {
	return value == nil || (!math.IsNaN(*value) && !math.IsInf(*value, 0))
}

func integerInRange(value int64, minimum, maximum *int64) bool {
	return (minimum == nil || value >= *minimum) && (maximum == nil || value <= *maximum)
}

func floatInRange(value float64, minimum, maximum *float64) bool {
	return (minimum == nil || value >= *minimum) && (maximum == nil || value <= *maximum)
}

func timeInRange(value time.Time, minimum, maximum *time.Time) bool {
	return (minimum == nil || !value.Before(*minimum)) && (maximum == nil || !value.After(*maximum))
}
