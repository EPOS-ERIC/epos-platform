package domain

import (
	"net/textproto"
	"strings"

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
		return invalidField("location", "Location is not supported.", nil)
	}
	if value.Explode && (value.Location != ParameterLocationQuery || !value.Schema.AllowsMultipleValues()) {
		return invalidField("explode", "Explode requires a query parameter with multiple values.", nil)
	}
	if value.Location == ParameterLocationPath && !value.Required {
		return invalidField("required", "Path parameters must be required.", nil)
	}
	if value.Location == ParameterLocationHeader {
		if !httpguts.ValidHeaderFieldName(value.Name) {
			return invalidField("name", "Header name must be a valid HTTP field name.", nil)
		}
		switch textproto.CanonicalMIMEHeaderKey(value.Name) {
		case "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "Proxy-Connection", "Keep-Alive":
			return invalidField("name", "Headers must not override destination or HTTP framing.", nil)
		}
	}
	return nil
}
