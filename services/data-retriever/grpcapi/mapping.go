package grpcapi

import (
	"fmt"

	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toDomainRetrieveRequest(request *dataretrieverv1.RetrieveRequest) (domain.RetrieveRequest, error) {
	if request == nil {
		return domain.RetrieveRequest{}, fmt.Errorf("retrieve request is required")
	}

	method, err := toDomainHTTPMethod(request.GetMethod(), false)
	if err != nil {
		return domain.RetrieveRequest{}, err
	}
	mode, err := toDomainRetrieveMode(request.GetMode())
	if err != nil {
		return domain.RetrieveRequest{}, err
	}

	retrieveRequest := domain.RetrieveRequest{
		Mode:             mode,
		ResourceID:       request.GetResourceId(),
		VersionID:        request.GetVersionId(),
		Method:           method,
		URLTemplate:      request.GetUrlTemplate(),
		PathParams:       request.GetPathParams(),
		QueryParams:      request.GetQueryParams(),
		Headers:          request.GetHeaders(),
		BearerToken:      request.GetBearerToken(),
		OutputParameters: make([]domain.OutputParameter, 0, len(request.GetOutputParameters())),
	}
	for _, parameter := range request.GetOutputParameters() {
		outputParameter, err := toDomainOutputParameter(parameter)
		if err != nil {
			return domain.RetrieveRequest{}, err
		}
		retrieveRequest.OutputParameters = append(retrieveRequest.OutputParameters, outputParameter)
	}
	if body := request.GetBody(); body != nil {
		retrieveRequest.Body = &domain.RequestBody{ContentType: body.GetContentType()}
		switch content := body.GetContent().(type) {
		case *dataretrieverv1.RequestBody_Raw:
			retrieveRequest.Body.Raw = content.Raw
		case *dataretrieverv1.RequestBody_JsonPayload:
			retrieveRequest.Body.JSON = content.JsonPayload
		default:
			return domain.RetrieveRequest{}, fmt.Errorf("body content is required")
		}
	}
	return retrieveRequest, nil
}

func toDomainRetrieveMode(mode dataretrieverv1.RetrieveMode) (domain.RetrieveMode, error) {
	switch mode {
	case dataretrieverv1.RetrieveMode_RETRIEVE_MODE_UNSPECIFIED, dataretrieverv1.RetrieveMode_RETRIEVE_MODE_EXECUTE:
		return domain.RetrieveModeExecute, nil
	case dataretrieverv1.RetrieveMode_RETRIEVE_MODE_DRY:
		return domain.RetrieveModeDry, nil
	default:
		return "", fmt.Errorf("mode is invalid")
	}
}

func toDomainHTTPMethod(method dataretrieverv1.HTTPMethod, allowUnspecified bool) (string, error) {
	switch method {
	case dataretrieverv1.HTTPMethod_HTTP_METHOD_UNSPECIFIED:
		if allowUnspecified {
			return "", nil
		}
		return "", fmt.Errorf("method is required")
	case dataretrieverv1.HTTPMethod_HTTP_METHOD_GET:
		return "GET", nil
	case dataretrieverv1.HTTPMethod_HTTP_METHOD_POST:
		return "POST", nil
	case dataretrieverv1.HTTPMethod_HTTP_METHOD_PUT:
		return "PUT", nil
	case dataretrieverv1.HTTPMethod_HTTP_METHOD_DELETE:
		return "DELETE", nil
	default:
		return "", fmt.Errorf("method is invalid")
	}
}

func toDomainOutputParameter(parameter *dataretrieverv1.OutputParameter) (domain.OutputParameter, error) {
	if parameter == nil {
		return domain.OutputParameter{}, fmt.Errorf("output parameter is required")
	}
	kind, err := toDomainOutputParameterKind(parameter.GetKind())
	if err != nil {
		return domain.OutputParameter{}, err
	}
	outputParameter := domain.OutputParameter{Name: parameter.GetName(), Kind: kind}
	if parameter.Pattern != nil {
		pattern := parameter.GetPattern()
		outputParameter.Pattern = &pattern
	}
	return outputParameter, nil
}

func toDomainOutputParameterKind(kind dataretrieverv1.OutputParameterKind) (string, error) {
	switch kind {
	case dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_STRING:
		return "STRING", nil
	case dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_BOOLEAN:
		return "BOOLEAN", nil
	case dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_INTEGER:
		return "INTEGER", nil
	case dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_FLOAT:
		return "FLOAT", nil
	case dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_DATETIME:
		return "DATETIME", nil
	case dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_JSON:
		return "JSON", nil
	case dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_BINARY:
		return "BINARY", nil
	default:
		return "", fmt.Errorf("output parameter kind is invalid")
	}
}

func toProtoRetrievalRecord(record repository.RetrievalRecord) *dataretrieverv1.RetrievalRecord {
	return &dataretrieverv1.RetrievalRecord{
		Id:               record.ID,
		Method:           record.Method,
		UrlTemplate:      record.URLTemplate,
		GeneratedUrl:     record.GeneratedURL,
		HttpStatus:       int32(record.HTTPStatus),
		ContentType:      record.ContentType,
		IsBinary:         record.IsBinary,
		PayloadHash:      record.PayloadHash,
		PayloadSizeBytes: int64(record.PayloadSize),
		UsedAuth:         record.UsedAuth,
		FetchedAt:        timestamppb.New(record.FetchedAt),
		CreatedAt:        timestamppb.New(record.CreatedAt),
		RequestBodyText:  record.RequestBodyText,
		ResourceId:       record.ResourceID,
		VersionId:        record.VersionID,
		OutputParameters: toProtoOutputParameters(record.OutputParameters),
	}
}

func toProtoOutputParameters(parameters []domain.OutputParameter) []*dataretrieverv1.OutputParameter {
	output := make([]*dataretrieverv1.OutputParameter, 0, len(parameters))
	for _, parameter := range parameters {
		kind, ok := toProtoOutputParameterKind(parameter.Kind)
		if !ok {
			continue
		}
		outputParameter := &dataretrieverv1.OutputParameter{Name: parameter.Name, Kind: kind}
		if parameter.Pattern != nil {
			outputParameter.Pattern = parameter.Pattern
		}
		output = append(output, outputParameter)
	}
	return output
}

func toProtoOutputParameterKind(kind string) (dataretrieverv1.OutputParameterKind, bool) {
	switch kind {
	case "STRING":
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_STRING, true
	case "BOOLEAN":
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_BOOLEAN, true
	case "INTEGER":
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_INTEGER, true
	case "FLOAT":
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_FLOAT, true
	case "DATETIME":
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_DATETIME, true
	case "JSON":
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_JSON, true
	case "BINARY":
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_BINARY, true
	default:
		return dataretrieverv1.OutputParameterKind_OUTPUT_PARAMETER_KIND_UNSPECIFIED, false
	}
}
