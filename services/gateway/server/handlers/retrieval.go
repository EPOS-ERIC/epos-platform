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
	response, err := h.dataRetriever.Retrieve(ctx, protoRequest)
	if err != nil {
		return nil, fmt.Errorf("retrieve resource version: %w", err)
	}
	return toAPIRetrieveResourceVersionResponse(response)
}

func toProtoRetrieveResourceVersionRequest(request *gatewayapi.RetrieveResourceVersionRequest) (*dataretrieverv1.RetrieveRequest, error) {
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
	result := &dataretrieverv1.RetrieveRequest{
		ResourceId:       request.ResourceID.String(),
		VersionId:        request.VersionID.String(),
		Method:           method,
		Mode:             mode,
		UrlTemplate:      request.URLTemplate,
		OutputParameters: make([]*dataretrieverv1.OutputParameter, 0, len(request.OutputParameters)),
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
	for _, parameter := range request.OutputParameters {
		outputParameter, err := toProtoRetrievalOutputParameter(parameter)
		if err != nil {
			return nil, err
		}
		result.OutputParameters = append(result.OutputParameters, outputParameter)
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

func toProtoRetrievalOutputParameter(parameter gatewayapi.RetrievalOutputParameter) (*dataretrieverv1.OutputParameter, error) {
	kind, err := toProtoRetrievalOutputParameterKind(parameter.Kind)
	if err != nil {
		return nil, err
	}
	result := &dataretrieverv1.OutputParameter{Name: string(parameter.Name), Kind: kind}
	if parameter.Pattern.Set && !parameter.Pattern.Null {
		pattern := parameter.Pattern.Value
		result.Pattern = &pattern
	}
	return result, nil
}

func toProtoRetrievalOutputParameterKind(kind gatewayapi.RetrievalOutputParameterKind) (dataretrieverv1.OutputParameterKind, error) {
	switch kind {
	case gatewayapi.RetrievalOutputParameterKindSTRING:
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_STRING, nil
	case gatewayapi.RetrievalOutputParameterKindBOOLEAN:
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_BOOLEAN, nil
	case gatewayapi.RetrievalOutputParameterKindINTEGER:
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_INTEGER, nil
	case gatewayapi.RetrievalOutputParameterKindFLOAT:
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_FLOAT, nil
	case gatewayapi.RetrievalOutputParameterKindDATETIME:
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_DATETIME, nil
	case gatewayapi.RetrievalOutputParameterKindJSON:
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_JSON, nil
	case gatewayapi.RetrievalOutputParameterKindBINARY:
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_BINARY, nil
	default:
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_UNSPECIFIED, fmt.Errorf("output parameter kind is invalid")
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

func toAPIRetrieveResourceVersionResponse(response *dataretrieverv1.RetrieveResponse) (*gatewayapi.RetrieveResourceVersionResponse, error) {
	if response == nil {
		return nil, fmt.Errorf("data-retriever response is empty")
	}
	if command := response.GetCurlCommand(); command != "" {
		result := &gatewayapi.RetrieveResourceVersionResponse{Mode: gatewayapi.RetrieveModeDRY}
		result.CurlCommand.SetTo(command)
		return result, nil
	}
	if response.GetRecord() == nil {
		return nil, fmt.Errorf("data-retriever execution response has no record")
	}

	result := &gatewayapi.RetrieveResourceVersionResponse{Mode: gatewayapi.RetrieveModeEXECUTE}
	result.RetrievalID.SetTo(response.GetRecord().GetId())
	result.UpstreamStatus.SetTo(response.GetUpstreamStatus())
	result.ContentType.SetTo(response.GetContentType())
	switch payload := response.Result.(type) {
	case *dataretrieverv1.RetrieveResponse_InlinePayload:
		result.InlinePayload.SetTo(payload.InlinePayload)
	case *dataretrieverv1.RetrieveResponse_RedirectUrl:
		result.RedirectURL.SetTo(payload.RedirectUrl)
	default:
		return nil, fmt.Errorf("data-retriever execution response has no result")
	}
	return result, nil
}
