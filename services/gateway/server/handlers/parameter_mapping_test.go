package handlers

import (
	"encoding/json"
	"testing"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func TestScalarParameterMappingsOmitTopLevelFields(t *testing.T) {
	draft := gatewayapi.NewStringScalarParameterDraftScalarParameterDraft(gatewayapi.StringScalarParameterDraft{
		Position: 1,
		Name:     gatewayapi.NonBlankString("value"),
		Label:    gatewayapi.NonBlankString("Value"),
		Enum:     []string{"one", "two"},
	})
	proto := toProtoScalarParameterDraft(draft)
	if proto == nil || proto.GetString_() == nil || len(proto.GetString_().GetEnum()) != 2 {
		t.Fatalf("protobuf scalar parameter = %+v, want scalar string parameter", proto)
	}

	apiValue, err := toAPIScalarParameter(&metadatalifecyclev1.ScalarParameter{
		Position: 1,
		Name:     "value",
		Label:    "Value",
		Value: &metadatalifecyclev1.ScalarParameter_String_{String_: &metadatalifecyclev1.ScalarStringParameter{
			Enum: []string{"one", "two"},
		}},
	})
	if err != nil {
		t.Fatalf("toAPIScalarParameter() error = %v", err)
	}
	value, ok := apiValue.GetStringScalarParameter()
	if !ok || len(value.Enum) != 2 {
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
	draft := gatewayapi.WebServiceDraft{
		Name:        gatewayapi.NonBlankString("service"),
		TemplateURL: gatewayapi.NonBlankString("https://example.test/{?value}"),
		Method:      gatewayapi.WebServiceMethodGET,
		Type:        gatewayapi.WebServiceType("GEOJSON"),
		VisibleOn: gatewayapi.NewOptNilWebServiceVisibleOnArray([]gatewayapi.WebServiceVisibleOn{
			gatewayapi.WebServiceVisibleOnMAP,
			gatewayapi.WebServiceVisibleOnGRAPH,
		}),
		Format: gatewayapi.NonBlankString("json"),
		Parameters: []gatewayapi.ParameterDraft{gatewayapi.NewStringParameterDraftParameterDraft(gatewayapi.StringParameterDraft{
			Position:       0,
			Name:           gatewayapi.NonBlankString("value"),
			Label:          gatewayapi.NonBlankString("Value"),
			Location:       gatewayapi.ParameterLocationQUERY,
			Explode:        true,
			MultipleValues: true,
			Enum:           []string{"one", "two"},
		})},
	}
	proto := toProtoWebServiceDraft(draft)
	if len(proto.Parameters) != 1 || !proto.Parameters[0].GetExplode() {
		t.Fatalf("protobuf parameters = %+v, want explode parameter", proto.Parameters)
	}
	if proto.Type != metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_GEOJSON || len(proto.VisibleOn) != 2 || proto.VisibleOn[0] != metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_MAP || proto.VisibleOn[1] != metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH {
		t.Fatalf("protobuf web-service fields = type %q, visible on %v", proto.Type, proto.VisibleOn)
	}

	id := uuid.New()
	value, err := toAPIWebServiceParameter(&metadatalifecyclev1.WebServiceParameter{
		Id: id.String(),
		Parameter: &metadatalifecyclev1.Parameter{
			Name:     "value",
			Label:    "Value",
			Location: metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_QUERY,
			Explode:  true,
			Value: &metadatalifecyclev1.Parameter_String_{String_: &metadatalifecyclev1.StringParameter{
				Enum:           []string{"one", "two"},
				MultipleValues: true,
			}},
		},
	})
	if err != nil {
		t.Fatalf("toAPIWebServiceParameter() error = %v", err)
	}
	apiParameter, ok := value.GetStringParameter()
	if !ok || !apiParameter.Explode {
		t.Fatal("API parameter Explode = false, want true")
	}

	service, err := toAPIWebService(&metadatalifecyclev1.WebService{
		Id:          id.String(),
		Name:        "service",
		TemplateUrl: "https://example.test",
		Method:      metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET,
		Type:        metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_GEOJSON,
		VisibleOn: []metadatalifecyclev1.WebServiceVisibleOn{
			metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_MAP,
			metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH,
		},
		Format: "json",
	})
	if err != nil {
		t.Fatalf("toAPIWebService() error = %v", err)
	}
	visibleOn, ok := service.VisibleOn.Get()
	if !ok || service.Type != gatewayapi.WebServiceType("GEOJSON") || len(visibleOn) != 2 || visibleOn[0] != gatewayapi.WebServiceVisibleOnMAP || visibleOn[1] != gatewayapi.WebServiceVisibleOnGRAPH {
		t.Fatalf("API web-service fields = type %q, visible on %v", service.Type, visibleOn)
	}
}

func TestWebServiceMappingRejectsUnspecifiedUpstreamEnums(t *testing.T) {
	id := uuid.New().String()
	base := func() *metadatalifecyclev1.WebService {
		return &metadatalifecyclev1.WebService{
			Id:          id,
			Name:        "service",
			TemplateUrl: "https://example.test",
			Method:      metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET,
			Type:        metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_GEOJSON,
			VisibleOn: []metadatalifecyclev1.WebServiceVisibleOn{
				metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH,
			},
			Format: "json",
		}
	}

	tests := []struct {
		name   string
		mutate func(*metadatalifecyclev1.WebService)
	}{
		{
			name: "method",
			mutate: func(value *metadatalifecyclev1.WebService) {
				value.Method = metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_UNSPECIFIED
			},
		},
		{
			name: "type",
			mutate: func(value *metadatalifecyclev1.WebService) {
				value.Type = metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_UNSPECIFIED
			},
		},
		{
			name: "visible on",
			mutate: func(value *metadatalifecyclev1.WebService) {
				value.VisibleOn = []metadatalifecyclev1.WebServiceVisibleOn{
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
