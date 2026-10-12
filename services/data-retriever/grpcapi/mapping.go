package grpcapi

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	"github.com/EPOS-ERIC/epos-platform/services/data-retriever/domain"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toDomainRetrievalInput(request *dataretrieverv1.RetrieveStreamRequest) (domain.RetrievalInput, error) {
	if request == nil {
		return domain.RetrievalInput{}, apperror.Field("request", "Request is required.", nil)
	}
	method, err := toDomainHTTPMethod(request.GetMethod())
	if err != nil {
		return domain.RetrievalInput{}, err
	}
	mode, err := toDomainRetrieveMode(request.GetMode())
	if err != nil {
		return domain.RetrievalInput{}, err
	}
	resourceID, err := uuid.Parse(request.GetResourceId())
	if err != nil {
		return domain.RetrievalInput{}, apperror.Field("resource_id", "Value is invalid.", err)
	}
	versionID, err := uuid.Parse(request.GetVersionId())
	if err != nil {
		return domain.RetrievalInput{}, apperror.Field("version_id", "Value is invalid.", err)
	}
	retrievalInput := domain.RetrievalInput{
		Mode:        mode,
		ResourceID:  resourceID,
		VersionID:   versionID,
		Method:      method,
		URLTemplate: request.GetUrlTemplate(),
		PathParams:  request.GetPathParams(),
		QueryParams: request.GetQueryParams(),
		Headers:     request.GetHeaders(),
		BearerToken: request.GetBearerToken(),
	}
	if body := request.GetBody(); body != nil {
		retrievalInput.Body = &domain.RequestBody{ContentType: body.GetContentType()}
		switch content := body.GetContent().(type) {
		case *dataretrieverv1.RequestBody_Raw:
			retrievalInput.Body.Raw = content.Raw
		case *dataretrieverv1.RequestBody_JsonPayload:
			retrievalInput.Body.JSON = content.JsonPayload
		default:
			return domain.RetrievalInput{}, apperror.Field("body", "Value is invalid.", fmt.Errorf("body content is required"))
		}
	}
	return retrievalInput, nil
}

func toDomainRetrieveMode(mode dataretrieverv1.RetrieveMode) (domain.RetrieveMode, error) {
	switch mode {
	case dataretrieverv1.RetrieveMode_RETRIEVE_MODE_UNSPECIFIED:
		return "", nil
	case dataretrieverv1.RetrieveMode_RETRIEVE_MODE_EXECUTE:
		return domain.RetrieveModeExecute, nil
	case dataretrieverv1.RetrieveMode_RETRIEVE_MODE_DRY:
		return domain.RetrieveModeDry, nil
	default:
		return "", apperror.Field("mode", "Value is invalid.", fmt.Errorf("mode is invalid"))
	}
}

func toDomainHTTPMethod(method dataretrieverv1.HTTPMethod) (string, error) {
	switch method {
	case dataretrieverv1.HTTPMethod_HTTP_METHOD_UNSPECIFIED:
		return "", nil
	case dataretrieverv1.HTTPMethod_HTTP_METHOD_GET:
		return "GET", nil
	case dataretrieverv1.HTTPMethod_HTTP_METHOD_POST:
		return "POST", nil
	case dataretrieverv1.HTTPMethod_HTTP_METHOD_PUT:
		return "PUT", nil
	case dataretrieverv1.HTTPMethod_HTTP_METHOD_DELETE:
		return "DELETE", nil
	default:
		return "", apperror.Field("method", "Value is invalid.", fmt.Errorf("method is invalid"))
	}
}

func toProtoRetrievalRecord(record domain.RetrievalRecord) *dataretrieverv1.RetrievalRecord {
	resourceID := ""
	if record.ResourceID != uuid.Nil {
		resourceID = record.ResourceID.String()
	}
	versionID := ""
	if record.VersionID != uuid.Nil {
		versionID = record.VersionID.String()
	}
	return &dataretrieverv1.RetrievalRecord{
		Id:               record.ID,
		Method:           record.Method,
		UrlTemplate:      record.URLTemplate,
		GeneratedUrl:     record.GeneratedURL,
		HttpStatus:       int32(record.HTTPStatus),
		ContentType:      record.ContentType,
		IsBinary:         record.IsBinary,
		PayloadHash:      record.PayloadHash,
		PayloadSizeBytes: record.PayloadSize,
		UsedAuth:         record.UsedAuth,
		FetchedAt:        timestamppb.New(record.FetchedAt),
		CreatedAt:        timestamppb.New(record.CreatedAt),
		ResourceId:       resourceID,
		VersionId:        versionID,
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
		output = append(output, &dataretrieverv1.OutputParameter{Name: parameter.Name, Kind: kind})
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
