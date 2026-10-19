package grpcapi

import (
	"slices"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func TestParameterMappingsUseScalarTemplateAndBodyChildren(t *testing.T) {
	child := domain.Parameter{
		Name:   "value",
		Label:  "Value",
		Schema: domain.StringSchema{Enum: []domain.StringEnumOption{{Value: "one"}, {Value: "two"}}},
		ID:     uuid.New(),
	}
	value := domain.Parameter{
		Name:  "filter",
		Label: "Filter",
		Schema: domain.TemplateSchema{
			Template:   "{{.value}}",
			Parameters: []domain.Parameter{child},
		},
		ID: uuid.New(),
	}

	proto := parameterOutput(domain.WebServiceParameter{Parameter: value, Location: domain.ParameterLocationQuery})
	template := proto.GetTemplate()
	if template == nil || len(template.Parameters) != 1 {
		t.Fatalf("template parameters = %+v, want one scalar parameter", template)
	}
	if template.Parameters[0].GetString_() == nil || len(template.Parameters[0].GetString_().GetEnum()) != 2 {
		t.Fatalf("template child = %+v, want scalar string parameter", template.Parameters[0])
	}
	inputProto := &metadatalifecyclev1.ParameterInput{
		Name:  "filter",
		Label: "Filter",
		Value: &metadatalifecyclev1.ParameterInput_Template{Template: &metadatalifecyclev1.TemplateParameterInput{
			Template: "{{.value}}",
			Parameters: []*metadatalifecyclev1.ScalarParameterInput{{
				Name:  "value",
				Label: "Value",
				Value: &metadatalifecyclev1.ScalarParameterInput_String_{String_: &metadatalifecyclev1.ScalarStringParameter{
					Enum: []*metadatalifecyclev1.StringEnumOption{{Value: "one"}, {Value: "two"}},
				}},
			}},
		}},
	}
	mapped, err := parameterInput(inputProto)
	if err != nil {
		t.Fatalf("parameterInput() error = %v", err)
	}
	mappedTemplate, ok := mapped.Schema.(domain.TemplateSchemaInput)
	if !ok || len(mappedTemplate.Parameters) != 1 || len(mappedTemplate.Parameters[0].Schema.(domain.StringSchema).Enum) != 2 {
		t.Fatalf("mapped template = %+v, want one scalar child", mapped.Schema)
	}

	contentType := "application/json"
	bodyTemplate := "{{.value}}"
	service, err := webServiceInput(&metadatalifecyclev1.WebServiceInput{
		Name:            "service",
		TemplateUrl:     "https://example.test",
		Method:          metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_POST,
		Type:            metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_GEOJSON,
		Format:          "application/json",
		BodyContentType: &contentType,
		BodyTemplate:    &bodyTemplate,
		BodyParameters: []*metadatalifecyclev1.ScalarParameterInput{{
			Name:  "value",
			Label: "Value",
			Value: &metadatalifecyclev1.ScalarParameterInput_String_{String_: &metadatalifecyclev1.ScalarStringParameter{
				Enum: []*metadatalifecyclev1.StringEnumOption{{Value: "one"}, {Value: "two"}},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("webServiceInput() error = %v", err)
	}
	if service.Body == nil || len(service.Body.Parameters) != 1 {
		t.Fatalf("mapped body = %+v, want one scalar child", service.Body)
	}
}

func TestWebServiceParameterMappingsPreserveExplode(t *testing.T) {
	parameter := &metadatalifecyclev1.ParameterInput{
		Name:  "value",
		Label: "Value",
		Value: &metadatalifecyclev1.ParameterInput_String_{String_: &metadatalifecyclev1.StringParameter{
			Enum:           []*metadatalifecyclev1.StringEnumOption{{Value: "one"}, {Value: "two"}},
			MultipleValues: true,
		}},
		Location: metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_QUERY,
		Explode:  true,
	}
	input, err := webServiceInput(&metadatalifecyclev1.WebServiceInput{
		Name:        "service",
		TemplateUrl: "https://example.test/{?value}",
		Method:      metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET,
		Type:        metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_GEOJSON,
		VisibleOn: []metadatalifecyclev1.WebServiceVisibleOn{
			metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH,
			metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_MAP,
		},
		Format:     "application/json",
		Parameters: []*metadatalifecyclev1.ParameterInput{parameter},
	})
	if err != nil {
		t.Fatalf("webServiceInput() error = %v", err)
	}
	if len(input.Parameters) != 1 || !input.Parameters[0].Explode {
		t.Fatalf("input parameters = %+v, want explode parameter", input.Parameters)
	}
	if input.Type != domain.WebServiceTypeGeoJSON || !slices.Equal(input.VisibleOn, []domain.WebServiceVisibleOn{domain.WebServiceVisibleOnGraph, domain.WebServiceVisibleOnMap}) {
		t.Fatalf("input web-service fields = type %q, visible on %v", input.Type, input.VisibleOn)
	}

	id := uuid.New()
	property := "https://example.test/value"
	output := webServiceOutput(domain.WebService{
		ID:          id,
		Name:        "service",
		TemplateURL: "https://example.test/{?value}",
		Method:      domain.HTTPMethodGet,
		Type:        domain.WebServiceTypeGeoJSON,
		VisibleOn:   []domain.WebServiceVisibleOn{domain.WebServiceVisibleOnGraph, domain.WebServiceVisibleOnMap},
		Format:      "application/json",
		Parameters: []domain.WebServiceParameter{{
			Name:     "value",
			Label:    "Value",
			Schema:   domain.StringSchema{Enum: []domain.StringEnumOption{{Value: "one"}, {Value: "two"}}, MultipleValues: true},
			ID:       id,
			Property: &property,
			Location: domain.ParameterLocationQuery,
			Explode:  true,
		}},
	})
	if len(output.Parameters) != 1 || output.Parameters[0].GetParameter() == nil || !output.Parameters[0].GetParameter().GetExplode() {
		t.Fatalf("output parameters = %+v, want explode parameter", output.Parameters)
	}
	if output.Type != metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_GEOJSON || len(output.VisibleOn) != 2 || output.VisibleOn[0] != metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH || output.VisibleOn[1] != metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_MAP {
		t.Fatalf("output web-service fields = type %q, visible on %v", output.Type, output.VisibleOn)
	}
}

func TestWebServiceInputRejectsUnspecifiedEnums(t *testing.T) {
	base := func() *metadatalifecyclev1.WebServiceInput {
		return &metadatalifecyclev1.WebServiceInput{
			Name:        "service",
			TemplateUrl: "https://example.test",
			Format:      "application/json",
			Method:      metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET,
			Type:        metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_GEOJSON,
		}
	}

	method := base()
	method.Method = metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_UNSPECIFIED
	if _, err := webServiceInput(method); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("unspecified method error = %v, want validation_failed", err)
	}

	typeValue := base()
	typeValue.Type = metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_UNSPECIFIED
	if _, err := webServiceInput(typeValue); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("unspecified type error = %v, want validation_failed", err)
	}

	visibleOn := base()
	visibleOn.VisibleOn = []metadatalifecyclev1.WebServiceVisibleOn{
		metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_UNSPECIFIED,
	}
	if _, err := webServiceInput(visibleOn); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("unspecified visible-on error = %v, want validation_failed", err)
	}

	parameter := base()
	parameter.Parameters = []*metadatalifecyclev1.ParameterInput{{
		Name:  "value",
		Label: "Value",
		Value: &metadatalifecyclev1.ParameterInput_String_{String_: &metadatalifecyclev1.StringParameter{}},
	}}
	if _, err := webServiceInput(parameter); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("unspecified parameter location error = %v, want validation_failed", err)
	}
}
