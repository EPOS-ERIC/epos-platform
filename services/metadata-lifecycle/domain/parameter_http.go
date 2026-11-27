package domain

import (
	"net/textproto"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"golang.org/x/net/http/httpguts"
)

// Normalize validates a caller-controlled top-level parameter.
func (value *WebServiceParameterInput) Normalize() error {
	parameter, err := parseParameter(value.ParameterInput, true, true)
	if err != nil {
		return err
	}
	value.ParameterInput = parameter
	value.Location = ParameterLocation(strings.ToUpper(strings.TrimSpace(string(value.Location))))
	if !value.Location.Valid() {
		return apperror.Field("location", "Location is not supported.", nil)
	}
	if value.Explode && (value.Location != ParameterLocationQuery || !value.Schema.AllowsMultipleValues()) {
		return apperror.Field("explode", "Explode requires a query parameter with multiple values.", nil)
	}
	if value.Location == ParameterLocationPath && !value.Required {
		return apperror.Field("required", "Path parameters must be required.", nil)
	}
	if value.Location == ParameterLocationHeader {
		if !httpguts.ValidHeaderFieldName(value.Name) {
			return apperror.Field("name", "Header name must be a valid HTTP field name.", nil)
		}
		switch textproto.CanonicalMIMEHeaderKey(value.Name) {
		case "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "Proxy-Connection", "Keep-Alive":
			return apperror.Field("name", "Headers must not override destination or HTTP framing.", nil)
		}
	}
	return nil
}

func (value WebServiceParameterInput) validateHeaderDefault() error {
	if value.Location != ParameterLocationHeader {
		return nil
	}

	defaultValue := value.Schema.DefaultValue()
	if defaultValue == nil {
		return nil
	}

	items, err := resolveAccessValue(value.Schema, defaultValue)
	if err != nil {
		return apperror.Field("default", "Default must match parameter type and constraints.", err)
	}

	for _, item := range items {
		if !httpguts.ValidHeaderFieldValue(item.Text()) {
			return apperror.Field("default", "Default must be a valid HTTP header value.", nil)
		}
	}

	return nil
}
