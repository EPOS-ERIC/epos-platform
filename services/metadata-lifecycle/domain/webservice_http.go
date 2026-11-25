package domain

import "github.com/EPOS-ERIC/epos-platform/packages/go/apperror"

type HTTPWebServiceInput struct {
	GenericWebServiceInput
}

func (*HTTPWebServiceInput) ServiceType() WebServiceType {
	return WebServiceTypeHTTP
}

func (w *HTTPWebServiceInput) Normalize() error {
	if w == nil {
		return apperror.Default(apperror.CodeInternal)
	}
	return w.normalize(WebServiceTypeHTTP)
}

type HTTPWebService struct {
	GenericWebService
}

func (*HTTPWebService) ServiceType() WebServiceType {
	return WebServiceTypeHTTP
}

func (w *HTTPWebService) Normalize() error {
	if w == nil {
		return apperror.Default(apperror.CodeInternal)
	}
	if err := w.normalize(WebServiceTypeHTTP); err != nil {
		return internalValidationError(err)
	}
	return nil
}

func (w *HTTPWebService) NativeFormats() []NativeFormat {
	return w.nativeFormats()
}

func (w *HTTPWebService) PrepareAccess(input DistributionAccessInput) (PreparedHTTPRequest, error) {
	if w == nil {
		return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
	}
	parameters, ok := input.Input.(HTTPAccessInput)
	if !ok {
		return PreparedHTTPRequest{}, apperror.Field("type", "Request type does not match service configuration.", nil)
	}
	return w.prepareHTTPRequest(parameters.Parameters, input.OutputFormat)
}
