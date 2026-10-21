package handlers

import (
	"math"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func fromAPIDistributionAccess(request *gatewayapi.DistributionAccessRequest, params gatewayapi.AccessDistributionParams) (domain.AccessRequest, error) {
	if request == nil {
		return domain.AccessRequest{}, apperror.Field("parameters", "Field is required.", nil)
	}

	input := domain.AccessRequest{
		DistributionID: params.DistributionId,
		Parameters:     make([]domain.AccessParameter, len(request.Parameters)),
	}

	if value, ok := params.OutputFormat.Get(); ok {
		input.OutputFormat = new(value)
	}

	if value, ok := params.ConversionID.Get(); ok {
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

func fromAPIAccessValue(value gatewayapi.DistributionAccessParameterValue) (domain.AccessValue, error) {
	switch value.Type {
	case gatewayapi.DistributionAccessScalarDistributionAccessParameterValue:
		scalar, err := fromAPIAccessScalar(value.DistributionAccessScalar)
		if err != nil {
			return domain.AccessValue{}, err
		}
		return domain.AccessValue{Scalar: &scalar}, nil
	case gatewayapi.DistributionAccessScalarArrayDistributionAccessParameterValue:
		items := make([]domain.AccessScalar, len(value.DistributionAccessScalarArray))
		for i, item := range value.DistributionAccessScalarArray {
			scalar, err := fromAPIAccessScalar(item)
			if err != nil {
				return domain.AccessValue{}, err
			}
			items[i] = scalar
		}
		return domain.AccessValue{Array: items}, nil
	default:
		return domain.AccessValue{}, apperror.Default(apperror.CodeInternal)
	}
}

func fromAPIAccessScalar(value gatewayapi.DistributionAccessScalar) (domain.AccessScalar, error) {
	switch value.Type {
	case gatewayapi.StringDistributionAccessScalar:
		return domain.AccessScalar{Kind: domain.AccessScalarString, String: value.String}, nil
	case gatewayapi.BoolDistributionAccessScalar:
		return domain.AccessScalar{Kind: domain.AccessScalarBoolean, Boolean: value.Bool}, nil
	case gatewayapi.Float64DistributionAccessScalar:
		number := value.Float64
		// MaxInt64 rounds up to 2^63 as float64, so the upper bound is exclusive.
		if math.Trunc(number) == number && number >= math.MinInt64 && number < -float64(math.MinInt64) {
			return domain.AccessScalar{Kind: domain.AccessScalarInteger, Integer: int64(number)}, nil
		}
		return domain.AccessScalar{Kind: domain.AccessScalarFloat, Float: number}, nil
	default:
		return domain.AccessScalar{}, apperror.Default(apperror.CodeInternal)
	}
}
