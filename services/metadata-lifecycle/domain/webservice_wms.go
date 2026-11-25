package domain

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

type WMSLayerInput struct {
	ResourceID   *uuid.UUID
	Name         string
	DefaultStyle string
	Styles       []string
}

type WMSLayer struct {
	ID           uuid.UUID
	ResourceID   uuid.UUID
	Name         string
	DefaultStyle string
	Styles       []string
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
			return nil, apperror.Field(path+".location", "WMS operation parameters must use QUERY location.", nil)
		}
		if _, exists := reserved[strings.ToUpper(parameter.Name)]; exists {
			return nil, apperror.Field(path+".name", "Standard WMS parameters cannot be redefined.", nil)
		}
	}
	return parameters, nil
}

func normalizeWMSURL(value *string) error {
	normalized, _, _ := strings.Cut(strings.TrimSpace(*value), "#")
	if normalized == "" {
		return apperror.Field("capabilities_url", "URL must not be blank.", nil)
	}
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return apperror.Field("capabilities_url", "URL must be an absolute HTTP(S) URL.", err)
	}
	*value = normalized
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
					return apperror.Field(fmt.Sprintf("%s.parameters[%d].default", path, j), "Required template inputs need a default for immediate execution.", nil)
				}
			}
			continue
		}
		if parameter.Schema.DefaultValue() == nil {
			return apperror.Field(path+".default", "Required parameters need a default for immediate execution.", nil)
		}
	}
	return nil
}

// ############### INPUT ###############

type WMSWebServiceInput struct {
	WebServiceCommonInput
	CapabilitiesURL          string
	DefaultLayer             string
	Layers                   []WMSLayerInput
	TransparencyEditable     bool
	DefaultTransparency      bool
	GetMapParameters         []WebServiceParameterInput
	GetFeatureInfoParameters []WebServiceParameterInput
}

func (*WMSWebServiceInput) ServiceType() WebServiceType {
	return WebServiceTypeWMS
}

func (value *WMSWebServiceInput) Normalize() error {
	if value == nil {
		return apperror.Default(apperror.CodeInternal)
	}
	if err := value.WebServiceCommonInput.Normalize(); err != nil {
		return err
	}
	if err := normalizeWMSURL(&value.CapabilitiesURL); err != nil {
		return err
	}
	if err := validateWMSLayers(value.DefaultLayer, value.Layers); err != nil {
		return err
	}
	mapParameters, err := parseWMSParameters(value.GetMapParameters, "get_map_parameters", wmsGetMapReservedParameters)
	if err != nil {
		return err
	}
	featureInfoParameters, err := parseWMSParameters(value.GetFeatureInfoParameters, "get_feature_info_parameters", wmsGetFeatureInfoReservedParameters)
	if err != nil {
		return err
	}
	value.GetMapParameters = mapParameters
	value.GetFeatureInfoParameters = featureInfoParameters
	if err := validateWMSRequiredDefaults("get_map_parameters", mapParameters); err != nil {
		return err
	}
	return validateWMSRequiredDefaults("get_feature_info_parameters", featureInfoParameters)
}

func validateWMSLayers(defaultLayer string, layers []WMSLayerInput) error {
	if strings.TrimSpace(defaultLayer) == "" {
		return apperror.Field("default_layer", "Default layer must not be blank.", nil)
	}
	if strings.TrimSpace(defaultLayer) != defaultLayer {
		return apperror.Field("default_layer", "Default layer must not contain surrounding whitespace.", nil)
	}
	if len(layers) == 0 {
		return apperror.Field("layers", "At least one layer is required.", nil)
	}
	names := make(map[string]struct{}, len(layers))
	for i, layer := range layers {
		path := fmt.Sprintf("layers[%d]", i)
		if strings.TrimSpace(layer.Name) == "" {
			return apperror.Field(path+".name", "Layer name must not be blank.", nil)
		}
		if _, exists := names[layer.Name]; exists {
			return apperror.Field(path+".name", "Layer names must be unique.", nil)
		}
		names[layer.Name] = struct{}{}
		if strings.TrimSpace(layer.DefaultStyle) != layer.DefaultStyle {
			return apperror.Field(path+".default_style", "Default style must not contain surrounding whitespace.", nil)
		}
		if len(layer.Styles) == 0 {
			return apperror.Field(path+".styles", "At least one style is required.", nil)
		}
		styles := make(map[string]struct{}, len(layer.Styles))
		for j, style := range layer.Styles {
			target := fmt.Sprintf("%s.styles[%d]", path, j)
			if style != "" && strings.TrimSpace(style) == "" {
				return apperror.Field(target, "Style name must not be blank unless it is the empty default style.", nil)
			}
			if _, exists := styles[style]; exists {
				return apperror.Field(target, "Style names must be unique within a layer.", nil)
			}
			styles[style] = struct{}{}
		}
		if _, exists := styles[layer.DefaultStyle]; !exists {
			return apperror.Field(path+".default_style", "Default style must be one of the configured styles.", nil)
		}
	}
	if _, exists := names[defaultLayer]; !exists {
		return apperror.Field("default_layer", "Default layer must be one of the configured layers.", nil)
	}
	return nil
}

// ############### END INPUT ###############

type WMSWebService struct {
	WebServiceCommon
	CapabilitiesURL          string
	DefaultLayer             string
	Layers                   []WMSLayer
	TransparencyEditable     bool
	DefaultTransparency      bool
	GetMapParameters         []WebServiceParameter
	GetFeatureInfoParameters []WebServiceParameter
}

func (w *WMSWebService) ServiceType() WebServiceType {
	return WebServiceTypeWMS
}

//nolint:dupl // WMS and WMTS intentionally share validation structure.
func (w *WMSWebService) Normalize() (err error) {
	if w == nil {
		return apperror.Default(apperror.CodeInternal)
	}
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if err := w.WebServiceCommon.Normalize(); err != nil {
		return err
	}
	if err := normalizeWMSURL(&w.CapabilitiesURL); err != nil {
		return err
	}
	layers := make([]WMSLayerInput, len(w.Layers))
	for i, layer := range w.Layers {
		layers[i] = WMSLayerInput{Name: layer.Name, DefaultStyle: layer.DefaultStyle, Styles: layer.Styles}
	}
	if err := validateWMSLayers(w.DefaultLayer, layers); err != nil {
		return err
	}
	w.GetMapParameters, err = normalizePersistedOperationParameters(w.GetMapParameters, "get_map_parameters", wmsGetMapReservedParameters, parseWMSParameters)
	if err != nil {
		return err
	}
	w.GetFeatureInfoParameters, err = normalizePersistedOperationParameters(w.GetFeatureInfoParameters, "get_feature_info_parameters", wmsGetFeatureInfoReservedParameters, parseWMSParameters)
	return err
}

func normalizePersistedOperationParameters(values []WebServiceParameter, field string, reserved map[string]struct{}, parse func([]WebServiceParameterInput, string, map[string]struct{}) ([]WebServiceParameterInput, error)) ([]WebServiceParameter, error) {
	if err := validatePersistedWebServiceParameters(values); err != nil {
		return nil, err
	}
	inputs, err := parse(webServiceParameterInputs(values), field, reserved)
	if err != nil {
		return nil, err
	}
	if err := validateWMSRequiredDefaults(field, inputs); err != nil {
		return nil, err
	}
	return applyNormalizedWebServiceParameters(values, inputs)
}
