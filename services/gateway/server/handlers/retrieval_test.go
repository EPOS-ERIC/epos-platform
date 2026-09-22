package handlers

import (
	"testing"

	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func TestToProtoRetrieveResourceVersionRequestDefaultsToExecute(t *testing.T) {
	request, err := toProtoRetrieveResourceVersionRequest(&gatewayapi.RetrieveResourceVersionRequest{
		ResourceID:  uuid.New(),
		VersionID:   uuid.New(),
		Method:      gatewayapi.RetrievalHTTPMethodGET,
		URLTemplate: "https://example.test/files/{id}",
		PathParams: gatewayapi.NewOptRetrieveResourceVersionRequestPathParams(
			gatewayapi.RetrieveResourceVersionRequestPathParams{"id": "one"},
		),
	})
	if err != nil {
		t.Fatalf("toProtoRetrieveResourceVersionRequest() error = %v", err)
	}
	if request.GetMode() != dataretrieverv1.RetrieveMode_RETRIEVE_MODE_EXECUTE {
		t.Fatalf("mode = %s, want EXECUTE", request.GetMode())
	}
	if request.GetResourceId() == "" || request.GetVersionId() == "" {
		t.Fatalf("resource/version IDs = %q/%q", request.GetResourceId(), request.GetVersionId())
	}
}

func TestToAPIRetrieveResourceVersionDryResponse(t *testing.T) {
	response, err := toAPIRetrieveResourceVersionResponse(&dataretrieverv1.RetrieveResponse{
		Result: &dataretrieverv1.RetrieveResponse_CurlCommand{CurlCommand: "curl --url 'https://example.test'"},
	})
	if err != nil {
		t.Fatalf("toAPIRetrieveResourceVersionResponse() error = %v", err)
	}
	if response.Mode != gatewayapi.RetrieveModeDRY || !response.CurlCommand.Set || response.CurlCommand.Value == "" {
		t.Fatalf("dry response = %+v", response)
	}
	if response.RetrievalID.Set || response.InlinePayload.Set || response.RedirectURL.Set {
		t.Fatalf("dry response includes execution fields: %+v", response)
	}
}
