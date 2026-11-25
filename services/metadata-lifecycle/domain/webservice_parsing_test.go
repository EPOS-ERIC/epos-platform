package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

func TestNewWebServiceInputNormalizesAndPreservesParameterOrder(t *testing.T) {
	input := HTTPWebServiceInput{
		VisibleOn:   []WebServiceVisibleOn{" map ", "GRAPH", "map"},
		TemplateURL: " https://example.test/items/{item-id} ",
		Method:      " post ",
		Format:      " Application/JSON ; Charset=UTF-8 ",
		Parameters: []WebServiceParameterInput{
			{Position: 1, Name: "z", Label: " Z ", Schema: StringSchema{}, Location: ParameterLocationQuery},
			{Position: 0, Name: "item-id", Label: " Item ", Required: true, Schema: StringSchema{}, Location: ParameterLocationPath},
			{Position: 1, Name: "a", Label: " A ", Schema: StringSchema{}, Location: ParameterLocationQuery},
		},
	}
	err := input.Normalize()
	if err != nil {
		t.Fatalf("NewWebServiceInput() error = %v", err)
	}
	configuration := input.GenericWebServiceInput
	if configuration.TemplateURL != "https://example.test/items/{item-id}" || configuration.Method != HTTPMethodPost || input.ServiceType() != WebServiceTypeHTTP || configuration.Format != "Application/JSON ; Charset=UTF-8" {
		t.Fatalf("normalized web service = %+v", input)
	}
	if got := input.VisibleOn; !slices.Equal(got, []WebServiceVisibleOn{WebServiceVisibleOnMap, WebServiceVisibleOnGraph}) {
		t.Fatalf("visible on = %v, want [MAP GRAPH]", got)
	}
	if got := []string{configuration.Parameters[0].Name, configuration.Parameters[1].Name, configuration.Parameters[2].Name}; !slices.Equal(got, []string{"z", "item-id", "a"}) {
		t.Fatalf("parameter order = %v", got)
	}
	if configuration.Parameters[1].Label != "Item" {
		t.Fatalf("parameter label = %q, want Item", configuration.Parameters[1].Label)
	}
}

func TestNewWebServiceInputNormalizesDocumentationURI(t *testing.T) {
	documentationURI := " https://example.test/docs "
	input := HTTPWebServiceInput{
		DocumentationURI: &documentationURI,
		TemplateURL:      "https://example.test",
		Method:           HTTPMethodGet,
		Format:           "application/json",
	}
	err := input.Normalize()
	if err != nil {
		t.Fatalf("NewWebServiceInput() error = %v", err)
	}
	if input.DocumentationURI == nil || *input.DocumentationURI != "https://example.test/docs" {
		t.Fatalf("documentation URI = %v, want normalized URI", input.DocumentationURI)
	}
	if input.DocumentationURI != &documentationURI {
		t.Fatal("documentation URI normalization must update the original string")
	}

	for _, invalidURI := range []string{"  ", "relative/path", "urn:example:docs", "https://:443/docs", "https://example.test/%zz"} {
		input.DocumentationURI = &invalidURI
		err := input.Normalize()
		appErr, ok := errors.AsType[*apperror.AppError](err)
		if !ok || appErr.Target != "documentation_uri" {
			t.Errorf("documentation URI %q error = %v, want documentation_uri validation error", invalidURI, err)
		}
	}
}

func TestNewWebServiceInputRejectsDuplicateNamesAcrossLocations(t *testing.T) {
	input := HTTPWebServiceInput{
		TemplateURL: "https://example.test",
		Method:      HTTPMethodGet,
		Format:      "application/json",
		Parameters: []WebServiceParameterInput{
			{Name: "value", Label: "Value", Schema: StringSchema{}, Location: ParameterLocationQuery},
			{Name: "value", Label: "Value", Schema: StringSchema{}, Location: ParameterLocationHeader},
		},
	}
	err := input.Normalize()
	assertInputValidation(t, err, "parameters[1].name")
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
			input := WebServiceParameterInput{
				Name:     "value",
				Label:    "Value",
				Schema:   test.schema,
				Location: ParameterLocationQuery,
				Explode:  true,
			}
			err := input.Normalize()
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
		input := WebServiceParameterInput{
			Name: "value", Label: "Value",
			Schema: StringSchema{
				Default: defaults,
				Enum:    []StringEnumOption{{Value: "json"}, {Value: "xml"}},
			},
			Location: ParameterLocationQuery,
		}
		err := input.Normalize()
		if !apperror.IsCode(err, apperror.CodeValidationFailed) {
			t.Fatalf("scalar defaults %v error = %v, want validation_failed", defaults, err)
		}
	}
}

func TestNewWebServiceParameterInputNormalizesEnumLabels(t *testing.T) {
	label := " JSON document "
	input := WebServiceParameterInput{
		Name:  "format",
		Label: "Format",
		Schema: StringSchema{
			Default: []string{"json"},
			Enum:    []StringEnumOption{{Value: "json", Label: &label}, {Value: "xml"}},
		},
		Location: ParameterLocationQuery,
	}
	err := input.Normalize()
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
	input := WebServiceParameterInput{
		Name:  "format",
		Label: "Format",
		Schema: IntegerSchema{
			Enum: []IntegerEnumOption{{Value: 1}, {Value: 1}},
		},
		Location: ParameterLocationQuery,
	}
	err := input.Normalize()
	assertInputValidation(t, err, "enum[1].value")
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
			input := WebServiceParameterInput{
				Name:     "value",
				Label:    "Value",
				Schema:   StringSchema{Enum: []StringEnumOption{{Value: "one"}, {Value: "two"}}, MultipleValues: test.multiple},
				Location: test.location,
				Explode:  true,
			}
			err := input.Normalize()
			assertInputValidation(t, err, "explode")
		})
	}
}

func TestNewWebServiceParameterInputAllowsMultipleValuesWithoutEnum(t *testing.T) {
	input := WebServiceParameterInput{
		Name:     "value",
		Label:    "Value",
		Schema:   StringSchema{MultipleValues: true},
		Location: ParameterLocationQuery,
	}
	err := input.Normalize()
	if err != nil {
		t.Fatalf("NewWebServiceParameterInput() error = %v", err)
	}
}

func TestNewWebServiceInputRejectsInvalidType(t *testing.T) {
	if WebServiceType("KML").Valid() {
		t.Fatal("KML service type must be invalid")
	}
	inputs := []WebServiceInput{
		(*HTTPWebServiceInput)(nil), (*DynamicWebServiceInput)(nil), (*WFSWebServiceInput)(nil),
		(*WMSWebServiceInput)(nil), (*WMTSWebServiceInput)(nil),
	}
	for _, input := range inputs {
		distribution := DistributionInput{WebService: input, DownloadableFile: &DownloadableFile{}}
		if err := distribution.Normalize(); !apperror.IsCode(err, apperror.CodeInternal) {
			t.Fatalf("typed-nil %T input error = %v, want internal", input, err)
		}
	}
	services := []WebService{
		(*HTTPWebService)(nil), (*DynamicWebService)(nil), (*WFSWebService)(nil),
		(*WMSWebService)(nil), (*WMTSWebService)(nil),
	}
	for _, service := range services {
		persisted := Distribution{WebService: service, DownloadableFile: &DownloadableFile{}}
		if err := persisted.Normalize(); !apperror.IsCode(err, apperror.CodeInternal) {
			t.Fatalf("typed-nil %T error = %v, want internal", service, err)
		}
		if executable, ok := service.(ExecutableWebService); ok {
			if _, err := executable.PrepareAccess(DistributionAccessInput{}); !apperror.IsCode(err, apperror.CodeInternal) {
				t.Fatalf("typed-nil %T access error = %v, want internal", service, err)
			}
		}
	}
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
			input := HTTPWebServiceInput{
				VisibleOn:   test.visibleOn,
				TemplateURL: "https://example.test",
				Method:      HTTPMethodGet,
				Format:      "application/json",
			}
			err := input.Normalize()
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
	service := WMSWebService{
		ID:              id,
		VisibleOn:       []WebServiceVisibleOn{"TABLE", "table", " graph "},
		CapabilitiesURL: "https://example.test/wms?SERVICE=WMS&REQUEST=GetCapabilities",
		DefaultLayer:    "layer",
		Layers: []WMSLayer{{
			Name: "layer", DefaultStyle: "", Styles: []string{""},
		}},
	}
	err := service.Normalize()
	if err != nil {
		t.Fatalf("NewWebService() error = %v", err)
	}
	if service.ServiceType() != WebServiceTypeWMS || !slices.Equal(service.VisibleOn, []WebServiceVisibleOn{WebServiceVisibleOnTable, WebServiceVisibleOnGraph}) {
		t.Fatalf("service = %+v, want normalized type and visible-on values", service)
	}

	longitudeID, latitudeID := uuid.New(), uuid.New()
	longitudeRole, latitudeRole := WebServiceParameterRoleLongitude, WebServiceParameterRoleLatitude
	dynamic := DynamicWebService{
		ID:          uuid.New(),
		TemplateURL: "https://example.test",
		Method:      HTTPMethodGet,
		Format:      "application/json",
		Parameters: []WebServiceParameter{
			{ID: longitudeID, Name: "lon", Label: "Longitude", Role: &longitudeRole, Required: true, Schema: FloatSchema{}, Location: ParameterLocationQuery},
			{ID: latitudeID, Name: "lat", Label: "Latitude", Role: &latitudeRole, Required: true, Schema: FloatSchema{}, Location: ParameterLocationQuery},
		},
	}
	if err := dynamic.Normalize(); err != nil {
		t.Fatalf("dynamic normalize: %v", err)
	}
	request, err := dynamic.PrepareAccess(DistributionAccessInput{Input: DynamicAccessInput{Longitude: 12, Latitude: 34}})
	if err != nil || !strings.Contains(request.URL, "lon=12") || !strings.Contains(request.URL, "lat=34") {
		t.Fatalf("dynamic request = %+v, error = %v", request, err)
	}
	if dynamic.Parameters[0].ID != longitudeID && dynamic.Parameters[1].ID != longitudeID {
		t.Fatal("dynamic normalization lost longitude ID")
	}

	wfs := WFSWebService{
		ID:                    uuid.New(),
		GetFeatureTemplateURL: "https://example.test/wfs",
		Version:               "2.0.0",
		FeatureTypes:          []string{"station"},
		DefaultFeatureType:    "station",
		OutputFormats:         []string{"application/json"},
		DefaultOutputFormat:   "application/json",
	}
	if err := wfs.Normalize(); err != nil {
		t.Fatalf("WFS normalize: %v", err)
	}
	request, err = wfs.PrepareAccess(DistributionAccessInput{Input: WFSAccessInput{}})
	if err != nil || !strings.Contains(request.URL, "typeNames=station") || !strings.Contains(request.URL, "SERVICE=WFS") {
		t.Fatalf("WFS request = %+v, error = %v", request, err)
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
	input := HTTPWebServiceInput{
		TemplateURL: "https://example.test",
		Method:      HTTPMethodGet,
		Format:      "application/json",
		Body:        body,
	}
	err := input.Normalize()
	if !apperror.IsCode(err, apperror.CodeValidationFailed) || !strings.Contains(err.Error(), "body is not allowed") {
		t.Fatalf("error = %v, want method body error", err)
	}

	input = HTTPWebServiceInput{
		TemplateURL: "https://example.test",
		Method:      HTTPMethodPost,
		Format:      "application/json",
		Body:        body,
	}
	err = input.Normalize()
	if err != nil {
		t.Fatalf("POST NewWebServiceInput() error = %v", err)
	}
	configuration := input.GenericWebServiceInput
	if configuration.Body == nil || configuration.Body.Parameters[0].Name != "value" {
		t.Fatalf("body = %+v", configuration.Body)
	}
	bodyParameterID := uuid.New()
	templateID, childID := uuid.New(), uuid.New()
	persisted := HTTPWebService{
		ID:          uuid.New(),
		TemplateURL: "https://example.test",
		Method:      HTTPMethodPost,
		Format:      "application/json",
		Parameters: []WebServiceParameter{{
			ID:   templateID,
			Name: "template", Label: "Template",
			Schema: TemplateSchema{Template: "{{.child}}", Parameters: []Parameter{{
				ID: childID, Name: "child", Label: "Child", Schema: StringSchema{},
			}}},
			Location: ParameterLocationQuery,
		}},
		Body: &RequestBody{
			Required:    true,
			ContentType: "application/json",
			Template:    `{"value": {{.value}}}`,
			Parameters: []Parameter{{
				ID:   bodyParameterID,
				Name: "value", Label: "Value", Schema: StringSchema{},
			}},
		},
	}
	if err := persisted.Normalize(); err != nil {
		t.Fatalf("persisted HTTP normalize: %v", err)
	}
	if persisted.Body.Parameters[0].ID != bodyParameterID {
		t.Fatal("body parameter ID changed")
	}
	schema := persisted.Parameters[0].Schema.(TemplateSchema)
	if persisted.Parameters[0].ID != templateID || schema.Parameters[0].ID != childID {
		t.Fatal("template parameter IDs changed")
	}
}

func TestNewRequestBodyRejectsUndeclaredTemplateReference(t *testing.T) {
	input := RequestBodyInput{
		ContentType: "application/json",
		Template:    `{"value": {{.missing}}}`,
		Parameters: []ParameterInput{{
			Name:   "value",
			Label:  "Value",
			Schema: StringSchema{},
		}},
	}
	err := input.Normalize()
	if !apperror.IsCode(err, apperror.CodeValidationFailed) || !strings.Contains(err.Error(), "undeclared parameter") {
		t.Fatalf("error = %v, want undeclared reference error", err)
	}
}

func TestParseTemplateSchemaDisallowsNestedTemplates(t *testing.T) {
	input := RequestBodyInput{
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
	}
	err := input.Normalize()
	assertInputValidation(t, err, "body_parameters[0].kind")
}

func TestDateTimeLayout(t *testing.T) {
	if err := validateDateTimeLayout("YYYY-MM-DDThh:mm:ssZ"); err != nil {
		t.Fatalf("validateDateTimeLayout() error = %v", err)
	}
	if err := validateDateTimeLayout("yyyy-MM-DD"); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("unsupported layout error = %v", err)
	}
}
