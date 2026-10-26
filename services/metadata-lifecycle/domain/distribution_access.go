package domain

import (
	"fmt"
	"net/textproto"
	"net/url"
	"slices"
	"strings"
	"text/template"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
	"golang.org/x/net/http/httpguts"
)

// PrepareAccess resolves caller values against validated metadata and renders a request.
// It applies defaults and native format selection, renders active templates and
// serializes path, query, header and body values without executing HTTP requests.
// Unresolved optional values are omitted; a nil body remains distinct from an
// explicitly empty rendered body. Content-Type is never inferred.
func (service WebService) PrepareAccess(input DistributionAccessInput) (PreparedHTTPRequest, error) {
	configuration, ok := service.Configuration.(GenericWebServiceConfiguration)
	if !ok {
		return PreparedHTTPRequest{}, invalidField("type", "Prepared HTTP access is not supported for this service configuration.", nil)
	}
	return configuration.PrepareAccess(input)
}

func (service GenericWebServiceConfiguration) PrepareAccess(input DistributionAccessInput) (PreparedHTTPRequest, error) {
	values, err := service.resolveAccessValues(input)
	if err != nil {
		return PreparedHTTPRequest{}, err
	}

	formatParameterID, formatValues, err := service.selectAccessFormat(input.OutputFormat)
	if err != nil {
		return PreparedHTTPRequest{}, err
	}
	if formatValues != nil {
		values.resolved[formatParameterID] = formatValues
	}

	// Render parameter templates and serialize values once, before placing them in the request.
	parameterText := make(map[uuid.UUID][]string, len(service.Parameters))
	for _, parameter := range service.Parameters {
		items := values.resolved[parameter.ID]
		if schema, ok := parameter.Schema.(TemplateSchema); ok {
			rendered, err := values.renderTemplate(schema.Template, schema.Parameters, parameter.Required)
			if err != nil {
				return PreparedHTTPRequest{}, apperror.PrefixTarget(err, "parameters")
			}

			items = nil
			if rendered != nil {
				items = AccessArray{AccessString(*rendered)}
			}
		}

		if items == nil {
			if parameter.Required {
				return PreparedHTTPRequest{}, invalidField("parameters", fmt.Sprintf("Required parameter %q (%s) is missing.", parameter.Name, parameter.ID), nil)
			}

			continue
		}

		text := make([]string, len(items))
		for i, item := range items {
			if item == nil {
				return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
			}

			text[i] = item.Text()
		}

		if !parameter.Location.Valid() {
			return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
		}

		parameterText[parameter.ID] = text
	}

	// Validate the template components so substitution cannot change the host or query.
	templateURL, err := url.Parse(service.TemplateURL)
	if err != nil || templateURL.Host == "" || (templateURL.Scheme != "http" && templateURL.Scheme != "https") || templateURL.Fragment != "" {
		return PreparedHTTPRequest{}, invalidField("template_url", "URL must be an absolute HTTP(S) URL without a fragment.", err)
	}

	if pathPlaceholderPattern.MatchString(templateURL.Host) || pathPlaceholderPattern.MatchString(templateURL.RawQuery) {
		return PreparedHTTPRequest{}, invalidField("template_url", "Placeholders are only supported in the URL path.", nil)
	}

	// Render path placeholders in one pass, preserving existing URL escapes.
	var pathReplacements []string
	for _, parameter := range service.Parameters {
		text := parameterText[parameter.ID]
		if parameter.Location != ParameterLocationPath || text == nil {
			continue
		}

		for i := range text {
			text[i] = url.PathEscape(text[i])
		}

		placeholder := "{" + parameter.Name + "}"
		if !strings.Contains(templateURL.Path, placeholder) {
			return PreparedHTTPRequest{}, invalidField("template_url", "Path parameter must have a placeholder in the URL path.", nil)
		}

		pathReplacements = append(pathReplacements, placeholder, strings.Join(text, ","))
	}

	renderedURL := strings.NewReplacer(pathReplacements...).Replace(service.TemplateURL)

	// Parse the rendered URL before adding query parameters.
	endpoint, err := url.Parse(renderedURL)
	if err != nil {
		return PreparedHTTPRequest{}, invalidField("template_url", "URL must be valid.", err)
	}

	// Append escaped query values, retaining existing query text and repeated keys.
	for _, parameter := range service.Parameters {
		text := parameterText[parameter.ID]
		if parameter.Location != ParameterLocationQuery || text == nil {
			continue
		}

		for i := range text {
			text[i] = url.QueryEscape(text[i])
		}

		if !parameter.Explode {
			text = []string{strings.Join(text, ",")}
		}

		for _, item := range text {
			if endpoint.RawQuery != "" {
				endpoint.RawQuery += "&"
			}

			endpoint.RawQuery += url.QueryEscape(parameter.Name) + "=" + item
		}
	}

	// Build headers independently of URL rendering.
	request := PreparedHTTPRequest{
		Method:  service.Method,
		URL:     endpoint.String(),
		Headers: make(map[string]string),
	}

	for _, parameter := range service.Parameters {
		text := parameterText[parameter.ID]
		if parameter.Location != ParameterLocationHeader || text == nil {
			continue
		}

		value := strings.Join(text, ",")
		if !httpguts.ValidHeaderFieldValue(value) {
			return PreparedHTTPRequest{}, invalidField("headers", "Header value must be a valid HTTP field value.", nil)
		}

		request.Headers[textproto.CanonicalMIMEHeaderKey(parameter.Name)] = value
	}

	// Render the body last, keeping an omitted body distinct from an empty one.
	if service.Body != nil {
		if service.Method == HTTPMethodGet || service.Method == HTTPMethodDelete {
			return PreparedHTTPRequest{}, invalidField("body", "Request bodies are not supported for GET or DELETE.", nil)
		}

		body, err := values.renderTemplate(service.Body.Template, service.Body.Parameters, service.Body.Required)
		if err != nil {
			return PreparedHTTPRequest{}, apperror.PrefixTarget(err, "body")
		}

		if body != nil {
			request.Body = []byte(*body)
		}
	}

	return request, nil
}

// selectAccessFormat checks that output_format is supported and returns the upstream
// format parameter UUID and validated values, without changing resolved values.
// It returns uuid.Nil and nil values when no format is selected or when the service
// supports that format without a parameter.
func (service GenericWebServiceConfiguration) selectAccessFormat(selected *string) (uuid.UUID, AccessArray, error) {
	if selected == nil {
		return uuid.Nil, nil, nil
	}

	formats := service.NativeFormats()
	if !slices.ContainsFunc(formats, func(format NativeFormat) bool { return format.OutputFormat == *selected }) {
		return uuid.Nil, nil, invalidField("output_format", "Select a native output format for this distribution.", nil)
	}

	for _, parameter := range service.Parameters {
		if parameter.Role != nil && *parameter.Role == WebServiceParameterRoleFormat {
			resolved, err := resolveAccessValue(parameter.Schema, AccessString(*selected))
			if err != nil {
				return uuid.Nil, nil, internalValidationError(err)
			}

			return parameter.ID, resolved, nil
		}
	}
	return uuid.Nil, nil, nil
}

// renderTemplate renders a parameter or body template with typed child values
// using text/template, without JSON escaping. Required templates reject missing
// required children and missing keys evaluated during execution; optional children
// in unexecuted branches may be absent. Optional templates require every
// declared child to resolve: unresolved templates are omitted when no children
// were supplied, otherwise preparation fails. A nil result means omitted; a
// non-nil result may contain an empty string.
func (values accessValues) renderTemplate(source string, parameters []Parameter, required bool) (*string, error) {
	data := make(map[string]AccessScalar, len(parameters))
	missing, supplied := false, false

	for _, parameter := range parameters {
		items := values.resolved[parameter.ID]
		if items == nil {
			missing = true
		} else {
			data[parameter.Name] = items[0]
		}

		supplied = supplied || values.supplied[parameter.ID]
	}

	if !required && missing {
		if supplied {
			return nil, invalidField("parameters", "Supply all children of an optional template when supplying any child.", nil)
		}

		return nil, nil
	}

	if required {
		for _, parameter := range parameters {
			if parameter.Required && values.resolved[parameter.ID] == nil {
				return nil, invalidField("parameters", fmt.Sprintf("Required parameter %q (%s) is missing.", parameter.Name, parameter.ID), nil)
			}
		}
	}

	parsed, err := template.New("request").Option("missingkey=error").Parse(source)
	if err != nil {
		return nil, internalValidationError(err)
	}

	var rendered strings.Builder
	if err := parsed.Execute(&rendered, data); err != nil {
		return nil, invalidField("parameters", "Template could not be rendered with supplied values.", err)
	}

	return new(rendered.String()), nil
}
