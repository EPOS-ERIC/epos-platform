package domain

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

type DynamicWebServiceInput struct {
	GenericWebServiceInput
}

func (*DynamicWebServiceInput) ServiceType() WebServiceType {
	return WebServiceTypeDynamic
}

func (w *DynamicWebServiceInput) Normalize() error {
	if w == nil {
		return apperror.Default(apperror.CodeInternal)
	}
	return w.normalize(WebServiceTypeDynamic)
}

type DynamicWebService struct {
	GenericWebService
}

func (*DynamicWebService) ServiceType() WebServiceType {
	return WebServiceTypeDynamic
}

func (w *DynamicWebService) Normalize() error {
	if w == nil {
		return apperror.Default(apperror.CodeInternal)
	}
	if err := w.normalize(WebServiceTypeDynamic); err != nil {
		return internalValidationError(err)
	}
	return nil
}

func (w *DynamicWebService) NativeFormats() []NativeFormat {
	return w.nativeFormats()
}

func (w *DynamicWebService) PrepareAccess(input DistributionAccessInput) (PreparedHTTPRequest, error) {
	if w == nil {
		return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
	}
	dynamic, ok := input.Input.(DynamicAccessInput)
	if !ok {
		return PreparedHTTPRequest{}, invalidField("type", "Request type does not match service configuration.", nil)
	}
	longitude := w.parameterWithRole(WebServiceParameterRoleLongitude)
	latitude := w.parameterWithRole(WebServiceParameterRoleLatitude)
	if longitude == nil || latitude == nil {
		return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
	}
	parameters := make([]DistributionAccessParameter, 0, len(dynamic.Parameters)+2)
	parameters = append(parameters, dynamic.Parameters...)
	for i, parameter := range dynamic.Parameters {
		if parameter.ID == longitude.ID || parameter.ID == latitude.ID {
			return PreparedHTTPRequest{}, invalidField(fmt.Sprintf("parameters[%d].id", i), "Coordinate values must be supplied through longitude and latitude.", nil)
		}
	}
	for _, coordinate := range []struct {
		name      string
		parameter *Parameter
		value     float64
	}{
		{"longitude", longitude, dynamic.Longitude},
		{"latitude", latitude, dynamic.Latitude},
	} {
		if _, err := resolveAccessValue(coordinate.parameter.Schema, AccessFloat(coordinate.value)); err != nil {
			if apperror.IsCode(err, apperror.CodeValidationFailed) {
				return PreparedHTTPRequest{}, invalidField(coordinate.name, "Coordinate must match the configured constraints.", err)
			}
			return PreparedHTTPRequest{}, err
		}
	}
	parameters = append(parameters,
		DistributionAccessParameter{ID: longitude.ID, Value: AccessFloat(dynamic.Longitude)},
		DistributionAccessParameter{ID: latitude.ID, Value: AccessFloat(dynamic.Latitude)},
	)
	return w.prepareHTTPRequest(parameters, input.OutputFormat)
}
