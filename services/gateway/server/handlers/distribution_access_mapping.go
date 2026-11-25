package handlers

import (
	"maps"
	"math"
	"net/url"
	"unicode/utf8"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func fromAPIDistributionAccess(
	request gatewayapi.DistributionAccessRequest,
	distributionID uuid.UUID,
	outputFormat gatewayapi.OptString,
	conversionID gatewayapi.OptUUID,
) (domain.AccessRequest, error) {
	input := domain.AccessRequest{
		DistributionID: distributionID,
	}
	var parameters []gatewayapi.DistributionAccessParameter
	switch request.Type {
	case gatewayapi.HTTPDistributionAccessRequestDistributionAccessRequest:
		parameters = request.HTTPDistributionAccessRequest.Parameters
	case gatewayapi.DynamicDistributionAccessRequestDistributionAccessRequest:
		parameters = request.DynamicDistributionAccessRequest.Parameters
	case gatewayapi.WFSDistributionAccessRequestDistributionAccessRequest:
		parameters = request.WFSDistributionAccessRequest.Parameters
	default:
		return domain.AccessRequest{}, apperror.Field("type", "Request type is not supported.", nil)
	}
	mapped := make([]domain.AccessParameter, len(parameters))

	if value, ok := outputFormat.Get(); ok {
		input.OutputFormat = new(value)
	}

	if value, ok := conversionID.Get(); ok {
		input.ConversionID = new(value)
	}

	for i, parameter := range parameters {
		value, err := fromAPIAccessValue(parameter.Value)
		if err != nil {
			return domain.AccessRequest{}, apperror.Default(apperror.CodeInternal)
		}
		mapped[i] = domain.AccessParameter{ID: parameter.ID, Value: value}
	}
	switch request.Type {
	case gatewayapi.HTTPDistributionAccessRequestDistributionAccessRequest:
		input.Input = domain.HTTPAccessInput{Parameters: mapped}
	case gatewayapi.DynamicDistributionAccessRequestDistributionAccessRequest:
		value := request.DynamicDistributionAccessRequest
		input.Input = domain.DynamicAccessInput{Longitude: value.Longitude, Latitude: value.Latitude, Parameters: mapped}
	case gatewayapi.WFSDistributionAccessRequestDistributionAccessRequest:
		value := request.WFSDistributionAccessRequest
		wfs := domain.WFSAccessInput{Parameters: mapped}
		if selected, ok := value.FeatureType.Get(); ok {
			wfs.FeatureType = new(selected)
		}
		if limit, ok := value.ResultLimit.Get(); ok {
			wfs.ResultLimit = new(limit)
		}
		if value.Bbox != nil {
			if len(value.Bbox) != 4 {
				return domain.AccessRequest{}, apperror.Field("bbox", "Bounds must contain west, south, east and north.", nil)
			}
			wfs.BBox = &domain.WFSBoundingBox{
				West: value.Bbox[0], South: value.Bbox[1], East: value.Bbox[2], North: value.Bbox[3],
			}
		}
		input.Input = wfs
	default:
		return domain.AccessRequest{}, apperror.Field("type", "Request type is not supported.", nil)
	}

	return input, nil
}

func toAPIPreparedDistributionRequest(request domain.PreparedRequest) (gatewayapi.PreparedDistributionRequest, error) {
	endpoint, err := url.Parse(request.URL)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return gatewayapi.PreparedDistributionRequest{}, apperror.Default(apperror.CodeInternal)
	}

	if !utf8.Valid(request.Body) {
		return gatewayapi.PreparedDistributionRequest{}, apperror.Default(apperror.CodeInternal)
	}

	method := gatewayapi.PreparedDistributionRequestMethod(request.Method)
	if err := method.Validate(); err != nil {
		return gatewayapi.PreparedDistributionRequest{}, apperror.Default(apperror.CodeInternal)
	}

	result := gatewayapi.PreparedDistributionRequest{
		Method:  method,
		URL:     *endpoint,
		Headers: maps.Clone(request.Headers),
	}

	var body *string
	if request.Body != nil {
		body = new(string(request.Body))
	}

	result.Body = toAPIRequiredNilString(body)

	return result, nil
}

func fromAPIAccessValue(value gatewayapi.DistributionAccessParameterValue) (domain.AccessValue, error) {
	switch value.Type {
	case gatewayapi.DistributionAccessScalarDistributionAccessParameterValue:
		return fromAPIAccessScalar(value.DistributionAccessScalar)
	case gatewayapi.DistributionAccessScalarArrayDistributionAccessParameterValue:
		items := make(domain.AccessArray, len(value.DistributionAccessScalarArray))
		for i, item := range value.DistributionAccessScalarArray {
			scalar, err := fromAPIAccessScalar(item)
			if err != nil {
				return nil, err
			}
			items[i] = scalar
		}
		return items, nil
	default:
		return nil, apperror.Default(apperror.CodeInternal)
	}
}

func fromAPIAccessScalar(value gatewayapi.DistributionAccessScalar) (domain.AccessScalar, error) {
	switch value.Type {
	case gatewayapi.StringDistributionAccessScalar:
		return domain.AccessString(value.String), nil
	case gatewayapi.BoolDistributionAccessScalar:
		return domain.AccessBoolean(value.Bool), nil
	case gatewayapi.Float64DistributionAccessScalar:
		number := value.Float64
		// MaxInt64 rounds up to 2^63 as float64, so the upper bound is exclusive.
		if math.Trunc(number) == number && number >= math.MinInt64 && number < -float64(math.MinInt64) {
			return domain.AccessInteger(number), nil
		}
		return domain.AccessFloat(number), nil
	default:
		return nil, apperror.Default(apperror.CodeInternal)
	}
}
