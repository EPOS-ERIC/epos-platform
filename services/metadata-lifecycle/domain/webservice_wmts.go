package domain

import (
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

type WMTSLayerInput struct {
	ResourceID   *uuid.UUID
	Name         string
	DefaultStyle *string
	Styles       []string
}

type WMTSLayer struct {
	ID           uuid.UUID
	ResourceID   uuid.UUID
	Name         string
	DefaultStyle *string
	Styles       []string
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
			return nil, apperror.Field(path+".location", "WMTS operation parameters must use QUERY location.", nil)
		}
		if _, exists := reserved[strings.ToUpper(parameter.Name)]; exists {
			return nil, apperror.Field(path+".name", "Standard WMTS parameters cannot be redefined.", nil)
		}
	}
	if err := validateWMSRequiredDefaults(field, parameters); err != nil {
		return nil, err
	}
	return parameters, nil
}

// ############### INPUT ###############

type WMTSWebServiceInput struct {
	WebServiceCommonInput
	CapabilitiesURL          string
	DefaultLayer             string
	Layers                   []WMTSLayerInput
	GetTileParameters        []WebServiceParameterInput
	GetFeatureInfoParameters []WebServiceParameterInput
}

func (*WMTSWebServiceInput) ServiceType() WebServiceType {
	return WebServiceTypeWMTS
}

func (value *WMTSWebServiceInput) Normalize() error {
	if value == nil {
		return apperror.Default(apperror.CodeInternal)
	}
	if err := value.WebServiceCommonInput.Normalize(); err != nil {
		return err
	}
	if err := normalizeWMSURL(&value.CapabilitiesURL); err != nil {
		return err
	}
	if err := validateWMTSLayers(value.DefaultLayer, value.Layers); err != nil {
		return err
	}
	var err error
	value.GetTileParameters, err = parseWMTSParameters(value.GetTileParameters, "get_tile_parameters", wmtsGetTileReservedParameters)
	if err != nil {
		return err
	}
	value.GetFeatureInfoParameters, err = parseWMTSParameters(value.GetFeatureInfoParameters, "get_feature_info_parameters", wmtsGetFeatureInfoReservedParameters)
	return err
}

func validateWMTSLayers(defaultLayer string, layers []WMTSLayerInput) error {
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
		if layer.DefaultStyle != nil && (*layer.DefaultStyle == "" || strings.TrimSpace(*layer.DefaultStyle) != *layer.DefaultStyle) {
			return apperror.Field(path+".default_style", "Default style must not be blank or contain surrounding whitespace.", nil)
		}
		styles := make(map[string]struct{}, len(layer.Styles))
		for j, style := range layer.Styles {
			target := fmt.Sprintf("%s.styles[%d]", path, j)
			if strings.TrimSpace(style) == "" {
				return apperror.Field(target, "Style name must not be blank.", nil)
			}
			if _, exists := styles[style]; exists {
				return apperror.Field(target, "Style names must be unique within a layer.", nil)
			}
			styles[style] = struct{}{}
		}
		if layer.DefaultStyle != nil {
			if _, exists := styles[*layer.DefaultStyle]; !exists {
				return apperror.Field(path+".default_style", "Default style must be one of the configured styles.", nil)
			}
		}
	}
	if _, exists := names[defaultLayer]; !exists || strings.TrimSpace(defaultLayer) != defaultLayer {
		return apperror.Field("default_layer", "Default layer must be one of the configured layers.", nil)
	}
	return nil
}

// ############### END INPUT ###############

type WMTSWebService struct {
	WebServiceCommon
	CapabilitiesURL          string
	DefaultLayer             string
	Layers                   []WMTSLayer
	GetTileParameters        []WebServiceParameter
	GetFeatureInfoParameters []WebServiceParameter
}

func (w *WMTSWebService) ServiceType() WebServiceType {
	return WebServiceTypeWMTS
}

//nolint:dupl // WMS and WMTS intentionally share validation structure.
func (w *WMTSWebService) Normalize() (err error) {
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
	layers := make([]WMTSLayerInput, len(w.Layers))
	for i, layer := range w.Layers {
		layers[i] = WMTSLayerInput{Name: layer.Name, DefaultStyle: layer.DefaultStyle, Styles: layer.Styles}
	}
	if err := validateWMTSLayers(w.DefaultLayer, layers); err != nil {
		return err
	}
	w.GetTileParameters, err = normalizePersistedOperationParameters(w.GetTileParameters, "get_tile_parameters", wmtsGetTileReservedParameters, parseWMTSParameters)
	if err != nil {
		return err
	}
	w.GetFeatureInfoParameters, err = normalizePersistedOperationParameters(w.GetFeatureInfoParameters, "get_feature_info_parameters", wmtsGetFeatureInfoReservedParameters, parseWMTSParameters)
	return err
}
