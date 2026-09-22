package handlers

import (
	"context"
	"fmt"

	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) RetrieveResourceVersion(ctx context.Context, request *gatewayapi.RetrieveResourceVersionRequest) (*gatewayapi.RetrieveResourceVersionResponse, error) {
	protoRequest, err := toProtoRetrieveResourceVersionRequest(request)
	if err != nil {
		return nil, fmt.Errorf("map retrieval request: %w", err)
	}
	stream, err := h.dataRetriever.RetrieveStream(ctx, protoRequest)
	if err != nil {
		return nil, fmt.Errorf("retrieve resource version: %w", err)
	}
	event, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("retrieve resource version: %w", err)
	}
	return toAPIDryResponse(event.GetHeaders())
}

func toProtoRetrieveResourceVersionRequest(request *gatewayapi.RetrieveResourceVersionRequest) (*dataretrieverv1.RetrieveStreamRequest, error) {
	if request == nil {
		return nil, fmt.Errorf("request is required")
	}

	method, err := toProtoRetrievalHTTPMethod(request.Method)
	if err != nil {
		return nil, err
	}
	mode, err := toProtoRetrieveMode(request.Mode)
	if err != nil {
		return nil, err
	}
	result := &dataretrieverv1.RetrieveStreamRequest{
		ResourceId:  request.ResourceID.String(),
		VersionId:   request.VersionID.String(),
		Method:      method,
		Mode:        mode,
		UrlTemplate: request.URLTemplate,
	}
	if request.PathParams.Set {
		result.PathParams = map[string]string(request.PathParams.Value)
	}
	if request.QueryParams.Set {
		result.QueryParams = map[string]string(request.QueryParams.Value)
	}
	if request.Headers.Set {
		result.Headers = map[string]string(request.Headers.Value)
	}
	if request.BearerToken.Set && !request.BearerToken.Null {
		result.BearerToken = request.BearerToken.Value
	}
	if request.Body.Set && !request.Body.Null {
		body, err := toProtoRetrievalRequestBody(request.Body.Value)
		if err != nil {
			return nil, err
		}
		result.Body = body
	}
	return result, nil
}

func toProtoRetrieveMode(mode gatewayapi.OptRetrieveMode) (dataretrieverv1.RetrieveMode, error) {
	if !mode.Set || mode.Value == gatewayapi.RetrieveModeEXECUTE {
		return dataretrieverv1.RetrieveMode_RETRIEVE_MODE_EXECUTE, nil
	}
	if mode.Value == gatewayapi.RetrieveModeDRY {
		return dataretrieverv1.RetrieveMode_RETRIEVE_MODE_DRY, nil
	}
	return dataretrieverv1.RetrieveMode_RETRIEVE_MODE_UNSPECIFIED, fmt.Errorf("mode is invalid")
}

func toProtoRetrievalHTTPMethod(method gatewayapi.RetrievalHTTPMethod) (dataretrieverv1.HTTPMethod, error) {
	switch method {
	case gatewayapi.RetrievalHTTPMethodGET:
		return dataretrieverv1.HTTPMethod_HTTP_METHOD_GET, nil
	case gatewayapi.RetrievalHTTPMethodPOST:
		return dataretrieverv1.HTTPMethod_HTTP_METHOD_POST, nil
	case gatewayapi.RetrievalHTTPMethodPUT:
		return dataretrieverv1.HTTPMethod_HTTP_METHOD_PUT, nil
	case gatewayapi.RetrievalHTTPMethodDELETE:
		return dataretrieverv1.HTTPMethod_HTTP_METHOD_DELETE, nil
	default:
		return dataretrieverv1.HTTPMethod_HTTP_METHOD_UNSPECIFIED, fmt.Errorf("method is invalid")
	}
}

func toProtoRetrievalRequestBody(body gatewayapi.RetrievalRequestBody) (*dataretrieverv1.RequestBody, error) {
	result := &dataretrieverv1.RequestBody{ContentType: string(body.ContentType)}
	hasRaw := body.Raw.Set && !body.Raw.Null
	hasJSON := body.JSONPayload.Set && !body.JSONPayload.Null
	if hasRaw == hasJSON {
		return nil, fmt.Errorf("exactly one of body.raw or body.json_payload is required")
	}
	if hasRaw {
		result.Content = &dataretrieverv1.RequestBody_Raw{Raw: body.Raw.Value}
	} else {
		result.Content = &dataretrieverv1.RequestBody_JsonPayload{JsonPayload: []byte(body.JSONPayload.Value)}
	}
	return result, nil
}

func toAPIDryResponse(headers *dataretrieverv1.RetrieveStreamHeaders) (*gatewayapi.RetrieveResourceVersionResponse, error) {
	if headers == nil || headers.GetCurlCommand() == "" {
		return nil, fmt.Errorf("data-retriever dry response is empty")
	}
	result := &gatewayapi.RetrieveResourceVersionResponse{Mode: gatewayapi.RetrieveModeDRY}
	result.CurlCommand.SetTo(headers.GetCurlCommand())
	return result, nil
}
