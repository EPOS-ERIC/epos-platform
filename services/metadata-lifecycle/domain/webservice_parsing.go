package domain

import (
	"cmp"
	"fmt"
	"math"
	"mime"
	"net/textproto"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
	"golang.org/x/net/http/httpguts"
)

var (
	pathPlaceholderPattern       = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_.-]*)\}`)
	templateReferencePattern     = regexp.MustCompile(`\{\{[^{}]*\.\s*([A-Za-z_][A-Za-z0-9_]*)`)
	templateParameterNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

func NewWebServiceInput(value WebServiceInput) (WebServiceInput, error) {
	value.Type = WebServiceType(strings.ToUpper(strings.TrimSpace(string(value.Type))))
	if !value.Type.Valid() {
		return WebServiceInput{}, invalidField("type", "Type is not supported.", nil)
	}
	if value.Configuration == nil {
		return WebServiceInput{}, invalidField("configuration", "Configuration is required.", nil)
	}
	configuration, err := parseWebServiceConfigurationInput(value.Type, value.Configuration)
	if err != nil {
		return WebServiceInput{}, fmt.Errorf("validate configuration: %w", apperror.PrefixTarget(err, "configuration"))
	}
	value.Configuration = configuration
	visibleOn, err := normalizeWebServiceVisibleOn(value.VisibleOn)
	if err != nil {
		return WebServiceInput{}, invalidField("visible_on", "Audience selection must contain supported values.", err)
	}
	value.VisibleOn = visibleOn
	if value.ProviderID != nil && *value.ProviderID == uuid.Nil {
		return WebServiceInput{}, invalidField("provider", "Provider must be a valid UUID.", nil)
	}
	if err := validateUUIDListField(value.ContactPointIDs, "contact_points", "Contact point"); err != nil {
		return WebServiceInput{}, err
	}
	if value.DocumentationURI != nil {
		*value.DocumentationURI = strings.TrimSpace(*value.DocumentationURI)
		if *value.DocumentationURI == "" {
			return WebServiceInput{}, invalidField("documentation_uri", "Documentation URI must not be blank.", nil)
		}
	}

	return value, nil
}

func parseWebServiceConfigurationInput(serviceType WebServiceType, value WebServiceConfigurationInput) (WebServiceConfigurationInput, error) {
	switch configuration := value.(type) {
	case GenericWebServiceConfigurationInput:
		if serviceType != WebServiceTypeHTTP && serviceType != WebServiceTypeDynamic {
			return nil, invalidField("type", "Configuration does not match service type.", nil)
		}
		return newGenericWebServiceConfigurationInput(serviceType, configuration)
	case WFSConfigurationInput:
		if serviceType != WebServiceTypeWFS {
			return nil, invalidField("type", "Configuration does not match service type.", nil)
		}
		return NewWFSConfigurationInput(configuration)
	case WMSConfigurationInput:
		if serviceType != WebServiceTypeWMS {
			return nil, invalidField("type", "Configuration does not match service type.", nil)
		}
		return NewWMSConfigurationInput(configuration)
	case WMTSConfigurationInput:
		if serviceType != WebServiceTypeWMTS {
			return nil, invalidField("type", "Configuration does not match service type.", nil)
		}
		return NewWMTSConfigurationInput(configuration)
	default:
		return nil, invalidField("configuration", "Configuration type is not supported.", nil)
	}
}

func newGenericWebServiceConfigurationInput(serviceType WebServiceType, value GenericWebServiceConfigurationInput) (GenericWebServiceConfigurationInput, error) {
	value.TemplateURL = strings.TrimSpace(value.TemplateURL)
	value.Format = strings.TrimSpace(value.Format)
	value.Method = HTTPMethod(strings.ToUpper(strings.TrimSpace(string(value.Method))))
	if value.TemplateURL == "" {
		return GenericWebServiceConfigurationInput{}, invalidField("template_url", "Template URL must not be blank.", nil)
	}
	if value.Format == "" {
		return GenericWebServiceConfigurationInput{}, invalidField("format", "Format must not be blank.", nil)
	}
	if !value.Method.Valid() {
		return GenericWebServiceConfigurationInput{}, invalidField("method", "Method is not supported.", nil)
	}
	parameters, err := parseWebServiceParameterInputs(value.Parameters, "parameters")
	if err != nil {
		return GenericWebServiceConfigurationInput{}, err
	}
	value.Parameters = parameters
	if value.Body != nil {
		body, err := NewRequestBodyInput(*value.Body)
		if err != nil {
			return GenericWebServiceConfigurationInput{}, err
		}
		value.Body = &body
	}
	var bodyParameters []ParameterInput
	if value.Body != nil {
		bodyParameters = value.Body.Parameters
	}
	if err := validateWebServiceParameterRoles(serviceType, parameters, "parameters", bodyParameters); err != nil {
		return GenericWebServiceConfigurationInput{}, err
	}
	if err := validateMethodBody(value.Method, value.Body); err != nil {
		return GenericWebServiceConfigurationInput{}, invalidField("body_template", "This method does not support a request body.", err)
	}
	if err := validatePathParameters(value.TemplateURL, pathParameterNames(parameters)); err != nil {
		return GenericWebServiceConfigurationInput{}, invalidField("template_url", "Template URL placeholders must match declared path parameters.", err)
	}
	return value, nil
}

func NewWFSConfigurationInput(value WFSConfigurationInput) (WFSConfigurationInput, error) {
	value.GetFeatureTemplateURL = strings.TrimSpace(value.GetFeatureTemplateURL)
	if value.Version != "1.1.0" && value.Version != "2.0.0" {
		return WFSConfigurationInput{}, invalidField("version", "Select WFS version 1.1.0 or 2.0.0.", nil)
	}
	if err := validateWFSChoices(value.FeatureTypes, value.DefaultFeatureType, "feature_types", "default_feature_type"); err != nil {
		return WFSConfigurationInput{}, err
	}
	if err := validateWFSChoices(value.OutputFormats, value.DefaultOutputFormat, "output_formats", "default_output_format"); err != nil {
		return WFSConfigurationInput{}, err
	}
	if value.ResultLimit != nil && *value.ResultLimit <= 0 {
		return WFSConfigurationInput{}, invalidField("result_limit", "Result limit must be positive.", nil)
	}
	parameters, err := parseWebServiceParameterInputs(value.Parameters, "parameters")
	if err != nil {
		return WFSConfigurationInput{}, err
	}
	if err := validateWebServiceParameterRoles(WebServiceTypeWFS, parameters, "parameters", nil); err != nil {
		return WFSConfigurationInput{}, err
	}
	for i, parameter := range parameters {
		target := fmt.Sprintf("parameters[%d]", i)
		if parameter.Location != ParameterLocationPath && parameter.Location != ParameterLocationQuery {
			return WFSConfigurationInput{}, invalidField(target+".location", "WFS parameters must use PATH or QUERY location.", nil)
		}
		if _, reserved := wfsReservedParameters[strings.ToUpper(parameter.Name)]; reserved {
			return WFSConfigurationInput{}, invalidField(target+".name", "Standard WFS parameters cannot be redefined.", nil)
		}
	}
	value.Parameters = parameters
	endpoint, err := url.Parse(value.GetFeatureTemplateURL)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Fragment != "" {
		return WFSConfigurationInput{}, invalidField("get_feature_template_url", "URL must be an absolute HTTP(S) URL without a fragment.", err)
	}
	if pathPlaceholderPattern.MatchString(endpoint.Host) || pathPlaceholderPattern.MatchString(endpoint.RawQuery) {
		return WFSConfigurationInput{}, invalidField("get_feature_template_url", "Placeholders are only supported in the URL path.", nil)
	}
	if err := validatePathParameters(value.GetFeatureTemplateURL, pathParameterNames(parameters)); err != nil {
		return WFSConfigurationInput{}, invalidField("get_feature_template_url", "Template URL placeholders must match declared path parameters.", err)
	}
	query, err := url.ParseQuery(endpoint.RawQuery)
	if err != nil {
		return WFSConfigurationInput{}, invalidField("get_feature_template_url", "URL query must be valid.", err)
	}
	for name := range query {
		if _, reserved := wfsGeneratedParameters[strings.ToUpper(name)]; reserved {
			return WFSConfigurationInput{}, invalidField("get_feature_template_url", "Platform-generated WFS parameters must not appear in the URL query.", nil)
		}
	}
	return value, nil
}

func validateWFSChoices(values []string, selected, field, defaultField string) error {
	if len(values) == 0 {
		return invalidField(field, "Configure at least one choice.", nil)
	}
	seen := make(map[string]struct{}, len(values))
	for i, value := range values {
		target := fmt.Sprintf("%s[%d]", field, i)
		if value == "" || strings.TrimSpace(value) != value {
			return invalidField(target, "Choices must not be blank or contain surrounding whitespace.", nil)
		}
		if _, exists := seen[value]; exists {
			return invalidField(target, "Choices must be unique.", nil)
		}
		seen[value] = struct{}{}
	}
	if _, exists := seen[selected]; !exists {
		return invalidField(defaultField, "Default must match a configured choice.", nil)
	}
	return nil
}

var (
	wfsGeneratedParameters = reservedParameterNames(
		"SERVICE", "REQUEST", "VERSION", "TYPENAME", "TYPENAMES", "OUTPUTFORMAT", "MAXFEATURES", "COUNT", "BBOX", "SRSNAME",
	)
	wfsReservedParameters = reservedParameterNames(
		"SERVICE", "REQUEST", "VERSION", "ACCEPTVERSIONS", "ACCEPTFORMATS", "SECTIONS", "UPDATESEQUENCE",
		"TYPENAME", "TYPENAMES", "OUTPUTFORMAT", "MAXFEATURES", "COUNT", "BBOX", "SRSNAME",
		"PROPERTYNAME", "FEATUREVERSION", "FEATUREID", "RESOURCEID", "FILTER", "FILTER_LANGUAGE", "SORTBY", "RESULTTYPE", "STARTINDEX",
		"NAMESPACE", "NAMESPACES", "ALIASES", "RESOLVE", "RESOLVEDEPTH", "RESOLVETIMEOUT", "STOREDQUERY_ID",
	)
)

func NewWMTSConfigurationInput(value WMTSConfigurationInput) (WMTSConfigurationInput, error) {
	value.CapabilitiesURL = strings.TrimSpace(value.CapabilitiesURL)
	if err := validateWMSURL("capabilities_url", value.CapabilitiesURL); err != nil {
		return WMTSConfigurationInput{}, err
	}
	if len(value.Layers) == 0 {
		return WMTSConfigurationInput{}, invalidField("layers", "At least one layer is required.", nil)
	}
	layers := make(map[string]struct{}, len(value.Layers))
	for i, layer := range value.Layers {
		path := fmt.Sprintf("layers[%d]", i)
		if strings.TrimSpace(layer.Name) == "" {
			return WMTSConfigurationInput{}, invalidField(path+".name", "Layer name must not be blank.", nil)
		}
		if _, exists := layers[layer.Name]; exists {
			return WMTSConfigurationInput{}, invalidField(path+".name", "Layer names must be unique.", nil)
		}
		layers[layer.Name] = struct{}{}
		if layer.DefaultStyle != nil && (*layer.DefaultStyle == "" || strings.TrimSpace(*layer.DefaultStyle) != *layer.DefaultStyle) {
			return WMTSConfigurationInput{}, invalidField(path+".default_style", "Default style must not be blank or contain surrounding whitespace.", nil)
		}
		styles := make(map[string]struct{}, len(layer.Styles))
		for j, style := range layer.Styles {
			if strings.TrimSpace(style) == "" {
				return WMTSConfigurationInput{}, invalidField(fmt.Sprintf("%s.styles[%d]", path, j), "Style name must not be blank.", nil)
			}
			if _, exists := styles[style]; exists {
				return WMTSConfigurationInput{}, invalidField(fmt.Sprintf("%s.styles[%d]", path, j), "Style names must be unique within a layer.", nil)
			}
			styles[style] = struct{}{}
		}
		if layer.DefaultStyle != nil {
			if _, exists := styles[*layer.DefaultStyle]; !exists {
				return WMTSConfigurationInput{}, invalidField(path+".default_style", "Default style must be one of the configured styles.", nil)
			}
		}
	}
	if _, exists := layers[value.DefaultLayer]; !exists || strings.TrimSpace(value.DefaultLayer) != value.DefaultLayer {
		return WMTSConfigurationInput{}, invalidField("default_layer", "Default layer must be one of the configured layers.", nil)
	}
	var err error
	value.GetTileParameters, err = parseWMTSParameters(value.GetTileParameters, "get_tile_parameters", wmtsGetTileReservedParameters)
	if err != nil {
		return WMTSConfigurationInput{}, err
	}
	value.GetFeatureInfoParameters, err = parseWMTSParameters(value.GetFeatureInfoParameters, "get_feature_info_parameters", wmtsGetFeatureInfoReservedParameters)
	if err != nil {
		return WMTSConfigurationInput{}, err
	}
	return value, nil
}

var (
	wmtsGetTileReservedParameters        = reservedParameterNames("SERVICE", "REQUEST", "VERSION", "LAYER", "STYLE", "FORMAT", "TILEMATRIXSET", "TILEMATRIX", "TILEROW", "TILECOL")
	wmtsGetFeatureInfoReservedParameters = reservedParameterNames("SERVICE", "REQUEST", "VERSION", "LAYER", "STYLE", "FORMAT", "TILEMATRIXSET", "TILEMATRIX", "TILEROW", "TILECOL", "INFOFORMAT", "I", "J")
)

func parseWMTSParameters(values []WebServiceParameterInput, field string, reserved map[string]struct{}) ([]WebServiceParameterInput, error) {
	parameters, err := parseWebServiceParameterInputs(values, field)
	if err != nil {
		return nil, err
	}
	if err := validateWebServiceParameterRoles(WebServiceTypeWMTS, parameters, field, nil); err != nil {
		return nil, err
	}
	for i, parameter := range parameters {
		path := fmt.Sprintf("%s[%d]", field, i)
		if parameter.Location != ParameterLocationQuery {
			return nil, invalidField(path+".location", "WMTS operation parameters must use QUERY location.", nil)
		}
		if _, exists := reserved[strings.ToUpper(parameter.Name)]; exists {
			return nil, invalidField(path+".name", "Standard WMTS parameters cannot be redefined.", nil)
		}
	}
	if err := validateWMSRequiredDefaults(field, parameters); err != nil {
		return nil, err
	}
	return parameters, nil
}

func NewWMSConfigurationInput(value WMSConfigurationInput) (WMSConfigurationInput, error) {
	value.CapabilitiesURL = strings.TrimSpace(value.CapabilitiesURL)
	if err := validateWMSURL("capabilities_url", value.CapabilitiesURL); err != nil {
		return WMSConfigurationInput{}, err
	}
	if strings.TrimSpace(value.DefaultLayer) == "" {
		return WMSConfigurationInput{}, invalidField("default_layer", "Default layer must not be blank.", nil)
	}
	if strings.TrimSpace(value.DefaultLayer) != value.DefaultLayer {
		return WMSConfigurationInput{}, invalidField("default_layer", "Default layer must not contain surrounding whitespace.", nil)
	}
	if len(value.Layers) == 0 {
		return WMSConfigurationInput{}, invalidField("layers", "At least one layer is required.", nil)
	}

	layerNames := make(map[string]struct{}, len(value.Layers))
	for i, layer := range value.Layers {
		path := fmt.Sprintf("layers[%d]", i)
		if strings.TrimSpace(layer.Name) == "" {
			return WMSConfigurationInput{}, invalidField(path+".name", "Layer name must not be blank.", nil)
		}
		if _, exists := layerNames[layer.Name]; exists {
			return WMSConfigurationInput{}, invalidField(path+".name", "Layer names must be unique.", nil)
		}
		layerNames[layer.Name] = struct{}{}
		if strings.TrimSpace(layer.DefaultStyle) != layer.DefaultStyle {
			return WMSConfigurationInput{}, invalidField(path+".default_style", "Default style must not contain surrounding whitespace.", nil)
		}
		if len(layer.Styles) == 0 {
			return WMSConfigurationInput{}, invalidField(path+".styles", "At least one style is required.", nil)
		}
		styleNames := make(map[string]struct{}, len(layer.Styles))
		for j, style := range layer.Styles {
			stylePath := fmt.Sprintf("%s.styles[%d]", path, j)
			if style != "" && strings.TrimSpace(style) == "" {
				return WMSConfigurationInput{}, invalidField(stylePath, "Style name must not be blank unless it is the empty default style.", nil)
			}
			if _, exists := styleNames[style]; exists {
				return WMSConfigurationInput{}, invalidField(stylePath, "Style names must be unique within a layer.", nil)
			}
			styleNames[style] = struct{}{}
		}
		if _, found := styleNames[layer.DefaultStyle]; !found {
			return WMSConfigurationInput{}, invalidField(path+".default_style", "Default style must be one of the configured styles.", nil)
		}
	}
	if _, found := layerNames[value.DefaultLayer]; !found {
		return WMSConfigurationInput{}, invalidField("default_layer", "Default layer must be one of the configured layers.", nil)
	}

	mapParameters, err := parseWMSParameters(value.GetMapParameters, "get_map_parameters", wmsGetMapReservedParameters)
	if err != nil {
		return WMSConfigurationInput{}, err
	}
	featureInfoParameters, err := parseWMSParameters(value.GetFeatureInfoParameters, "get_feature_info_parameters", wmsGetFeatureInfoReservedParameters)
	if err != nil {
		return WMSConfigurationInput{}, err
	}
	value.GetMapParameters = mapParameters
	value.GetFeatureInfoParameters = featureInfoParameters
	if err := validateWMSRequiredDefaults("get_map_parameters", mapParameters); err != nil {
		return WMSConfigurationInput{}, err
	}
	if err := validateWMSRequiredDefaults("get_feature_info_parameters", featureInfoParameters); err != nil {
		return WMSConfigurationInput{}, err
	}
	return value, nil
}

var (
	wmsGetMapReservedParameters = reservedParameterNames(
		"SERVICE", "VERSION", "REQUEST", "LAYERS", "STYLES", "SRS", "CRS", "BBOX", "WIDTH", "HEIGHT", "FORMAT", "TRANSPARENT", "BGCOLOR", "EXCEPTIONS",
	)
	wmsGetFeatureInfoReservedParameters = reservedParameterNames(
		"SERVICE", "VERSION", "REQUEST", "LAYERS", "STYLES", "SRS", "CRS", "BBOX", "WIDTH", "HEIGHT", "FORMAT", "TRANSPARENT", "BGCOLOR", "EXCEPTIONS",
		"QUERY_LAYERS", "INFO_FORMAT", "FEATURE_COUNT", "X", "Y", "I", "J",
	)
)

func reservedParameterNames(names ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(names))
	for _, name := range names {
		result[name] = struct{}{}
	}
	return result
}

func parseWMSParameters(values []WebServiceParameterInput, field string, reserved map[string]struct{}) ([]WebServiceParameterInput, error) {
	parameters, err := parseWebServiceParameterInputs(values, field)
	if err != nil {
		return nil, err
	}
	if err := validateWebServiceParameterRoles(WebServiceTypeWMS, parameters, field, nil); err != nil {
		return nil, err
	}
	for i, parameter := range parameters {
		path := fmt.Sprintf("%s[%d]", field, i)
		if parameter.Location != ParameterLocationQuery {
			return nil, invalidField(path+".location", "WMS operation parameters must use QUERY location.", nil)
		}
		if _, exists := reserved[strings.ToUpper(parameter.Name)]; exists {
			return nil, invalidField(path+".name", "Standard WMS parameters cannot be redefined.", nil)
		}
	}
	return parameters, nil
}

func validateWMSURL(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalidField(field, "URL must not be blank.", nil)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (strings.ToLower(parsed.Scheme) != "http" && strings.ToLower(parsed.Scheme) != "https") || parsed.Fragment != "" {
		return invalidField(field, "URL must be an absolute HTTP(S) URL without a fragment.", err)
	}
	return nil
}

func validateWMSRequiredDefaults(field string, values []WebServiceParameterInput) error {
	for i, parameter := range values {
		path := fmt.Sprintf("%s[%d]", field, i)
		if !parameter.Required {
			continue
		}
		if schema, ok := parameter.Schema.(TemplateSchemaInput); ok {
			for j, input := range schema.Parameters {
				if input.Required && input.Schema.DefaultValue() == nil {
					return invalidField(fmt.Sprintf("%s.schema.parameters[%d].default", path, j), "Required template inputs need a default for immediate execution.", nil)
				}
			}
			continue
		}
		if parameter.Schema.DefaultValue() == nil {
			return invalidField(path+".default", "Required parameters need a default for immediate execution.", nil)
		}
	}
	return nil
}

// NewWebService parses and validates persisted web-service metadata.
func NewWebService(value WebService) (result WebService, err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if value.ID == uuid.Nil {
		return WebService{}, fmt.Errorf("web service ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if err := validatePersistedWebServiceConfiguration(value.Configuration); err != nil {
		return WebService{}, err
	}

	input, err := NewWebServiceInput(value.Input())
	if err != nil {
		return WebService{}, err
	}
	value.Type = input.Type
	value.VisibleOn = input.VisibleOn
	value.DocumentationURI = input.DocumentationURI
	value.Configuration, err = restoreWebServiceConfiguration(input.Configuration, value.Configuration)
	if err != nil {
		return WebService{}, fmt.Errorf("restore configuration parameter IDs: %w", err)
	}
	if value.Provider != nil {
		if err := validateOrganization(*value.Provider); err != nil {
			return WebService{}, fmt.Errorf("provider: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
	}
	if err := validateContactPoints(value.ContactPoints, "web service contact point"); err != nil {
		return WebService{}, fmt.Errorf("contact points: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	return value, nil
}

func validatePersistedWebServiceConfiguration(value WebServiceConfiguration) error {
	switch configuration := value.(type) {
	case GenericWebServiceConfiguration:
		if err := validatePersistedWebServiceParameters(configuration.Parameters); err != nil {
			return err
		}
		if configuration.Body != nil {
			if err := validatePersistedParameters(configuration.Body.Parameters); err != nil {
				return fmt.Errorf("request body parameters: %w", err)
			}
		}
	case WFSConfiguration:
		if err := validatePersistedWebServiceParameters(configuration.Parameters); err != nil {
			return err
		}
	case WMSConfiguration:
		if err := validatePersistedWebServiceParameters(configuration.GetMapParameters); err != nil {
			return fmt.Errorf("GetMap parameters: %w", err)
		}
		if err := validatePersistedWebServiceParameters(configuration.GetFeatureInfoParameters); err != nil {
			return fmt.Errorf("GetFeatureInfo parameters: %w", err)
		}
	case WMTSConfiguration:
		if err := validatePersistedWebServiceParameters(configuration.GetTileParameters); err != nil {
			return fmt.Errorf("GetTile parameters: %w", err)
		}
		if err := validatePersistedWebServiceParameters(configuration.GetFeatureInfoParameters); err != nil {
			return fmt.Errorf("GetFeatureInfo parameters: %w", err)
		}
	default:
		return fmt.Errorf("web service configuration is missing or unsupported: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

func validatePersistedWebServiceParameters(values []WebServiceParameter) error {
	for _, value := range values {
		if err := validatePersistedParameter(value.Parameter); err != nil {
			return err
		}
	}
	return nil
}

func restoreWebServiceConfiguration(input WebServiceConfigurationInput, existing WebServiceConfiguration) (WebServiceConfiguration, error) {
	switch configuration := input.(type) {
	case GenericWebServiceConfigurationInput:
		persisted, ok := existing.(GenericWebServiceConfiguration)
		if !ok {
			return nil, fmt.Errorf("generic configuration is missing: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		parameters, err := restoreWebServiceParameters(configuration.Parameters, persisted.Parameters)
		if err != nil {
			return nil, err
		}
		var body *RequestBody
		if configuration.Body != nil {
			if persisted.Body == nil {
				return nil, fmt.Errorf("request body configuration is missing: %w", apperror.Default(apperror.CodeValidationFailed))
			}
			restored, err := restoreRequestBody(*configuration.Body, *persisted.Body)
			if err != nil {
				return nil, err
			}
			body = &restored
		}
		return GenericWebServiceConfiguration{
			TemplateURL: configuration.TemplateURL,
			Method:      configuration.Method,
			Body:        body,
			Parameters:  parameters,
			Format:      configuration.Format,
		}, nil
	case WFSConfigurationInput:
		persisted, ok := existing.(WFSConfiguration)
		if !ok {
			return nil, fmt.Errorf("WFS configuration is missing: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		parameters, err := restoreWebServiceParameters(configuration.Parameters, persisted.Parameters)
		if err != nil {
			return nil, err
		}
		return WFSConfiguration{
			GetFeatureTemplateURL: configuration.GetFeatureTemplateURL,
			Version:               configuration.Version,
			FeatureTypes:          configuration.FeatureTypes,
			DefaultFeatureType:    configuration.DefaultFeatureType,
			OutputFormats:         configuration.OutputFormats,
			DefaultOutputFormat:   configuration.DefaultOutputFormat,
			ResultLimit:           configuration.ResultLimit,
			ResultLimitEditable:   configuration.ResultLimitEditable,
			Parameters:            parameters,
		}, nil
	case WMTSConfigurationInput:
		persisted, ok := existing.(WMTSConfiguration)
		if !ok {
			return nil, fmt.Errorf("WMTS configuration is missing: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		tile, err := restoreWebServiceParameters(configuration.GetTileParameters, persisted.GetTileParameters)
		if err != nil {
			return nil, fmt.Errorf("GetTile parameters: %w", err)
		}
		info, err := restoreWebServiceParameters(configuration.GetFeatureInfoParameters, persisted.GetFeatureInfoParameters)
		if err != nil {
			return nil, fmt.Errorf("GetFeatureInfo parameters: %w", err)
		}
		return WMTSConfiguration{
			CapabilitiesURL:          configuration.CapabilitiesURL,
			DefaultLayer:             configuration.DefaultLayer,
			Layers:                   persisted.Layers,
			GetTileParameters:        tile,
			GetFeatureInfoParameters: info,
		}, nil

	case WMSConfigurationInput:
		persisted, ok := existing.(WMSConfiguration)
		if !ok {
			return nil, fmt.Errorf("WMS configuration is missing: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		mapParameters, err := restoreWebServiceParameters(configuration.GetMapParameters, persisted.GetMapParameters)
		if err != nil {
			return nil, fmt.Errorf("GetMap parameters: %w", err)
		}
		featureInfoParameters, err := restoreWebServiceParameters(configuration.GetFeatureInfoParameters, persisted.GetFeatureInfoParameters)
		if err != nil {
			return nil, fmt.Errorf("GetFeatureInfo parameters: %w", err)
		}
		return WMSConfiguration{
			CapabilitiesURL:          configuration.CapabilitiesURL,
			DefaultLayer:             configuration.DefaultLayer,
			Layers:                   persisted.Layers,
			TransparencyEditable:     configuration.TransparencyEditable,
			DefaultTransparency:      configuration.DefaultTransparency,
			GetMapParameters:         mapParameters,
			GetFeatureInfoParameters: featureInfoParameters,
		}, nil

	default:
		return nil, fmt.Errorf("web service input configuration is unsupported: %w", apperror.Default(apperror.CodeValidationFailed))
	}
}

func restoreWebServiceParameters(values []WebServiceParameterInput, existing []WebServiceParameter) ([]WebServiceParameter, error) {
	byName := make(map[string]WebServiceParameter, len(existing))
	for _, value := range existing {
		byName[value.Name] = value
	}
	result := make([]WebServiceParameter, len(values))
	for i, value := range values {
		persisted, err := restoreParameter(value.ParameterInput, byName[value.Name].Parameter)
		if err != nil {
			return nil, fmt.Errorf("restore parameter %q ID: %w", value.Name, err)
		}
		result[i] = WebServiceParameter{
			Parameter: persisted,
			Location:  value.Location,
			Explode:   value.Explode,
		}
	}
	return result, nil
}

func normalizeWebServiceVisibleOn(values []WebServiceVisibleOn) ([]WebServiceVisibleOn, error) {
	if values == nil {
		return nil, nil
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("visible on must contain at least one value: %w", apperror.Default(apperror.CodeValidationFailed))
	}

	result := make([]WebServiceVisibleOn, 0, len(values))
	seen := make(map[WebServiceVisibleOn]struct{}, len(values))
	for _, value := range values {
		value = WebServiceVisibleOn(strings.ToUpper(strings.TrimSpace(string(value))))
		if !value.Valid() {
			return nil, fmt.Errorf("visible on value %q is invalid: %w", value, apperror.Default(apperror.CodeValidationFailed))
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

// NewWebServiceParameterInput parses a caller-controlled top-level parameter.
func NewWebServiceParameterInput(value WebServiceParameterInput) (WebServiceParameterInput, error) {
	parameter, err := parseParameter(value.ParameterInput, true, true)
	if err != nil {
		return WebServiceParameterInput{}, err
	}
	value.ParameterInput = parameter
	value.Location = ParameterLocation(strings.ToUpper(strings.TrimSpace(string(value.Location))))
	if !value.Location.Valid() {
		return WebServiceParameterInput{}, invalidField("location", "Location is not supported.", nil)
	}
	if value.Explode && (value.Location != ParameterLocationQuery || !value.Schema.AllowsMultipleValues()) {
		return WebServiceParameterInput{}, invalidField("explode", "Explode requires a query parameter with multiple values.", nil)
	}
	if value.Location == ParameterLocationPath && !value.Required {
		return WebServiceParameterInput{}, invalidField("required", "Path parameters must be required.", nil)
	}
	if value.Location == ParameterLocationHeader {
		if !httpguts.ValidHeaderFieldName(value.Name) {
			return WebServiceParameterInput{}, invalidField("name", "Header name must be a valid HTTP field name.", nil)
		}
		switch textproto.CanonicalMIMEHeaderKey(value.Name) {
		case "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "Proxy-Connection", "Keep-Alive":
			return WebServiceParameterInput{}, invalidField("name", "Headers must not override destination or HTTP framing.", nil)
		default:
		}
	}
	return value, nil
}

// NewRequestBodyInput parses and validates caller-controlled request-body fields.
func NewRequestBodyInput(value RequestBodyInput) (RequestBodyInput, error) {
	if value.Required == nil {
		value.Required = new(true)
	}
	value.ContentType = strings.TrimSpace(value.ContentType)
	value.Template = strings.TrimSpace(value.Template)

	if value.ContentType == "" {
		return RequestBodyInput{}, invalidField("body_content_type", "Content type must not be blank.", nil)
	}

	contentType, err := normalizeMediaType(value.ContentType)
	if err != nil {
		return RequestBodyInput{}, invalidField("body_content_type", "Content type must be a valid media type.", err)
	}
	value.ContentType = contentType

	templateValue, parameters, err := parseTemplateDefinition(value.Template, value.Parameters)
	if err != nil {
		return RequestBodyInput{}, prefixOrField(err, "body_parameters", "body_template", "Template must be valid and reference declared parameters.")
	}
	value.Template = templateValue
	value.Parameters = parameters

	return value, nil
}

func normalizeMediaType(value string) (string, error) {
	mediaType, parameters, err := mime.ParseMediaType(value)
	if err != nil {
		return "", fmt.Errorf("parse MIME media type %q: %w: %w", value, err, apperror.Default(apperror.CodeValidationFailed))
	}

	if !strings.Contains(mediaType, "/") {
		return "", fmt.Errorf("MIME media type %q must contain type and subtype: %w", value, apperror.Default(apperror.CodeValidationFailed))
	}

	normalized := mime.FormatMediaType(mediaType, parameters)
	if normalized == "" {
		return "", fmt.Errorf("format MIME media type %q: %w", value, apperror.Default(apperror.CodeValidationFailed))
	}

	return normalized, nil
}

func parseTemplateDefinition(source string, values []ParameterInput) (string, []ParameterInput, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", nil, fmt.Errorf("template is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if _, err := template.New("parameter").Funcs(cqlTemplateFunctions).Option("missingkey=error").Parse(source); err != nil {
		return "", nil, fmt.Errorf("parse template: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	parameters, err := parseParameters(values)
	if err != nil {
		return "", nil, fmt.Errorf("template parameters: %w", err)
	}
	if err := validateTemplateReferences(source, parameters); err != nil {
		return "", nil, fmt.Errorf("template references: %w", err)
	}
	return source, parameters, nil
}

func parseWebServiceParameterInputs(values []WebServiceParameterInput, field string) ([]WebServiceParameterInput, error) {
	result := make([]WebServiceParameterInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	seenHeaders := make(map[string]struct{})
	for i, value := range values {
		parsed, err := NewWebServiceParameterInput(value)
		if err != nil {
			return nil, fmt.Errorf("validate web service parameter: %w", apperror.PrefixTarget(err, fmt.Sprintf("%s[%d]", field, i)))
		}
		if _, exists := seen[parsed.Name]; exists {
			return nil, invalidField(fmt.Sprintf("%s[%d].name", field, i), "Parameter names must be unique.", nil)
		}
		if parsed.Location == ParameterLocationHeader {
			name := textproto.CanonicalMIMEHeaderKey(parsed.Name)
			if _, exists := seenHeaders[name]; exists {
				return nil, invalidField(fmt.Sprintf("%s[%d].name", field, i), "Header names must be unique ignoring case.", nil)
			}
			seenHeaders[name] = struct{}{}
		}
		seen[parsed.Name] = struct{}{}
		result = append(result, parsed)
	}
	slices.SortFunc(result, func(a, b WebServiceParameterInput) int {
		if order := cmp.Compare(a.Position, b.Position); order != 0 {
			return order
		}
		return strings.Compare(a.Name, b.Name)
	})
	return result, nil
}

func webServiceParameterInputs(values []WebServiceParameter) []WebServiceParameterInput {
	result := make([]WebServiceParameterInput, len(values))
	for i, value := range values {
		result[i] = WebServiceParameterInput{
			ParameterInput: value.Input(),
			Location:       value.Location,
			Explode:        value.Explode,
		}
	}
	return result
}

func (value Parameter) Input() ParameterInput {
	input := value.ParameterInput
	if value.ResourceID != uuid.Nil {
		input.ResourceID = &value.ResourceID
	}
	input.Schema = parameterSchemaInput(input.Schema)
	return input
}

func parameterSchemaInput(value ParameterSchema) ParameterSchema {
	switch schema := value.(type) {
	case TemplateSchema:
		return TemplateSchemaInput{Template: schema.Template, Parameters: parameterInputs(schema.Parameters)}
	default:
		return value
	}
}

func parameterInputs(values []Parameter) []ParameterInput {
	result := make([]ParameterInput, len(values))
	for i, value := range values {
		result[i] = value.Input()
	}
	return result
}

func requestBodyInput(value *RequestBody) *RequestBodyInput {
	if value == nil {
		return nil
	}
	return &RequestBodyInput{
		Required:    new(value.Required),
		ContentType: value.ContentType,
		Template:    value.Template,
		Parameters:  parameterInputs(value.Parameters),
	}
}

func validatePersistedParameter(value Parameter) error {
	if value.ID == uuid.Nil {
		return fmt.Errorf("parameter ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if schema, ok := value.Schema.(TemplateSchema); ok {
		return validatePersistedParameters(schema.Parameters)
	}
	if _, ok := value.Schema.(TemplateSchemaInput); ok {
		return fmt.Errorf("input template schema in persisted parameter: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

func validatePersistedParameters(values []Parameter) error {
	for _, value := range values {
		if err := validatePersistedParameter(value); err != nil {
			return err
		}
	}
	return nil
}

func restoreParameter(value ParameterInput, existing Parameter) (Parameter, error) {
	if existing.ID == uuid.Nil {
		return Parameter{}, fmt.Errorf("parameter ID is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	result := Parameter{
		ParameterInput: value,
		ID:             existing.ID,
		ResourceID:     existing.ResourceID,
	}
	if schema, ok := value.Schema.(TemplateSchemaInput); ok {
		existingSchema, ok := existing.Schema.(TemplateSchema)
		if !ok {
			return Parameter{}, fmt.Errorf("persisted parameter template is missing: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		parameters, err := restoreParameters(schema.Parameters, existingSchema.Parameters)
		if err != nil {
			return Parameter{}, err
		}
		result.Schema = TemplateSchema{
			Template:   schema.Template,
			Parameters: parameters,
		}
	}
	return result, nil
}

func restoreParameters(values []ParameterInput, existing []Parameter) ([]Parameter, error) {
	byName := make(map[string]Parameter, len(existing))
	for _, value := range existing {
		byName[value.Name] = value
	}
	result := make([]Parameter, len(values))
	for i, value := range values {
		persisted, err := restoreParameter(value, byName[value.Name])
		if err != nil {
			return nil, fmt.Errorf("restore parameter %q: %w", value.Name, err)
		}
		result[i] = persisted
	}
	return result, nil
}

func restoreRequestBody(value RequestBodyInput, existing RequestBody) (RequestBody, error) {
	parameters, err := restoreParameters(value.Parameters, existing.Parameters)
	if err != nil {
		return RequestBody{}, err
	}
	return RequestBody{
		Required:    *value.Required,
		ContentType: value.ContentType,
		Template:    value.Template,
		Parameters:  parameters,
	}, nil
}

func parseParameter(value ParameterInput, allowTemplate, allowMultipleValues bool) (ParameterInput, error) {
	value.Name = strings.TrimSpace(value.Name)
	value.Label = strings.TrimSpace(value.Label)
	value.Description = optionalTrimmedString(value.Description)
	value.Property = optionalTrimmedString(value.Property)
	if value.Role != nil {
		role := WebServiceParameterRole(strings.ToUpper(strings.TrimSpace(string(*value.Role))))
		if !role.Valid() {
			return ParameterInput{}, invalidField("role", "Role is not supported.", nil)
		}
		value.Role = &role
	}
	if value.Name == "" {
		return ParameterInput{}, invalidField("name", "Name must not be blank.", nil)
	}
	if value.Label == "" {
		return ParameterInput{}, invalidField("label", "Label must not be blank.", nil)
	}
	if value.Schema == nil {
		return ParameterInput{}, invalidField("schema", "Schema is required.", nil)
	}
	schema, err := parseParameterSchema(value.Schema, allowTemplate)
	if err != nil {
		return ParameterInput{}, prefixOrField(err, "schema", "schema", "Schema definition is invalid.")
	}
	if !allowMultipleValues && schema.AllowsMultipleValues() {
		return ParameterInput{}, invalidField("schema.multiple_values", "Multiple values are not allowed here.", nil)
	}
	value.Schema = schema
	return value, nil
}

func parseParameters(values []ParameterInput) ([]ParameterInput, error) {
	result := make([]ParameterInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for i, value := range values {
		parsed, err := parseParameter(value, false, false)
		if err != nil {
			return nil, fmt.Errorf("validate nested parameter: %w", apperror.PrefixTarget(err, fmt.Sprintf("[%d]", i)))
		}
		if !templateParameterNamePattern.MatchString(parsed.Name) {
			return nil, invalidField(fmt.Sprintf("[%d].name", i), "Name must start with a letter or underscore and contain only letters, numbers, and underscores.", nil)
		}
		if _, exists := seen[parsed.Name]; exists {
			return nil, invalidField(fmt.Sprintf("[%d].name", i), "Parameter names must be unique.", nil)
		}
		seen[parsed.Name] = struct{}{}
		result = append(result, parsed)
	}
	slices.SortFunc(result, func(a, b ParameterInput) int {
		if order := cmp.Compare(a.Position, b.Position); order != 0 {
			return order
		}
		return strings.Compare(a.Name, b.Name)
	})
	return result, nil
}

func parseParameterSchema(value ParameterSchema, allowTemplate bool) (ParameterSchema, error) {
	switch schema := value.(type) {
	case StringSchema:
		return parseStringSchema(schema)
	case BooleanSchema:
		return schema, nil
	case IntegerSchema:
		return parseIntegerSchema(schema)
	case FloatSchema:
		return parseFloatSchema(schema)
	case DateTimeSchema:
		return parseDateTimeSchema(schema)
	case TemplateSchemaInput:
		if !allowTemplate {
			return nil, fmt.Errorf("template schema is not allowed here: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		return parseTemplateSchema(schema)
	default:
		return nil, fmt.Errorf("unsupported parameter schema %T: %w", value, apperror.Default(apperror.CodeValidationFailed))
	}
}

func parseStringSchema(value StringSchema) (StringSchema, error) {
	value.Pattern = optionalTrimmedString(value.Pattern)
	if value.Pattern != nil {
		if _, err := regexp.Compile(*value.Pattern); err != nil {
			return StringSchema{}, fmt.Errorf("invalid string pattern: %w: %w", err, apperror.Default(apperror.CodeValidationFailed))
		}
	}
	if value.Enum != nil && len(value.Enum) == 0 {
		return StringSchema{}, fmt.Errorf("string enum cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	seen := make(map[string]struct{}, len(value.Enum))
	for i := range value.Enum {
		item := &value.Enum[i]
		if item.Value == "" {
			return StringSchema{}, fmt.Errorf("string enum values cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		if _, exists := seen[item.Value]; exists {
			return StringSchema{}, fmt.Errorf("string enum value %q is duplicated: %w", item.Value, apperror.Default(apperror.CodeValidationFailed))
		}
		seen[item.Value] = struct{}{}
		if item.Label != nil {
			label := strings.TrimSpace(*item.Label)
			if label == "" {
				return StringSchema{}, fmt.Errorf("string enum label cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
			}
			item.Label = &label
		}
		if value.Pattern != nil && !regexp.MustCompile(*value.Pattern).MatchString(item.Value) {
			return StringSchema{}, fmt.Errorf("string enum value %q does not match pattern: %w", item.Value, apperror.Default(apperror.CodeValidationFailed))
		}
	}
	if value.Default != nil {
		if len(value.Default) == 0 {
			return StringSchema{}, fmt.Errorf("string defaults cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		if !value.MultipleValues && len(value.Default) > 1 {
			return StringSchema{}, fmt.Errorf("scalar string parameters allow at most one default: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		for _, defaultValue := range value.Default {
			if strings.TrimSpace(defaultValue) == "" {
				return StringSchema{}, fmt.Errorf("string default cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
			}
			if value.Pattern != nil && !regexp.MustCompile(*value.Pattern).MatchString(defaultValue) {
				return StringSchema{}, fmt.Errorf("string default does not match pattern: %w", apperror.Default(apperror.CodeValidationFailed))
			}
			if value.Enum != nil && !slices.ContainsFunc(value.Enum, func(item StringEnumOption) bool {
				return item.Value == defaultValue
			}) {
				return StringSchema{}, fmt.Errorf("string default is not an allowed value: %w", apperror.Default(apperror.CodeValidationFailed))
			}
		}
	}
	return value, nil
}

func parseIntegerSchema(value IntegerSchema) (IntegerSchema, error) {
	if value.Enum != nil && len(value.Enum) == 0 {
		return IntegerSchema{}, fmt.Errorf("integer enum cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value.Minimum != nil && value.Maximum != nil && *value.Minimum > *value.Maximum {
		return IntegerSchema{}, fmt.Errorf("integer minimum must not exceed maximum: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	seen := make(map[int64]struct{}, len(value.Enum))
	for i := range value.Enum {
		item := &value.Enum[i]
		if _, exists := seen[item.Value]; exists {
			return IntegerSchema{}, fmt.Errorf("integer enum value %d is duplicated: %w", item.Value, apperror.Default(apperror.CodeValidationFailed))
		}
		seen[item.Value] = struct{}{}
		if item.Label != nil {
			label := strings.TrimSpace(*item.Label)
			if label == "" {
				return IntegerSchema{}, fmt.Errorf("integer enum label cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
			}
			item.Label = &label
		}
		if !integerInRange(item.Value, value.Minimum, value.Maximum) {
			return IntegerSchema{}, fmt.Errorf("integer enum value is outside allowed range: %w", apperror.Default(apperror.CodeValidationFailed))
		}
	}
	if value.Default != nil {
		if len(value.Default) == 0 {
			return IntegerSchema{}, fmt.Errorf("integer defaults cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		if !value.MultipleValues && len(value.Default) > 1 {
			return IntegerSchema{}, fmt.Errorf("scalar integer parameters allow at most one default: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		for _, defaultValue := range value.Default {
			if !integerInRange(defaultValue, value.Minimum, value.Maximum) {
				return IntegerSchema{}, fmt.Errorf("integer default is outside allowed range: %w", apperror.Default(apperror.CodeValidationFailed))
			}
			if value.Enum != nil && !slices.ContainsFunc(value.Enum, func(item IntegerEnumOption) bool {
				return item.Value == defaultValue
			}) {
				return IntegerSchema{}, fmt.Errorf("integer default is not an allowed value: %w", apperror.Default(apperror.CodeValidationFailed))
			}
		}
	}
	return value, nil
}

func parseFloatSchema(value FloatSchema) (FloatSchema, error) {
	if value.Enum != nil && len(value.Enum) == 0 {
		return FloatSchema{}, fmt.Errorf("float enum cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	seen := make(map[float64]struct{}, len(value.Enum))
	for i := range value.Enum {
		item := &value.Enum[i]
		if _, exists := seen[item.Value]; exists {
			return FloatSchema{}, fmt.Errorf("float enum value %v is duplicated: %w", item.Value, apperror.Default(apperror.CodeValidationFailed))
		}
		seen[item.Value] = struct{}{}
		if item.Label != nil {
			label := strings.TrimSpace(*item.Label)
			if label == "" {
				return FloatSchema{}, fmt.Errorf("float enum label cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
			}
			item.Label = &label
		}
		if math.IsNaN(item.Value) || math.IsInf(item.Value, 0) || !floatInRange(item.Value, value.Minimum, value.Maximum) {
			return FloatSchema{}, fmt.Errorf("float enum value is invalid or outside allowed range: %w", apperror.Default(apperror.CodeValidationFailed))
		}
	}
	if value.Minimum != nil && value.Maximum != nil && *value.Minimum > *value.Maximum {
		return FloatSchema{}, fmt.Errorf("float minimum must not exceed maximum: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value.Default != nil {
		if len(value.Default) == 0 {
			return FloatSchema{}, fmt.Errorf("float defaults cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		if !value.MultipleValues && len(value.Default) > 1 {
			return FloatSchema{}, fmt.Errorf("scalar float parameters allow at most one default: %w", apperror.Default(apperror.CodeValidationFailed))
		}
		for _, defaultValue := range value.Default {
			if math.IsNaN(defaultValue) || math.IsInf(defaultValue, 0) || !floatInRange(defaultValue, value.Minimum, value.Maximum) {
				return FloatSchema{}, fmt.Errorf("float default is invalid or outside allowed range: %w", apperror.Default(apperror.CodeValidationFailed))
			}
			if value.Enum != nil && !slices.ContainsFunc(value.Enum, func(item FloatEnumOption) bool {
				return item.Value == defaultValue
			}) {
				return FloatSchema{}, fmt.Errorf("float default is not an allowed value: %w", apperror.Default(apperror.CodeValidationFailed))
			}
		}
	}
	return value, nil
}

func parseDateTimeSchema(value DateTimeSchema) (DateTimeSchema, error) {
	if value.Layout != nil {
		layout := strings.TrimSpace(*value.Layout)
		value.Layout = &layout
		if err := validateDateTimeLayout(layout); err != nil {
			return DateTimeSchema{}, err
		}
	}
	if value.Minimum != nil && value.Maximum != nil && value.Minimum.After(*value.Maximum) {
		return DateTimeSchema{}, fmt.Errorf("date-time minimum must not be after maximum: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value.Default != nil && !timeInRange(*value.Default, value.Minimum, value.Maximum) {
		return DateTimeSchema{}, fmt.Errorf("date-time default is outside allowed range: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	return value, nil
}

func parseTemplateSchema(value TemplateSchemaInput) (TemplateSchemaInput, error) {
	templateValue, parameters, err := parseTemplateDefinition(value.Template, value.Parameters)
	if err != nil {
		return TemplateSchemaInput{}, fmt.Errorf("validate template parameters: %w", apperror.PrefixTarget(err, "parameters"))
	}
	value.Template = templateValue
	value.Parameters = parameters
	return value, nil
}

func validateTemplateReferences(source string, parameters []ParameterInput) error {
	allowed := make(map[string]struct{}, len(parameters))
	for _, parameter := range parameters {
		allowed[parameter.Name] = struct{}{}
	}
	for _, match := range templateReferencePattern.FindAllStringSubmatch(source, -1) {
		if _, exists := allowed[match[1]]; !exists {
			return fmt.Errorf("template references undeclared parameter %q: %w", match[1], apperror.Default(apperror.CodeValidationFailed))
		}
	}
	return nil
}

func validatePathParameters(templateURL string, pathParameters map[string]struct{}) error {
	placeholders := make(map[string]struct{})
	for _, match := range pathPlaceholderPattern.FindAllStringSubmatch(templateURL, -1) {
		placeholders[match[1]] = struct{}{}
	}
	for name := range pathParameters {
		if _, exists := placeholders[name]; !exists {
			return fmt.Errorf("path parameter %q has no URL placeholder: %w", name, apperror.Default(apperror.CodeValidationFailed))
		}
		delete(placeholders, name)
	}
	if len(placeholders) != 0 {
		for name := range placeholders {
			return fmt.Errorf("URL placeholder %q has no path parameter: %w", name, apperror.Default(apperror.CodeValidationFailed))
		}
	}
	return nil
}

func pathParameterNames(parameters []WebServiceParameterInput) map[string]struct{} {
	result := make(map[string]struct{})
	for _, parameter := range parameters {
		if parameter.Location == ParameterLocationPath {
			result[parameter.Name] = struct{}{}
		}
	}
	return result
}

func validateMethodBody(method HTTPMethod, body *RequestBodyInput) error {
	if body != nil && (method == HTTPMethodGet || method == HTTPMethodDelete) {
		return fmt.Errorf("body is not allowed for %s: %w", method, apperror.Default(apperror.CodeValidationFailed))
	}
	return nil
}

func validateDateTimeLayout(layout string) error {
	if strings.TrimSpace(layout) == "" {
		return fmt.Errorf("date-time layout cannot be empty: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	tokens := []struct {
		name    string
		goValue string
	}{
		{"YYYY", "2006"},
		{"MM", "01"},
		{"DD", "02"},
		{"hh", "15"},
		{"mm", "04"},
		{"ss", "05"},
	}
	for index := 0; index < len(layout); {
		matched := false
		for _, token := range tokens {
			if strings.HasPrefix(layout[index:], token.name) {
				index += len(token.name)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		if strings.ContainsRune("YyMDHhms", rune(layout[index])) {
			return fmt.Errorf("unsupported date-time layout token near %q: %w", layout[index:], apperror.Default(apperror.CodeValidationFailed))
		}
		index++
	}
	return nil
}

func optionalTrimmedString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func integerInRange(value int64, minimum, maximum *int64) bool {
	return (minimum == nil || value >= *minimum) && (maximum == nil || value <= *maximum)
}

func floatInRange(value float64, minimum, maximum *float64) bool {
	return (minimum == nil || value >= *minimum) && (maximum == nil || value <= *maximum)
}

func timeInRange(value time.Time, minimum, maximum *time.Time) bool {
	return (minimum == nil || !value.Before(*minimum)) && (maximum == nil || !value.After(*maximum))
}
