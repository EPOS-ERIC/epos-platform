package handlers

import (
	"encoding/json"
	"net/url"
	"slices"
	"testing"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func TestScalarParameterMappingsOmitTopLevelFields(t *testing.T) {
	draft := gatewayapi.NewStringScalarParameterDraftScalarParameterDraft(gatewayapi.StringScalarParameterDraft{
		Position: 1,
		Name:     "value",
		Label:    "Value",
		Enum:     []gatewayapi.StringEnumOption{{Value: "one", Label: gatewayapi.NewOptString("One")}, {Value: "two"}},
	})
	proto := toProtoScalarParameterDraft(draft)
	if proto == nil || proto.GetString_() == nil || len(proto.GetString_().GetEnum()) != 2 || proto.GetString_().GetEnum()[0].GetLabel() != "One" {
		t.Fatalf("protobuf scalar parameter = %+v, want scalar string parameter", proto)
	}

	label := "One"
	id := uuid.New()
	apiValue, err := toAPIScalarParameter(&metadatalifecyclev1.ScalarParameter{
		Position:   1,
		Name:       "value",
		Label:      "Value",
		Id:         id.String(),
		ResourceId: id.String(),
		Value: &metadatalifecyclev1.ScalarParameter_String_{String_: &metadatalifecyclev1.ScalarStringParameter{
			Enum: []*metadatalifecyclev1.StringEnumOption{{Value: "one", Label: &label}, {Value: "two"}},
		}},
	})
	if err != nil {
		t.Fatalf("toAPIScalarParameter() error = %v", err)
	}
	value, ok := apiValue.GetStringScalarParameter()
	if !ok || len(value.Enum) != 2 || !value.Enum[0].Label.Set || value.Enum[0].Label.Value != "One" {
		t.Fatalf("API scalar parameter = %+v, want scalar string parameter", apiValue)
	}
}

func TestScalarParameterArraysRejectTemplateValues(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name: "body parameter",
			payload: `{
                "name": "service",
                "template_url": "https://example.test",
                "method": "POST",
                "type": "GEOJSON",
                "format": "json",
                "body_parameters": [{
                    "kind": "TEMPLATE",
                    "position": 0,
                    "name": "value",
                    "label": "Value",
                    "required": false,
                    "read_only": false,
                    "template": "{{.nested}}",
                    "parameters": []
                }]
            }`,
		},
		{
			name: "template child parameter",
			payload: `{
                "name": "service",
                "template_url": "https://example.test",
                "method": "GET",
                "type": "GEOJSON",
                "format": "json",
                "parameters": [{
                    "kind": "TEMPLATE",
                    "position": 0,
                    "name": "value",
                    "label": "Value",
                    "required": false,
                    "read_only": false,
                    "location": "QUERY",
                    "explode": false,
                    "template": "{{.nested}}",
                    "parameters": [{
                        "kind": "TEMPLATE",
                        "position": 0,
                        "name": "nested",
                        "label": "Nested",
                        "required": false,
                        "read_only": false,
                        "template": "{{.value}}",
                        "parameters": []
                    }]
                }]
            }`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var service gatewayapi.WebServiceDraft
			if err := json.Unmarshal([]byte(test.payload), &service); err == nil {
				t.Fatal("json.Unmarshal() error = nil, want scalar union rejection")
			}
		})
	}
}

func TestWebServiceParameterMappingsPreserveExplode(t *testing.T) {
	documentationURI, err := url.Parse("https://example.test/docs")
	if err != nil {
		t.Fatal(err)
	}
	draft := gatewayapi.WebServiceDraft{
		Type: gatewayapi.HttpWebServiceDraftWebServiceDraft,
		HttpWebServiceDraft: gatewayapi.HttpWebServiceDraft{
			DocumentationURI: gatewayapi.NewOptURI(*documentationURI),
			VisibleOn: gatewayapi.NewOptNilWebServiceVisibleOnArray([]gatewayapi.WebServiceVisibleOn{
				gatewayapi.WebServiceVisibleOnMAP,
				gatewayapi.WebServiceVisibleOnGRAPH,
			}),
			Type:        gatewayapi.HttpWebServiceDraftTypeHTTP,
			TemplateURL: "https://example.test/{?value}",
			Method:      gatewayapi.WebServiceMethodGET,
			Format:      "json",
			Parameters: []gatewayapi.ParameterDraft{gatewayapi.NewStringParameterDraftParameterDraft(gatewayapi.StringParameterDraft{
				Position:       0,
				Name:           "value",
				Label:          "Value",
				Location:       gatewayapi.ParameterLocationQUERY,
				Explode:        true,
				MultipleValues: true,
				Default:        gatewayapi.NewOptNilStringArray([]string{"one", "two"}),
				Enum:           []gatewayapi.StringEnumOption{{Value: "one"}, {Value: "two"}},
			})},
		},
	}
	proto := toProtoWebServiceDraft(draft)
	if proto.GetHttp().GetCommon().GetDocumentationUri() != documentationURI.String() {
		t.Fatalf("protobuf documentation URI = %q, want %q", proto.GetHttp().GetCommon().GetDocumentationUri(), documentationURI)
	}
	parameters := proto.GetHttp().GetParameters()
	if len(parameters) != 1 || !parameters[0].GetExplode() {
		t.Fatalf("protobuf parameters = %+v, want explode parameter", parameters)
	}
	if got := parameters[0].GetString_().Default; !slices.Equal(got, []string{"one", "two"}) {
		t.Fatalf("protobuf default = %v, want [one two]", got)
	}
	visibleValues := proto.GetHttp().GetCommon().VisibleOn
	if len(visibleValues) != 2 || visibleValues[0] != metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_MAP || visibleValues[1] != metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH {
		t.Fatalf("protobuf web-service visible on = %v", visibleValues)
	}

	id := uuid.New()
	value, err := toAPIParameter(&metadatalifecyclev1.Parameter{
		Id:         id.String(),
		ResourceId: id.String(),
		Name:       "value",
		Label:      "Value",
		Location:   metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_QUERY,
		Explode:    true,
		Value: &metadatalifecyclev1.Parameter_String_{String_: &metadatalifecyclev1.StringParameter{
			Default:        []string{"one", "two"},
			Enum:           []*metadatalifecyclev1.StringEnumOption{{Value: "one"}, {Value: "two"}},
			MultipleValues: true,
		}},
	})
	if err != nil {
		t.Fatalf("toAPIParameter() error = %v", err)
	}
	apiParameter, ok := value.GetStringParameter()
	if !ok || !apiParameter.Explode {
		t.Fatal("API parameter Explode = false, want true")
	}
	if defaults, ok := apiParameter.Default.Get(); !ok || !slices.Equal(defaults, []string{"one", "two"}) {
		t.Fatalf("API default = %v, %v; want [one two]", defaults, ok)
	}

	service, err := toAPIWebService(&metadatalifecyclev1.WebService{
		Service: &metadatalifecyclev1.WebService_Http{Http: &metadatalifecyclev1.HTTPWebService{
			Common: &metadatalifecyclev1.WebServiceCommon{
				Id: id.String(), ResourceId: id.String(), DocumentationUri: new(documentationURI.String()),
				VisibleOn: []metadatalifecyclev1.WebServiceVisibleOn{
					metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_MAP,
					metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH,
				},
			},
			TemplateUrl:   "https://example.test",
			Method:        metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET,
			Format:        "json",
			NativeFormats: []*metadatalifecyclev1.NativeFormat{{OutputFormat: "json", Label: "json"}},
		}},
	})
	if err != nil {
		t.Fatalf("toAPIWebService() error = %v", err)
	}
	httpService, ok := service.GetHttpWebService()
	if !ok {
		t.Fatal("expected HTTP service")
	}
	visibleOn, ok := httpService.VisibleOn.Get()
	if !ok || len(visibleOn) != 2 || visibleOn[0] != gatewayapi.WebServiceVisibleOnMAP || visibleOn[1] != gatewayapi.WebServiceVisibleOnGRAPH {
		t.Fatalf("API web-service visible on = %v", visibleOn)
	}
	if uri, ok := httpService.DocumentationURI.Get(); !ok || uri.String() != documentationURI.String() {
		t.Fatalf("API documentation URI = %q, %t; want %q", uri.String(), ok, documentationURI.String())
	}
}

func TestWebServiceMappingRejectsUnspecifiedUpstreamEnums(t *testing.T) {
	id := uuid.New().String()
	base := func() *metadatalifecyclev1.WebService {
		return &metadatalifecyclev1.WebService{
			Service: &metadatalifecyclev1.WebService_Http{Http: &metadatalifecyclev1.HTTPWebService{
				Common: &metadatalifecyclev1.WebServiceCommon{
					Id: id, ResourceId: id,
					VisibleOn: []metadatalifecyclev1.WebServiceVisibleOn{
						metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH,
					},
				},
				TemplateUrl:   "https://example.test",
				Method:        metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET,
				Format:        "json",
				NativeFormats: []*metadatalifecyclev1.NativeFormat{{OutputFormat: "json", Label: "json"}},
			}},
		}
	}

	tests := []struct {
		name   string
		mutate func(*metadatalifecyclev1.WebService)
	}{
		{
			name: "method",
			mutate: func(value *metadatalifecyclev1.WebService) {
				value.GetHttp().Method = metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_UNSPECIFIED
			},
		},
		{
			name: "type",
			mutate: func(value *metadatalifecyclev1.WebService) {
				value.Service = nil
			},
		},
		{
			name: "visible on",
			mutate: func(value *metadatalifecyclev1.WebService) {
				value.GetHttp().Common.VisibleOn = []metadatalifecyclev1.WebServiceVisibleOn{
					metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_UNSPECIFIED,
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := base()
			test.mutate(value)
			if _, err := toAPIWebService(value); err == nil {
				t.Fatal("toAPIWebService() error = nil, want invalid enum error")
			}
		})
	}
}
