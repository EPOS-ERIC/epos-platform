package grpcapi

import (
	"slices"
	"testing"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParameterMappingsUseScalarTemplateAndBodyChildren(t *testing.T) {
	child := domain.Parameter{
		Name:   "value",
		Label:  "Value",
		Schema: domain.StringSchema{Enum: []domain.StringEnumOption{{Value: "one"}, {Value: "two"}}},
	}
	value := domain.Parameter{
		Name:  "filter",
		Label: "Filter",
		Schema: domain.TemplateSchema{
			Template:   "{{.value}}",
			Parameters: []domain.Parameter{child},
		},
	}

	proto := parameterOutput(domain.WebServiceParameter{Parameter: value, Location: domain.ParameterLocationQuery})
	template := proto.GetTemplate()
	if template == nil || len(template.Parameters) != 1 {
		t.Fatalf("template parameters = %+v, want one scalar parameter", template)
	}
	if template.Parameters[0].GetString_() == nil || len(template.Parameters[0].GetString_().GetEnum()) != 2 {
		t.Fatalf("template child = %+v, want scalar string parameter", template.Parameters[0])
	}

	mapped, err := parameterInput(proto)
	if err != nil {
		t.Fatalf("parameterInput() error = %v", err)
	}
	mappedTemplate, ok := mapped.Schema.(domain.TemplateSchema)
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
		Format:          "json",
		BodyContentType: &contentType,
		BodyTemplate:    &bodyTemplate,
		BodyParameters:  []*metadatalifecyclev1.ScalarParameter{template.Parameters[0]},
	})
	if err != nil {
		t.Fatalf("webServiceInput() error = %v", err)
	}
	if service.Body == nil || len(service.Body.Parameters) != 1 {
		t.Fatalf("mapped body = %+v, want one scalar child", service.Body)
	}
}

func TestWebServiceParameterMappingsPreserveExplode(t *testing.T) {
	parameter := &metadatalifecyclev1.Parameter{
		Name:  "value",
		Label: "Value",
		Value: &metadatalifecyclev1.Parameter_String_{String_: &metadatalifecyclev1.StringParameter{
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
		Format:     "json",
		Parameters: []*metadatalifecyclev1.Parameter{parameter},
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
		Format:      "json",
		Parameters: []domain.WebServiceParameter{{
			ID:       id,
			Name:     "value",
			Label:    "Value",
			Schema:   domain.StringSchema{Enum: []domain.StringEnumOption{{Value: "one"}, {Value: "two"}}, MultipleValues: true},
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
			Format:      "json",
			Method:      metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET,
			Type:        metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_GEOJSON,
		}
	}

	method := base()
	method.Method = metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_UNSPECIFIED
	if _, err := webServiceInput(method); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unspecified method error = %v, want InvalidArgument", err)
	}

	typeValue := base()
	typeValue.Type = metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_UNSPECIFIED
	if _, err := webServiceInput(typeValue); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unspecified type error = %v, want InvalidArgument", err)
	}

	visibleOn := base()
	visibleOn.VisibleOn = []metadatalifecyclev1.WebServiceVisibleOn{
		metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_UNSPECIFIED,
	}
	if _, err := webServiceInput(visibleOn); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unspecified visible-on error = %v, want InvalidArgument", err)
	}

	parameter := base()
	parameter.Parameters = []*metadatalifecyclev1.Parameter{{
		Name:  "value",
		Label: "Value",
		Value: &metadatalifecyclev1.Parameter_String_{String_: &metadatalifecyclev1.StringParameter{}},
	}}
	if _, err := webServiceInput(parameter); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unspecified parameter location error = %v, want InvalidArgument", err)
	}
}
