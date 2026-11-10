package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

func TestNewWebServiceInputNormalizesAndOrdersParameters(t *testing.T) {
	input, err := NewWebServiceInput(WebServiceInput{
		Type: " http ",
		Configuration: GenericWebServiceConfigurationInput{
			TemplateURL: " https://example.test/items/{item-id} ",
			Method:      " post ",
			Format:      " Application/JSON ; Charset=UTF-8 ",
			Parameters: []WebServiceParameterInput{
				{Position: 1, Name: "z", Label: " Z ", Schema: StringSchema{}, Location: ParameterLocationQuery},
				{Position: 0, Name: "item-id", Label: " Item ", Required: true, Schema: StringSchema{}, Location: ParameterLocationPath},
				{Position: 1, Name: "a", Label: " A ", Schema: StringSchema{}, Location: ParameterLocationQuery},
			},
		},
		VisibleOn: []WebServiceVisibleOn{" map ", "GRAPH", "map"},
	})
	if err != nil {
		t.Fatalf("NewWebServiceInput() error = %v", err)
	}
	configuration := input.Configuration.(GenericWebServiceConfigurationInput)
	if configuration.TemplateURL != "https://example.test/items/{item-id}" || configuration.Method != HTTPMethodPost || input.Type != WebServiceTypeHTTP || configuration.Format != "Application/JSON ; Charset=UTF-8" {
		t.Fatalf("normalized web service = %+v", input)
	}
	if got := input.VisibleOn; !slices.Equal(got, []WebServiceVisibleOn{WebServiceVisibleOnMap, WebServiceVisibleOnGraph}) {
		t.Fatalf("visible on = %v, want [MAP GRAPH]", got)
	}
	if got := []string{configuration.Parameters[0].Name, configuration.Parameters[1].Name, configuration.Parameters[2].Name}; !slices.Equal(got, []string{"item-id", "a", "z"}) {
		t.Fatalf("parameter order = %v", got)
	}
	if configuration.Parameters[0].Label != "Item" {
		t.Fatalf("parameter label = %q, want Item", configuration.Parameters[0].Label)
	}
}

func TestNewWebServiceInputNormalizesDocumentationURI(t *testing.T) {
	documentationURI := " https://example.test/docs "
	input, err := NewWebServiceInput(WebServiceInput{
		Type:             WebServiceTypeHTTP,
		DocumentationURI: &documentationURI,
		Configuration: GenericWebServiceConfigurationInput{
			TemplateURL: "https://example.test",
			Method:      HTTPMethodGet,
			Format:      "application/json",
		},
	})
	if err != nil {
		t.Fatalf("NewWebServiceInput() error = %v", err)
	}
	if input.DocumentationURI == nil || *input.DocumentationURI != "https://example.test/docs" {
		t.Fatalf("documentation URI = %v, want normalized URI", input.DocumentationURI)
	}

	emptyURI := "  "
	_, err = NewWebServiceInput(WebServiceInput{
		Type:             WebServiceTypeHTTP,
		DocumentationURI: &emptyURI,
		Configuration: GenericWebServiceConfigurationInput{
			TemplateURL: "https://example.test",
			Method:      HTTPMethodGet,
			Format:      "application/json",
		},
	})
	appErr, ok := errors.AsType[*apperror.AppError](err)
	if !ok || appErr.Target != "documentation_uri" {
		t.Fatalf("empty documentation URI error = %v, want documentation_uri validation error", err)
	}
}

func TestNewWebServiceInputRejectsDuplicateNamesAcrossLocations(t *testing.T) {
	_, err := NewWebServiceInput(WebServiceInput{
		Type: WebServiceTypeHTTP,
		Configuration: GenericWebServiceConfigurationInput{
			TemplateURL: "https://example.test",
			Method:      HTTPMethodGet,
			Format:      "application/json",
			Parameters: []WebServiceParameterInput{
				{Name: "value", Label: "Value", Schema: StringSchema{}, Location: ParameterLocationQuery},
				{Name: "value", Label: "Value", Schema: StringSchema{}, Location: ParameterLocationHeader},
			},
		},
	})
	assertInputValidation(t, err, "configuration.parameters[1].name")
}

func TestNewWebServiceParameterInputAllowsMultipleEnumValues(t *testing.T) {
	tests := []struct {
		name   string
		schema ParameterSchema
	}{
		{name: "string", schema: StringSchema{Default: []string{"json", "xml"}, Enum: []StringEnumOption{{Value: "json"}, {Value: "xml"}}, MultipleValues: true}},
		{name: "integer", schema: IntegerSchema{Default: []int64{1, 2}, Enum: []IntegerEnumOption{{Value: 1}, {Value: 2}}, MultipleValues: true}},
		{name: "float", schema: FloatSchema{Default: []float64{1.5, 2.5}, Enum: []FloatEnumOption{{Value: 1.5}, {Value: 2.5}}, MultipleValues: true}},
		{name: "string without enum", schema: StringSchema{Default: []string{"json", "xml"}, MultipleValues: true}},
		{name: "integer without enum", schema: IntegerSchema{Default: []int64{1, 2}, MultipleValues: true}},
		{name: "float without enum", schema: FloatSchema{Default: []float64{1.5, 2.5}, MultipleValues: true}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, err := NewWebServiceParameterInput(WebServiceParameterInput{
				Name:     "value",
				Label:    "Value",
				Schema:   test.schema,
				Location: ParameterLocationQuery,
				Explode:  true,
			})
			if err != nil {
				t.Fatalf("NewWebServiceParameterInput() error = %v", err)
			}
			if !input.Schema.AllowsMultipleValues() {
				t.Fatal("schema multiple values = false, want true")
			}
			defaults, ok := input.Schema.DefaultValue().(AccessArray)
			if !ok || len(defaults) != 2 {
				t.Fatalf("default value = %#v, want two-value array", input.Schema.DefaultValue())
			}
		})
	}

	for _, defaults := range [][]string{{}, {"json", "xml"}} {
		_, err := NewWebServiceParameterInput(WebServiceParameterInput{
			Name: "value", Label: "Value",
			Schema: StringSchema{
				Default: defaults,
				Enum:    []StringEnumOption{{Value: "json"}, {Value: "xml"}},
			},
			Location: ParameterLocationQuery,
		})
		if !apperror.IsCode(err, apperror.CodeValidationFailed) {
			t.Fatalf("scalar defaults %v error = %v, want validation_failed", defaults, err)
		}
	}
}

func TestNewWebServiceParameterInputNormalizesEnumLabels(t *testing.T) {
	label := " JSON document "
	input, err := NewWebServiceParameterInput(WebServiceParameterInput{
		Name:  "format",
		Label: "Format",
		Schema: StringSchema{
			Default: []string{"json"},
			Enum:    []StringEnumOption{{Value: "json", Label: &label}, {Value: "xml"}},
		},
		Location: ParameterLocationQuery,
	})
	if err != nil {
		t.Fatalf("NewWebServiceParameterInput() error = %v", err)
	}

	schema := input.Schema.(StringSchema)
	if schema.Enum[0].Label == nil || *schema.Enum[0].Label != "JSON document" {
		t.Fatalf("enum label = %v, want normalized label", schema.Enum[0].Label)
	}
	if got := schema.DefaultValue(); got != AccessString("json") {
		t.Fatalf("default value = %#v, want scalar json", got)
	}
}

func TestNewWebServiceParameterInputRejectsDuplicateEnumValues(t *testing.T) {
	_, err := NewWebServiceParameterInput(WebServiceParameterInput{
		Name:  "format",
		Label: "Format",
		Schema: IntegerSchema{
			Enum: []IntegerEnumOption{{Value: 1}, {Value: 1}},
		},
		Location: ParameterLocationQuery,
	})
	if !apperror.IsCode(err, apperror.CodeValidationFailed) || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("error = %v, want duplicate enum error", err)
	}
}

func TestNewWebServiceParameterInputRejectsInvalidExplode(t *testing.T) {
	tests := []struct {
		name     string
		location ParameterLocation
		multiple bool
	}{
		{name: "non-query", location: ParameterLocationPath, multiple: true},
		{name: "single-value", location: ParameterLocationQuery, multiple: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewWebServiceParameterInput(WebServiceParameterInput{
				Name:     "value",
				Label:    "Value",
				Schema:   StringSchema{Enum: []StringEnumOption{{Value: "one"}, {Value: "two"}}, MultipleValues: test.multiple},
				Location: test.location,
				Explode:  true,
			})
			assertInputValidation(t, err, "explode")
		})
	}
}

func TestNewWebServiceParameterInputAllowsMultipleValuesWithoutEnum(t *testing.T) {
	_, err := NewWebServiceParameterInput(WebServiceParameterInput{
		Name:     "value",
		Label:    "Value",
		Schema:   StringSchema{MultipleValues: true},
		Location: ParameterLocationQuery,
	})
	if err != nil {
		t.Fatalf("NewWebServiceParameterInput() error = %v", err)
	}
}

func TestNewWebServiceInputRejectsInvalidType(t *testing.T) {
	_, err := NewWebServiceInput(WebServiceInput{
		Type: "KML",
	})
	assertInputValidation(t, err, "type")
}

func TestNormalizeMediaTypeIncludesInvalidInputInError(t *testing.T) {
	for _, value := range []string{"not a media type", "json"} {
		_, err := normalizeMediaType(value)
		if !apperror.IsCode(err, apperror.CodeValidationFailed) || !strings.Contains(err.Error(), `"`+value+`"`) {
			t.Errorf("normalizeMediaType(%q) error = %v, want invalid dataset error containing input", value, err)
		}
	}
}

func TestNewWebServiceInputRejectsInvalidVisibleOn(t *testing.T) {
	tests := []struct {
		name      string
		visibleOn []WebServiceVisibleOn
	}{
		{name: "empty", visibleOn: []WebServiceVisibleOn{}},
		{name: "unsupported", visibleOn: []WebServiceVisibleOn{"GRAPH", "FORM"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewWebServiceInput(WebServiceInput{
				Type:      WebServiceTypeHTTP,
				VisibleOn: test.visibleOn,
				Configuration: GenericWebServiceConfigurationInput{
					TemplateURL: "https://example.test",
					Method:      HTTPMethodGet,
					Format:      "application/json",
				},
			})
			assertInputValidation(t, err, "visible_on")
		})
	}
}

func assertInputValidation(t *testing.T, err error, target string) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %T, want *apperror.AppError: %v", err, err)
	}
	if !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("error = %v, want validation_failed", err)
	}
	if appErr.Target != target {
		t.Fatalf("AppError target = %q, want %q", appErr.Target, target)
	}
}

func TestNewWebServiceNormalizesTypeAndVisibleOn(t *testing.T) {
	id := uuid.New()
	service, err := NewWebService(WebService{
		ID:        id,
		Type:      " wms ",
		VisibleOn: []WebServiceVisibleOn{"TABLE", "table", " graph "},
		Configuration: WMSConfiguration{
			CapabilitiesURL: "https://example.test/wms?SERVICE=WMS&REQUEST=GetCapabilities",
			DefaultLayer:    "layer",
			Layers: []WMSLayer{{
				Name: "layer", DefaultStyle: "", Styles: []string{""},
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewWebService() error = %v", err)
	}
	if service.Type != WebServiceTypeWMS || !slices.Equal(service.VisibleOn, []WebServiceVisibleOn{WebServiceVisibleOnTable, WebServiceVisibleOnGraph}) {
		t.Fatalf("service = %+v, want normalized type and visible-on values", service)
	}
}

func TestNewWebServiceInputValidatesBodyAgainstMethod(t *testing.T) {
	body := &RequestBodyInput{
		ContentType: "application/json",
		Template:    `{"value": {{.value}}}`,
		Parameters: []ParameterInput{{
			Name:   "value",
			Label:  "Value",
			Schema: StringSchema{},
		}},
	}
	_, err := NewWebServiceInput(WebServiceInput{
		Type: WebServiceTypeHTTP,
		Configuration: GenericWebServiceConfigurationInput{
			TemplateURL: "https://example.test",
			Method:      HTTPMethodGet,
			Format:      "application/json",
			Body:        body,
		},
	})
	if !apperror.IsCode(err, apperror.CodeValidationFailed) || !strings.Contains(err.Error(), "body is not allowed") {
		t.Fatalf("error = %v, want method body error", err)
	}

	input, err := NewWebServiceInput(WebServiceInput{
		Type: WebServiceTypeHTTP,
		Configuration: GenericWebServiceConfigurationInput{
			TemplateURL: "https://example.test",
			Method:      HTTPMethodPost,
			Format:      "application/json",
			Body:        body,
		},
	})
	if err != nil {
		t.Fatalf("POST NewWebServiceInput() error = %v", err)
	}
	configuration := input.Configuration.(GenericWebServiceConfigurationInput)
	if configuration.Body == nil || configuration.Body.Parameters[0].Name != "value" {
		t.Fatalf("body = %+v", configuration.Body)
	}
}

func TestNewRequestBodyRejectsUndeclaredTemplateReference(t *testing.T) {
	_, err := NewRequestBodyInput(RequestBodyInput{
		ContentType: "application/json",
		Template:    `{"value": {{.missing}}}`,
		Parameters: []ParameterInput{{
			Name:   "value",
			Label:  "Value",
			Schema: StringSchema{},
		}},
	})
	if !apperror.IsCode(err, apperror.CodeValidationFailed) || !strings.Contains(err.Error(), "undeclared parameter") {
		t.Fatalf("error = %v, want undeclared reference error", err)
	}
}

func TestParseTemplateSchemaDisallowsNestedTemplates(t *testing.T) {
	_, err := NewRequestBodyInput(RequestBodyInput{
		ContentType: "application/json",
		Template:    `{"value": {{.value}}}`,
		Parameters: []ParameterInput{{
			Name:  "value",
			Label: "Value",
			Schema: TemplateSchemaInput{
				Template: "{{.nested}}",
				Parameters: []ParameterInput{{
					Name: "nested", Label: "Nested", Schema: StringSchema{},
				}},
			},
		}},
	})
	if !apperror.IsCode(err, apperror.CodeValidationFailed) || !strings.Contains(err.Error(), "template schema is not allowed") {
		t.Fatalf("error = %v, want nested template error", err)
	}
}

func TestDateTimeLayout(t *testing.T) {
	if err := validateDateTimeLayout("YYYY-MM-DDThh:mm:ssZ"); err != nil {
		t.Fatalf("validateDateTimeLayout() error = %v", err)
	}
	if err := validateDateTimeLayout("yyyy-MM-DD"); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("unsupported layout error = %v", err)
	}
}
