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
	request *gatewayapi.DistributionAccessRequest,
	distributionID uuid.UUID,
	outputFormat gatewayapi.OptString,
	conversionID gatewayapi.OptUUID,
) (domain.AccessRequest, error) {
	if request == nil {
		return domain.AccessRequest{}, apperror.Field("parameters", "Field is required.", nil)
	}

	input := domain.AccessRequest{
		DistributionID: distributionID,
		Parameters:     make([]domain.AccessParameter, len(request.Parameters)),
	}

	if value, ok := outputFormat.Get(); ok {
		input.OutputFormat = new(value)
	}

	if value, ok := conversionID.Get(); ok {
		input.ConversionID = new(value)
	}

	for i, parameter := range request.Parameters {
		value, err := fromAPIAccessValue(parameter.Value)
		if err != nil {
			return domain.AccessRequest{}, apperror.Default(apperror.CodeInternal)
		}
		input.Parameters[i] = domain.AccessParameter{ID: parameter.ID, Value: value}
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
		Headers: make(gatewayapi.PreparedDistributionRequestHeaders, len(request.Headers)),
	}

	maps.Copy(result.Headers, request.Headers)

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
