package handlers

import (
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func toProtoRetrieveResourceVersionRequest(request *gatewayapi.RetrieveResourceVersionRequest) (*dataretrieverv1.RetrieveStreamRequest, error) {
	if request == nil {
		return nil, apperror.Field("request", "The request is invalid.", errors.New("request is required"))
	}

	method, err := toProtoRetrievalHTTPMethod(request.Method)
	if err != nil {
		return nil, apperror.Field("method", "Value is invalid.", err)
	}
	mode, err := toProtoRetrieveMode(request.Mode)
	if err != nil {
		return nil, apperror.Field("mode", "Value is invalid.", err)
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
		return nil, apperror.Field("body", "Provide exactly one of raw or json_payload.", errors.New("request body must have exactly one content value"))
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
		return nil, apperror.Default(apperror.CodeInternal)
	}
	return &gatewayapi.RetrieveResourceVersionResponse{
		Mode:        gatewayapi.RetrieveResourceVersionResponseModeDRY,
		CurlCommand: gatewayapi.NewOptString(headers.GetCurlCommand()),
	}, nil
}
