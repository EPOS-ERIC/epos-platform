package clients

import (
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	dataretrieverv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/data_retriever/v1"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

func fromProtoPreparedDistributionDownload(response *metadatalifecyclev1.PrepareDistributionDownloadResponse) (domain.DownloadTarget, error) {
	if response == nil {
		return domain.DownloadTarget{}, apperror.Default(apperror.CodeInternal)
	}

	resourceID, err := uuid.Parse(response.GetResourceId())
	if err != nil || resourceID == uuid.Nil {
		return domain.DownloadTarget{}, apperror.Default(apperror.CodeInternal)
	}

	versionID, err := uuid.Parse(response.GetVersionId())
	if err != nil || versionID == uuid.Nil {
		return domain.DownloadTarget{}, apperror.Default(apperror.CodeInternal)
	}

	if response.GetUrl() == "" {
		return domain.DownloadTarget{}, apperror.Default(apperror.CodeInternal)
	}

	return domain.DownloadTarget{
		ResourceID: resourceID,
		VersionID:  versionID,
		URL:        response.GetUrl(),
	}, nil
}

func toProtoDistributionAccess(request domain.AccessRequest) (*metadatalifecyclev1.PrepareDistributionAccessRequest, error) {
	var assignments []domain.AccessParameter
	switch input := request.Input.(type) {
	case domain.HTTPAccessInput:
		assignments = input.Parameters
	case domain.DynamicAccessInput:
		assignments = input.Parameters
	case domain.WFSAccessInput:
		assignments = input.Parameters
	default:
		return nil, apperror.Default(apperror.CodeInternal)
	}
	parameters := make([]*metadatalifecyclev1.DistributionAccessParameterInput, len(assignments))
	for i, parameter := range assignments {
		value, err := toProtoAccessParameter(parameter)
		if err != nil {
			return nil, err
		}
		parameters[i] = value
	}
	result := &metadatalifecyclev1.PrepareDistributionAccessRequest{
		DistributionId: request.DistributionID.String(),
		OutputFormat:   request.OutputFormat,
	}
	switch input := request.Input.(type) {
	case domain.HTTPAccessInput:
		result.Input = &metadatalifecyclev1.PrepareDistributionAccessRequest_Http{
			Http: &metadatalifecyclev1.HTTPDistributionAccessInput{Parameters: parameters},
		}
	case domain.DynamicAccessInput:
		result.Input = &metadatalifecyclev1.PrepareDistributionAccessRequest_Dynamic{
			Dynamic: &metadatalifecyclev1.DynamicDistributionAccessInput{
				Longitude:  input.Longitude,
				Latitude:   input.Latitude,
				Parameters: parameters,
			},
		}
	case domain.WFSAccessInput:
		wfs := &metadatalifecyclev1.WFSDistributionAccessInput{
			FeatureType: input.FeatureType,
			ResultLimit: input.ResultLimit,
			Parameters:  parameters,
		}
		if input.BBox != nil {
			wfs.Bbox = &metadatalifecyclev1.WFSBoundingBox{
				West: input.BBox.West, South: input.BBox.South, East: input.BBox.East, North: input.BBox.North,
			}
		}
		result.Input = &metadatalifecyclev1.PrepareDistributionAccessRequest_Wfs{Wfs: wfs}
	}
	return result, nil
}

func toProtoAccessParameter(parameter domain.AccessParameter) (*metadatalifecyclev1.DistributionAccessParameterInput, error) {
	result := &metadatalifecyclev1.DistributionAccessParameterInput{Id: parameter.ID.String()}
	if value, ok := parameter.Value.(domain.AccessScalar); ok {
		scalar, err := toProtoAccessScalar(value)
		if err != nil {
			return nil, err
		}
		result.Value = &metadatalifecyclev1.DistributionAccessParameterInput_Scalar{Scalar: scalar}
		return result, nil
	}
	items, ok := parameter.Value.(domain.AccessArray)
	if !ok || len(items) == 0 {
		return nil, apperror.Default(apperror.CodeInternal)
	}
	values := make([]*metadatalifecyclev1.DistributionAccessScalarValue, len(items))
	for i, item := range items {
		scalar, err := toProtoAccessScalar(item)
		if err != nil {
			return nil, err
		}
		values[i] = scalar
	}
	result.Value = &metadatalifecyclev1.DistributionAccessParameterInput_Array{
		Array: &metadatalifecyclev1.DistributionAccessArrayValue{Values: values},
	}
	return result, nil
}

func toProtoAccessScalar(scalar domain.AccessScalar) (*metadatalifecyclev1.DistributionAccessScalarValue, error) {
	result := &metadatalifecyclev1.DistributionAccessScalarValue{}
	switch value := scalar.(type) {
	case domain.AccessString:
		result.Value = &metadatalifecyclev1.DistributionAccessScalarValue_StringValue{StringValue: string(value)}
	case domain.AccessBoolean:
		result.Value = &metadatalifecyclev1.DistributionAccessScalarValue_BooleanValue{BooleanValue: bool(value)}
	case domain.AccessInteger:
		result.Value = &metadatalifecyclev1.DistributionAccessScalarValue_IntegerValue{IntegerValue: int64(value)}
	case domain.AccessFloat:
		result.Value = &metadatalifecyclev1.DistributionAccessScalarValue_FloatValue{FloatValue: float64(value)}
	default:
		return nil, apperror.Default(apperror.CodeInternal)
	}
	return result, nil
}

func fromProtoPreparedDistributionAccess(response *metadatalifecyclev1.PrepareDistributionAccessResponse) (domain.PreparedRequest, error) {
	if response == nil {
		return domain.PreparedRequest{}, apperror.Default(apperror.CodeInternal)
	}
	resourceID, err := uuid.Parse(response.GetResourceId())
	if err != nil || resourceID == uuid.Nil {
		return domain.PreparedRequest{}, apperror.Default(apperror.CodeInternal)
	}
	versionID, err := uuid.Parse(response.GetVersionId())
	if err != nil || versionID == uuid.Nil {
		return domain.PreparedRequest{}, apperror.Default(apperror.CodeInternal)
	}
	if response.GetUrl() == "" {
		return domain.PreparedRequest{}, apperror.Default(apperror.CodeInternal)
	}
	var method domain.HTTPMethod
	switch response.GetMethod() {
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET:
		method = domain.HTTPMethodGET
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_POST:
		method = domain.HTTPMethodPOST
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_PUT:
		method = domain.HTTPMethodPUT
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_DELETE:
		method = domain.HTTPMethodDELETE
	default:
		return domain.PreparedRequest{}, apperror.Default(apperror.CodeInternal)
	}
	return domain.PreparedRequest{
		ResourceID: resourceID,
		VersionID:  versionID,
		Method:     method,
		URL:        response.GetUrl(),
		Headers:    response.GetHeaders(),
		Body:       response.GetBody(),
	}, nil
}

func fromProtoRetrievalStreamHeaders(response *dataretrieverv1.RetrieveStreamResponse) (int, error) {
	if response == nil || response.GetHeaders() == nil {
		return 0, apperror.Default(apperror.CodeInternal)
	}

	status := int(response.GetHeaders().GetUpstreamStatus())
	if status < http.StatusOK || status > 599 {
		return 0, apperror.Default(apperror.CodeInternal)
	}

	return status, nil
}
