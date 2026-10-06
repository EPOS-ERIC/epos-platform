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
		Name:        " service ",
		TemplateURL: " https://example.test/items/{item-id} ",
		Method:      " post ",
		Type:        " geojson ",
		VisibleOn:   []WebServiceVisibleOn{" map ", "GRAPH", "map"},
		Format:      " Application/JSON ; Charset=UTF-8 ",
		Parameters: []WebServiceParameterInput{
			{Position: 1, Name: "z", Label: " Z ", Schema: StringSchema{}, Location: ParameterLocationQuery},
			{Position: 0, Name: "item-id", Label: " Item ", Schema: StringSchema{}, Location: ParameterLocationPath},
			{Position: 1, Name: "a", Label: " A ", Schema: StringSchema{}, Location: ParameterLocationQuery},
		},
	})
	if err != nil {
		t.Fatalf("NewWebServiceInput() error = %v", err)
	}
	if input.Name != "service" || input.TemplateURL != "https://example.test/items/{item-id}" || input.Method != HTTPMethodPost || input.Type != WebServiceTypeGeoJSON || input.Format != "application/json; charset=UTF-8" {
		t.Fatalf("normalized web service = %+v", input)
	}
	if got := input.VisibleOn; !slices.Equal(got, []WebServiceVisibleOn{WebServiceVisibleOnMap, WebServiceVisibleOnGraph}) {
		t.Fatalf("visible on = %v, want [MAP GRAPH]", got)
	}
	if got := []string{input.Parameters[0].Name, input.Parameters[1].Name, input.Parameters[2].Name}; !slices.Equal(got, []string{"item-id", "a", "z"}) {
		t.Fatalf("parameter order = %v", got)
	}
	if input.Parameters[0].Label != "Item" {
		t.Fatalf("parameter label = %q, want Item", input.Parameters[0].Label)
	}
}

func TestNewWebServiceDescriptionIsOptionalButNotEmpty(t *testing.T) {
	base := func(description *string) WebServiceInput {
		return WebServiceInput{
			Name:        "service",
			Description: description,
			TemplateURL: "https://example.test",
			Method:      HTTPMethodGet,
			Type:        WebServiceTypeGeoJSON,
			Format:      "application/json",
		}
	}

	input, err := NewWebServiceInput(base(nil))
	if err != nil || input.Description != nil {
		t.Fatalf("nil description = %v, %v; want accepted nil", input.Description, err)
	}

	description := " description "
	input, err = NewWebServiceInput(base(&description))
	if err != nil || input.Description == nil || *input.Description != "description" {
		t.Fatalf("description = %v, %v; want trimmed description", input.Description, err)
	}

	empty := " "
	if _, err := NewWebServiceInput(base(&empty)); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("empty description error = %v, want invalid dataset", err)
	}
}

func TestNewWebServiceInputRejectsDuplicateNamesAcrossLocations(t *testing.T) {
	_, err := NewWebServiceInput(WebServiceInput{
		Name:        "service",
		TemplateURL: "https://example.test",
		Method:      HTTPMethodGet,
		Type:        WebServiceTypeGeoJSON,
		Format:      "application/json",
		Parameters: []WebServiceParameterInput{
			{Name: "value", Label: "Value", Schema: StringSchema{}, Location: ParameterLocationQuery},
			{Name: "value", Label: "Value", Schema: StringSchema{}, Location: ParameterLocationHeader},
		},
	})
	assertInputValidation(t, err, "parameters[1].name")
}

func TestNewWebServiceParameterInputAllowsMultipleEnumValues(t *testing.T) {
	tests := []struct {
		name   string
		schema ParameterSchema
	}{
		{name: "string", schema: StringSchema{Enum: []StringEnumOption{{Value: "json"}, {Value: "xml"}}, MultipleValues: true}},
		{name: "integer", schema: IntegerSchema{Enum: []IntegerEnumOption{{Value: 1}, {Value: 2}}, MultipleValues: true}},
		{name: "float", schema: FloatSchema{Enum: []FloatEnumOption{{Value: 1.5}, {Value: 2.5}}, MultipleValues: true}},
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
			if !schemaHasMultipleValues(input.Schema) {
				t.Fatal("schema multiple values = false, want true")
			}
		})
	}
}

func TestNewWebServiceParameterInputNormalizesEnumLabels(t *testing.T) {
	label := " JSON document "
	input, err := NewWebServiceParameterInput(WebServiceParameterInput{
		Name:  "format",
		Label: "Format",
		Schema: StringSchema{
			Enum: []StringEnumOption{{Value: "json", Label: &label}, {Value: "xml"}},
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

func TestNewWebServiceParameterInputRejectsMultipleValuesWithoutEnum(t *testing.T) {
	_, err := NewWebServiceParameterInput(WebServiceParameterInput{
		Name:     "value",
		Label:    "Value",
		Schema:   StringSchema{MultipleValues: true},
		Location: ParameterLocationQuery,
	})
	assertInputValidation(t, err, "schema.enum")
}

func TestNewWebServiceInputRejectsInvalidType(t *testing.T) {
	_, err := NewWebServiceInput(WebServiceInput{
		Name:        "service",
		TemplateURL: "https://example.test",
		Method:      HTTPMethodGet,
		Type:        "KML",
		Format:      "application/json",
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
				Name:        "service",
				TemplateURL: "https://example.test",
				Method:      HTTPMethodGet,
				Type:        WebServiceTypeGeoJSON,
				VisibleOn:   test.visibleOn,
				Format:      "application/json",
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
		ID:          id,
		Name:        "service",
		TemplateURL: "https://example.test",
		Method:      HTTPMethodGet,
		Type:        " wms ",
		VisibleOn:   []WebServiceVisibleOn{"TABLE", "table", " graph "},
		Format:      "application/json",
	})
	if err != nil {
		t.Fatalf("NewWebService() error = %v", err)
	}
	if service.Type != WebServiceTypeWMS || !slices.Equal(service.VisibleOn, []WebServiceVisibleOn{WebServiceVisibleOnTable, WebServiceVisibleOnGraph}) {
		t.Fatalf("service = %+v, want normalized type and visible-on values", service)
	}
}

func TestNewWebServiceInputValidatesBodyAgainstMethod(t *testing.T) {
	body := &RequestBody{
		ContentType: "application/json",
		Template:    `{"value": {{.value}}}`,
		Parameters: []Parameter{{
			Name:   "value",
			Label:  "Value",
			Schema: StringSchema{},
		}},
	}
	_, err := NewWebServiceInput(WebServiceInput{
		Name:        "service",
		TemplateURL: "https://example.test",
		Method:      HTTPMethodGet,
		Type:        WebServiceTypeGeoJSON,
		Format:      "application/json",
		Body:        body,
	})
	if !apperror.IsCode(err, apperror.CodeValidationFailed) || !strings.Contains(err.Error(), "body is not allowed") {
		t.Fatalf("error = %v, want method body error", err)
	}

	input, err := NewWebServiceInput(WebServiceInput{
		Name:        "service",
		TemplateURL: "https://example.test",
		Method:      HTTPMethodPost,
		Type:        WebServiceTypeGeoJSON,
		Format:      "application/json",
		Body:        body,
	})
	if err != nil {
		t.Fatalf("POST NewWebServiceInput() error = %v", err)
	}
	if input.Body == nil || input.Body.Parameters[0].Name != "value" {
		t.Fatalf("body = %+v", input.Body)
	}
}

func TestNewRequestBodyRejectsUndeclaredTemplateReference(t *testing.T) {
	_, err := NewRequestBody(RequestBody{
		ContentType: "application/json",
		Template:    `{"value": {{.missing}}}`,
		Parameters: []Parameter{{
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
	_, err := NewRequestBody(RequestBody{
		ContentType: "application/json",
		Template:    `{"value": {{.value}}}`,
		Parameters: []Parameter{{
			Name:  "value",
			Label: "Value",
			Schema: TemplateSchema{
				Template: "{{.nested}}",
				Parameters: []Parameter{{
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
