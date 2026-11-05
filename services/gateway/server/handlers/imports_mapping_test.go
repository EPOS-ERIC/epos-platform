package handlers

import (
	"net/url"
	"testing"

	semanticaggregatorv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/semantic_aggregator/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func TestToProtoImportResourcesRequest(t *testing.T) {
	baseURI, err := url.Parse("https://example.org/catalog/")
	if err != nil {
		t.Fatal(err)
	}
	referenceURI, err := url.Parse("https://example.org/organization")
	if err != nil {
		t.Fatal(err)
	}
	request, err := toProtoImportResourcesRequest(&gatewayapi.ImportResourcesRequest{
		Turtle:         "@prefix ex: <https://example.org/> .",
		BaseUri:        *baseURI,
		MappingId:      "epos-dcat-common",
		MappingVersion: 1,
		References: []gatewayapi.ImportReferenceBinding{{
			Kind: gatewayapi.ImportReferenceBindingKindOrganization,
			Iri:  *referenceURI,
			ID:   [16]byte{0x12, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x12, 0xd3, 0xa4, 0x56, 0x42, 0x66, 0x14, 0x17, 0x40, 0x01},
		}},
	})
	if err != nil {
		t.Fatalf("toProtoImportResourcesRequest() error = %v", err)
	}
	if got, want := string(request.Turtle), "@prefix ex: <https://example.org/> ."; got != want {
		t.Errorf("Turtle = %q, want %q", got, want)
	}
	if request.GroupId != "" { t.Errorf("GroupId = %q, want empty", request.GroupId) }
	if got, want := request.References[0].Kind, semanticaggregatorv1.ReferenceKind_REFERENCE_KIND_ORGANIZATION; got != want {
		t.Errorf("References[0].Kind = %v, want %v", got, want)
	}
}

func TestToAPIImportResourcesResult(t *testing.T) {
	result, err := toAPIImportResourcesResult(&semanticaggregatorv1.ImportResourcesResponse{
		Imported: []*semanticaggregatorv1.ImportedResource{{
			Subject:    "https://example.org/application",
			ResourceId: "123e4567-e89b-12d3-a456-426614174001",
			VersionId:  "123e4567-e89b-12d3-a456-426614174002",
			Kind:       semanticaggregatorv1.ImportedResourceKind_IMPORTED_RESOURCE_KIND_SOFTWARE_APPLICATION,
		}},
		Failure: &semanticaggregatorv1.ImportFailure{
			Subject:        "https://example.org/source",
			Code:           "UNAVAILABLE",
			OutcomeUnknown: true,
			Message:        "upstream is unavailable",
		},
	})
	if err != nil {
		t.Fatalf("toAPIImportResourcesResult() error = %v", err)
	}
	if got, want := result.Imported[0].Kind, gatewayapi.ImportedResourceKindSoftwareApplication; got != want {
		t.Errorf("Imported[0].Kind = %q, want %q", got, want)
	}
	if !result.Failure.IsSet() {
		t.Fatal("Failure is not set")
	}
	if result.Failure.Value.Phase.IsSet() {
		t.Error("Failure.Phase is set for unspecified protobuf phase")
	}
	if !result.Failure.Value.Message.IsSet() || result.Failure.Value.Message.Value != "upstream is unavailable" {
		t.Errorf("Failure.Message = %#v, want upstream message", result.Failure.Value.Message)
	}
}
