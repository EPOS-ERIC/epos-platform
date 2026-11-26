package domain

import (
	"errors"
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

// ############### INPUT ###############

type GenericWebServiceInput struct {
	WebServiceCommonInput
	TemplateURL string
	Method      HTTPMethod
	Body        *RequestBodyInput
	Parameters  []WebServiceParameterInput
	Format      string
}

func (w *GenericWebServiceInput) normalize(serviceType WebServiceType) error {
	if err := w.Normalize(); err != nil {
		return err
	}
	w.TemplateURL = strings.TrimSpace(w.TemplateURL)
	w.Format = strings.TrimSpace(w.Format)
	w.Method = HTTPMethod(strings.ToUpper(strings.TrimSpace(string(w.Method))))
	if w.TemplateURL == "" {
		return apperror.Field("template_url", "Template URL must not be blank.", nil)
	}
	endpoint, err := url.Parse(w.TemplateURL)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Fragment != "" {
		return apperror.Field("template_url", "URL must be an absolute HTTP(S) URL without a fragment.", err)
	}

	if pathPlaceholderPattern.MatchString(endpoint.Host) || pathPlaceholderPattern.MatchString(endpoint.RawQuery) {
		return apperror.Field("template_url", "Placeholders are only supported in the URL path.", nil)
	}

	if w.Format == "" {
		return apperror.Field("format", "Format must not be blank.", nil)
	}
	if !w.Method.Valid() {
		return apperror.Field("method", "Method is not supported.", nil)
	}
	parameters, err := parseWebServiceParameterInputs(w.Parameters, "parameters")
	if err != nil {
		return err
	}
	w.Parameters = parameters
	var bodyParameters []ParameterInput
	if w.Body != nil {
		if err := w.Body.Normalize(); err != nil {
			return err
		}
		bodyParameters = w.Body.Parameters
	}
	if err := validateWebServiceParameterRoles(serviceType, parameters, "parameters", bodyParameters); err != nil {
		return err
	}
	if err := validateMethodBody(w.Method, w.Body); err != nil {
		return apperror.Field("body_template", "This method does not support a request body.", err)
	}
	if err := validatePathParameters(w.TemplateURL, pathParameterNames(parameters)); err != nil {
		return apperror.Field("template_url", "Template URL placeholders must match declared path parameters.", err)
	}
	return nil
}

// ############### END INPUT ###############

type GenericWebService struct {
	WebServiceCommon
	TemplateURL string
	Method      HTTPMethod
	Body        *RequestBody
	Parameters  []WebServiceParameter
	Format      string
}

func (w *GenericWebService) normalize(serviceType WebServiceType) error {
	if err := w.Normalize(); err != nil {
		return err
	}
	w.TemplateURL = strings.TrimSpace(w.TemplateURL)
	w.Format = strings.TrimSpace(w.Format)
	w.Method = HTTPMethod(strings.ToUpper(strings.TrimSpace(string(w.Method))))
	if w.TemplateURL == "" {
		return apperror.Field("template_url", "Template URL must not be blank.", nil)
	}
	if w.Format == "" {
		return apperror.Field("format", "Format must not be blank.", nil)
	}
	if !w.Method.Valid() {
		return apperror.Field("method", "Method is not supported.", nil)
	}
	if err := validatePersistedWebServiceParameters(w.Parameters); err != nil {
		return err
	}
	parameters, err := parseWebServiceParameterInputs(webServiceParameterInputs(w.Parameters), "parameters")
	if err != nil {
		return err
	}
	var bodyInput *RequestBodyInput
	if w.Body != nil {
		if err := validatePersistedParameters(w.Body.Parameters); err != nil {
			return err
		}
		bodyInput = requestBodyInput(w.Body)
		if err := bodyInput.Normalize(); err != nil {
			return err
		}
	}
	var bodyParameters []ParameterInput
	if bodyInput != nil {
		bodyParameters = bodyInput.Parameters
	}
	if err := validateWebServiceParameterRoles(serviceType, parameters, "parameters", bodyParameters); err != nil {
		return err
	}
	if err := validateMethodBody(w.Method, bodyInput); err != nil {
		return apperror.Field("body_template", "This method does not support a request body.", err)
	}
	if err := validatePathParameters(w.TemplateURL, pathParameterNames(parameters)); err != nil {
		return apperror.Field("template_url", "Template URL placeholders must match declared path parameters.", err)
	}
	normalized, err := applyNormalizedWebServiceParameters(w.Parameters, parameters)
	if err != nil {
		return err
	}
	w.Parameters = normalized
	if bodyInput != nil {
		parameters, err := applyNormalizedParameters(w.Body.Parameters, bodyInput.Parameters)
		if err != nil {
			return err
		}
		w.Body.Required = *bodyInput.Required
		w.Body.ContentType = bodyInput.ContentType
		w.Body.Template = bodyInput.Template
		w.Body.Parameters = parameters
	}
	return nil
}

func (w *GenericWebService) nativeFormats() []NativeFormat {
	var formatSchema *StringSchema

	parameter := w.parameterWithRole(WebServiceParameterRoleFormat)
	if parameter != nil {
		schema := parameter.Schema.(StringSchema)
		formatSchema = &schema
	}

	if formatSchema == nil {
		return []NativeFormat{{
			OutputFormat: w.Format,
			Label:        w.Format,
		}}
	}

	if len(formatSchema.Enum) == 0 {
		return []NativeFormat{{
			OutputFormat: formatSchema.Default[0],
			Label:        formatSchema.Default[0],
		}}
	}

	result := make([]NativeFormat, 0, len(formatSchema.Enum))
	for _, option := range formatSchema.Enum {
		label := option.Value
		if option.Label != nil {
			label = *option.Label
		}
		result = append(result, NativeFormat{
			OutputFormat: option.Value,
			Label:        label,
		})
	}

	return result
}

func (w GenericWebService) parameterWithRole(role WebServiceParameterRole) *Parameter {
	for i := range w.Parameters {
		parameter := &w.Parameters[i]
		if parameter.Role != nil && *parameter.Role == role {
			return &parameter.Parameter
		}

		if template, ok := parameter.Schema.(TemplateSchema); ok {
			for j := range template.Parameters {
				if template.Parameters[j].Role != nil && *template.Parameters[j].Role == role {
					return &template.Parameters[j]
				}
			}
		}
	}

	if w.Body != nil {
		for i := range w.Body.Parameters {
			if w.Body.Parameters[i].Role != nil && *w.Body.Parameters[i].Role == role {
				return &w.Body.Parameters[i]
			}
		}
	}

	return nil
}

func (w GenericWebService) prepareHTTPRequest(parameters []DistributionAccessParameter, outputFormat *string) (PreparedHTTPRequest, error) {
	values, err := w.resolveAccessValues(parameters)
	if err != nil {
		return PreparedHTTPRequest{}, err
	}

	formatParameterID, formatValues, err := w.selectAccessFormat(outputFormat)
	if err != nil {
		return PreparedHTTPRequest{}, err
	}
	if formatValues != nil {
		values.resolved[formatParameterID] = formatValues
		values.supplied[formatParameterID] = true
	}

	// Render parameter templates and serialize values once, before placing them in the request.
	parameterText := make(map[uuid.UUID][]string, len(w.Parameters))
	for _, parameter := range w.Parameters {
		items := values.resolved[parameter.ID]
		if schema, ok := parameter.Schema.(TemplateSchema); ok {
			rendered, err := values.renderTemplate(schema.Template, schema.Parameters, parameter.Required)
			if err != nil {
				return PreparedHTTPRequest{}, err
			}

			items = nil
			if rendered != nil {
				items = AccessArray{AccessString(*rendered)}
			}
		}

		if items == nil {
			if parameter.Required {
				return PreparedHTTPRequest{}, apperror.Field("parameters", fmt.Sprintf("Required parameter %q (%s) is missing.", parameter.Name, parameter.ID), nil)
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
	templateURL, err := url.Parse(w.TemplateURL)
	if err != nil || templateURL.Host == "" || (templateURL.Scheme != "http" && templateURL.Scheme != "https") || templateURL.Fragment != "" {
		return PreparedHTTPRequest{}, internalValidationError(err)
	}

	if pathPlaceholderPattern.MatchString(templateURL.Host) || pathPlaceholderPattern.MatchString(templateURL.RawQuery) {
		return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
	}

	// Render path placeholders in one pass, preserving existing URL escapes.
	var pathReplacements []string
	for _, parameter := range w.Parameters {
		text := parameterText[parameter.ID]
		if parameter.Location != ParameterLocationPath || text == nil {
			continue
		}

		for i := range text {
			text[i] = url.PathEscape(text[i])
		}

		placeholder := "{" + parameter.Name + "}"
		if !strings.Contains(templateURL.Path, placeholder) {
			return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
		}

		pathReplacements = append(pathReplacements, placeholder, strings.Join(text, ","))
	}

	renderedURL := strings.NewReplacer(pathReplacements...).Replace(w.TemplateURL)

	// Parse the rendered URL before adding query parameters.
	endpoint, err := url.Parse(renderedURL)
	if err != nil {
		return PreparedHTTPRequest{}, internalValidationError(err)
	}

	// Append escaped query values, retaining existing query text and repeated keys.
	for _, parameter := range w.Parameters {
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
		Method:  w.Method,
		URL:     endpoint.String(),
		Headers: make(map[string]string),
	}

	for _, parameter := range w.Parameters {
		text := parameterText[parameter.ID]
		if parameter.Location != ParameterLocationHeader || text == nil {
			continue
		}

		value := strings.Join(text, ",")
		if !httpguts.ValidHeaderFieldValue(value) {
			return PreparedHTTPRequest{}, apperror.Field("parameters", "Header value must be a valid HTTP field value.", nil)
		}

		request.Headers[textproto.CanonicalMIMEHeaderKey(parameter.Name)] = value
	}

	// Render the body last, keeping an omitted body distinct from an empty one.
	if w.Body != nil {
		if w.Method == HTTPMethodGet || w.Method == HTTPMethodDelete {
			return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
		}

		body, err := values.renderTemplate(w.Body.Template, w.Body.Parameters, w.Body.Required)
		if err != nil {
			return PreparedHTTPRequest{}, err
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
func (w GenericWebService) selectAccessFormat(selected *string) (uuid.UUID, AccessArray, error) {
	if selected == nil {
		return uuid.Nil, nil, nil
	}

	formats := w.nativeFormats()
	if !slices.ContainsFunc(formats, func(format NativeFormat) bool { return format.OutputFormat == *selected }) {
		return uuid.Nil, nil, apperror.Field("output_format", "Select a native output format for this distribution.", nil)
	}

	if parameter := w.parameterWithRole(WebServiceParameterRoleFormat); parameter != nil {
		resolved, err := resolveAccessValue(parameter.Schema, AccessString(*selected))
		if err != nil {
			return uuid.Nil, nil, internalValidationError(err)
		}

		return parameter.ID, resolved, nil
	}

	return uuid.Nil, nil, nil
}

// accessValues tracks validated parameter values and whether the caller supplied them.
// Template activation uses this distinction to reject partially supplied optional templates.
type accessValues struct {
	// resolved contains supplied or defaulted values by parameter UUID; scalars use one-element arrays.
	resolved map[uuid.UUID]AccessArray
	// supplied records explicit caller assignments, excluding metadata defaults.
	supplied map[uuid.UUID]bool
}

// resolveAccessValues validates assignments against top-level, template-child and body-child
// metadata, then applies defaults to omitted values. It rejects unknown, duplicate,
// read-only and direct template assignments. Required-value checks happen during rendering,
// when optional template and body activation can be determined.
func (w GenericWebService) resolveAccessValues(assignments []DistributionAccessParameter) (accessValues, error) {
	parameters := make(map[uuid.UUID]Parameter)
	for _, parameter := range w.Parameters {
		parameters[parameter.ID] = parameter.Parameter
		if schema, ok := parameter.Schema.(TemplateSchema); ok {
			for _, child := range schema.Parameters {
				parameters[child.ID] = child
			}
		}
	}

	if w.Body != nil {
		for _, child := range w.Body.Parameters {
			parameters[child.ID] = child
		}
	}

	values := accessValues{
		resolved: make(map[uuid.UUID]AccessArray, len(parameters)),
		supplied: make(map[uuid.UUID]bool, len(assignments)),
	}

	for i, assignment := range assignments {
		target := fmt.Sprintf("parameters[%d]", i)
		parameter, exists := parameters[assignment.ID]
		if !exists {
			return accessValues{}, apperror.Field(target+".id", "Parameter does not exist in this distribution.", nil)
		}

		if values.supplied[assignment.ID] {
			return accessValues{}, apperror.Field(target+".id", "Parameter must not be assigned more than once.", nil)
		}

		if parameter.ReadOnly {
			return accessValues{}, apperror.Field(target+".id", "Read-only parameters cannot be assigned.", nil)
		}

		if _, template := parameter.Schema.(TemplateSchema); template {
			return accessValues{}, apperror.Field(target+".id", "Supply template child values instead of the template parameter itself.", nil)
		}

		resolved, err := resolveAccessValue(parameter.Schema, assignment.Value)
		if err != nil {
			return accessValues{}, apperror.PrefixTarget(err, target)
		}

		values.resolved[assignment.ID] = resolved
		values.supplied[assignment.ID] = true
	}

	for id, parameter := range parameters {
		if values.supplied[id] {
			continue
		}

		value := parameter.Schema.DefaultValue()
		if value == nil {
			continue
		}

		resolved, err := resolveAccessValue(parameter.Schema, value)
		if err != nil {
			return accessValues{}, internalValidationError(err)
		}

		values.resolved[id] = resolved
	}

	return values, nil
}

// resolveAccessValue checks scalar/array shape against schema multiplicity and
// validates each element. It rejects missing values and empty arrays, and returns scalars as
// one-element arrays so rendering can process both shapes uniformly.
func resolveAccessValue(schema ParameterSchema, value AccessValue) (AccessArray, error) {
	if value == nil {
		return nil, apperror.Field("value", "Value is required.", nil)
	}

	items, array := value.(AccessArray)
	if array != schema.AllowsMultipleValues() {
		expected := "a scalar"
		if schema.AllowsMultipleValues() {
			expected = "an array"
		}
		return nil, apperror.Field("value", fmt.Sprintf("Expected %s; received %s.", expected, value.TypeName()), nil)
	}

	if !array {
		scalar, ok := value.(AccessScalar)
		if !ok {
			return nil, apperror.Field("value", "Value must match parameter type and constraints.", nil)
		}

		items = AccessArray{scalar}
	}

	if len(items) == 0 {
		return nil, apperror.Field("value", "Arrays must contain at least one value.", nil)
	}

	result := make(AccessArray, len(items))
	for i, item := range items {
		resolved, err := validateAccessScalar(schema, item)
		if err != nil {
			if appErr, ok := errors.AsType[*apperror.AppError](err); array && ok && appErr.Code == apperror.CodeValidationFailed {
				return nil, apperror.Field(fmt.Sprintf("value[%d]", i), appErr.Detail, err)
			}
			return nil, err
		}

		result[i] = resolved
	}

	return result, nil
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
			return nil, apperror.Field("parameters", "Supply all children of an optional template when supplying any child.", nil)
		}

		return nil, nil
	}

	if required {
		for _, parameter := range parameters {
			if parameter.Required && values.resolved[parameter.ID] == nil {
				return nil, apperror.Field("parameters", fmt.Sprintf("Required parameter %q (%s) is missing.", parameter.Name, parameter.ID), nil)
			}
		}
	}

	parsed, err := template.New("request").Option("missingkey=error").Parse(source)
	if err != nil {
		return nil, internalValidationError(err)
	}

	var rendered strings.Builder
	if err := parsed.Execute(&rendered, data); err != nil {
		return nil, apperror.Field("parameters", "Template could not be rendered with supplied values.", err)
	}

	return new(rendered.String()), nil
}
