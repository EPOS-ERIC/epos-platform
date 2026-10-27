package grpcapi

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toProtoWebService(value domain.WebService) *metadatalifecyclev1.WebService {
	result := &metadatalifecyclev1.WebService{
		Id:               value.ID.String(),
		Name:             value.Name,
		Description:      value.Description,
		Type:             webServiceTypeOutput(value.Type),
		VisibleOn:        webServiceVisibleOnOutput(value.VisibleOn),
		Documentation:    documentationOutput(value.Documentation),
		SpatialCoverage:  geometryString(value.SpatialCoverage),
		TemporalCoverage: temporalOutput(value.TemporalCoverage),
	}
	switch configuration := value.Configuration.(type) {
	case domain.GenericWebServiceConfiguration:
		result.Configuration = &metadatalifecyclev1.WebService_Generic{Generic: toProtoGenericWebServiceConfiguration(configuration)}
	case domain.WFSConfiguration:
		result.Configuration = &metadatalifecyclev1.WebService_Wfs{Wfs: toProtoWFSWebServiceConfiguration(configuration)}
	case domain.WMSConfiguration:
		result.Configuration = &metadatalifecyclev1.WebService_Wms{Wms: toProtoWMSWebServiceConfiguration(configuration)}
	case domain.WMTSConfiguration:
		result.Configuration = &metadatalifecyclev1.WebService_Wmts{Wmts: toProtoWMTSWebServiceConfiguration(configuration)}
	}
	if value.Provider != nil {
		result.ProviderReference = referenceOutput(value.Provider.ID, value.Provider.Name)
	}
	for _, contact := range value.ContactPoints {
		result.ContactPoints = append(result.ContactPoints, contact.ID.String())
	}
	return result
}

func toProtoGenericWebServiceConfiguration(value domain.GenericWebServiceConfiguration) *metadatalifecyclev1.GenericWebServiceConfiguration {
	result := &metadatalifecyclev1.GenericWebServiceConfiguration{
		TemplateUrl: value.TemplateURL,
		Format:      value.Format,
		Method:      webServiceMethodOutput(value.Method),
	}
	for _, parameter := range value.Parameters {
		result.Parameters = append(result.Parameters, parameterOutput(parameter))
	}
	if value.Body != nil {
		result.BodyContentType = &value.Body.ContentType
		result.BodyTemplate = &value.Body.Template
		result.BodyRequired = new(value.Body.Required)
		for _, parameter := range value.Body.Parameters {
			result.BodyParameters = append(result.BodyParameters, scalarParameterOutput(parameter))
		}
	}
	return result
}

func toProtoWFSWebServiceConfiguration(value domain.WFSConfiguration) *metadatalifecyclev1.WFSWebServiceConfiguration {
	result := &metadatalifecyclev1.WFSWebServiceConfiguration{
		GetFeatureTemplateUrl: value.GetFeatureTemplateURL,
		Version:               value.Version,
		FeatureTypes:          value.FeatureTypes,
		DefaultFeatureType:    value.DefaultFeatureType,
		OutputFormats:         value.OutputFormats,
		DefaultOutputFormat:   value.DefaultOutputFormat,
		ResultLimit:           value.ResultLimit,
		ResultLimitEditable:   value.ResultLimitEditable,
	}
	for _, parameter := range value.Parameters {
		result.Parameters = append(result.Parameters, parameterOutput(parameter))
	}
	return result
}

func toProtoWMTSWebServiceConfiguration(value domain.WMTSConfiguration) *metadatalifecyclev1.WMTSWebServiceConfiguration {
	result := &metadatalifecyclev1.WMTSWebServiceConfiguration{CapabilitiesUrl: value.CapabilitiesURL, DefaultLayer: value.DefaultLayer}
	for _, layer := range value.Layers {
		result.Layers = append(result.Layers, &metadatalifecyclev1.WMTSLayer{Name: layer.Name, DefaultStyle: layer.DefaultStyle, Styles: layer.Styles})
	}
	for _, parameter := range value.GetTileParameters {
		result.GetTileParameters = append(result.GetTileParameters, parameterOutput(parameter))
	}
	for _, parameter := range value.GetFeatureInfoParameters {
		result.GetFeatureInfoParameters = append(result.GetFeatureInfoParameters, parameterOutput(parameter))
	}
	return result
}

func toProtoWMSWebServiceConfiguration(value domain.WMSConfiguration) *metadatalifecyclev1.WMSWebServiceConfiguration {
	result := &metadatalifecyclev1.WMSWebServiceConfiguration{
		CapabilitiesUrl:      value.CapabilitiesURL,
		DefaultLayer:         value.DefaultLayer,
		TransparencyEditable: value.TransparencyEditable,
		DefaultTransparency:  value.DefaultTransparency,
	}
	for _, parameter := range value.GetMapParameters {
		result.GetMapParameters = append(result.GetMapParameters, parameterOutput(parameter))
	}
	for _, parameter := range value.GetFeatureInfoParameters {
		result.GetFeatureInfoParameters = append(result.GetFeatureInfoParameters, parameterOutput(parameter))
	}
	for _, layer := range value.Layers {
		mapped := &metadatalifecyclev1.WMSLayer{
			Name:         layer.Name,
			DefaultStyle: layer.DefaultStyle,
			Styles:       layer.Styles,
		}
		result.Layers = append(result.Layers, mapped)
	}
	return result
}

func toDomainWebServiceInput(value *metadatalifecyclev1.WebServiceInput) (domain.WebServiceInput, error) {
	if value == nil {
		return domain.WebServiceInput{}, apperror.New(apperror.CodeValidationFailed, "web service is required") //nolint:wrapcheck
	}
	contactPoints, err := resourceParseUUIDs(value.GetContactPoints(), "web service contact point", "contact_points")
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	var providerID *uuid.UUID
	if value.ProviderId != nil {
		id, err := parseUUIDAt(value.GetProviderId(), "web service provider", "provider")
		if err != nil {
			return domain.WebServiceInput{}, err
		}
		providerID = &id
	}
	spatialCoverage, err := geometryInput(value.SpatialCoverage, "spatial_coverage")
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	temporalCoverage, err := temporalInput(value.GetTemporalCoverage())
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	documentation, err := documentationInput(value.GetDocumentation())
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	serviceType, err := webServiceTypeInput(value.GetType())
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	visibleOn, err := webServiceVisibleOnInput(value.GetVisibleOn())
	if err != nil {
		return domain.WebServiceInput{}, err
	}
	configuration, err := toDomainWebServiceConfigurationInput(value)
	if err != nil {
		return domain.WebServiceInput{}, fmt.Errorf("map web service configuration: %w", apperror.PrefixTarget(err, "configuration"))
	}
	return domain.WebServiceInput{
		Name:             value.GetName(),
		Description:      value.Description,
		Type:             serviceType,
		Configuration:    configuration,
		VisibleOn:        visibleOn,
		ProviderID:       providerID,
		Documentation:    documentation,
		SpatialCoverage:  spatialCoverage,
		TemporalCoverage: temporalCoverage,
		ContactPointIDs:  contactPoints,
	}, nil
}

func toDomainWebServiceConfigurationInput(value *metadatalifecyclev1.WebServiceInput) (domain.WebServiceConfigurationInput, error) {
	switch configuration := value.GetConfiguration().(type) {
	case *metadatalifecyclev1.WebServiceInput_Generic:
		if configuration.Generic == nil {
			return nil, apperror.Field("", "Generic web service configuration is required.", nil)
		}
		return toDomainGenericWebServiceConfigurationInput(configuration.Generic)
	case *metadatalifecyclev1.WebServiceInput_Wfs:
		if configuration.Wfs == nil {
			return nil, apperror.Field("", "WFS configuration is required.", nil)
		}
		return toDomainWFSWebServiceConfigurationInput(configuration.Wfs)
	case *metadatalifecyclev1.WebServiceInput_Wms:
		if configuration.Wms == nil {
			return nil, apperror.Field("", "WMS configuration is required.", nil)
		}
		return toDomainWMSWebServiceConfigurationInput(configuration.Wms)
	case *metadatalifecyclev1.WebServiceInput_Wmts:
		if configuration.Wmts == nil {
			return nil, apperror.Field("", "WMTS configuration is required.", nil)
		}
		return toDomainWMTSWebServiceConfigurationInput(configuration.Wmts)
	default:
		return nil, apperror.Field("", "Web service configuration is required.", nil)
	}
}

func toDomainGenericWebServiceConfigurationInput(value *metadatalifecyclev1.GenericWebServiceConfigurationInput) (domain.GenericWebServiceConfigurationInput, error) {
	method, err := webServiceMethodInput(value.GetMethod())
	if err != nil {
		return domain.GenericWebServiceConfigurationInput{}, err
	}
	parameters, err := toDomainWebServiceParameters(value.GetParameters(), "parameters")
	if err != nil {
		return domain.GenericWebServiceConfigurationInput{}, err
	}
	var body *domain.RequestBodyInput
	if value.BodyRequired != nil && value.BodyContentType == nil && value.BodyTemplate == nil && len(value.GetBodyParameters()) == 0 {
		return domain.GenericWebServiceConfigurationInput{}, apperror.Field("body_required", "Body required is only allowed with a configured request body.", nil)
	}
	if value.BodyContentType != nil || value.BodyTemplate != nil || len(value.GetBodyParameters()) > 0 {
		if value.BodyContentType == nil || value.BodyTemplate == nil {
			return domain.GenericWebServiceConfigurationInput{}, apperror.Field("body_template", "Body content type and template are required together.", nil)
		}
		bodyParameters := make([]domain.ParameterInput, 0, len(value.GetBodyParameters()))
		for i, parameter := range value.GetBodyParameters() {
			mapped, err := scalarParameterInput(parameter)
			if err != nil {
				return domain.GenericWebServiceConfigurationInput{}, fmt.Errorf("map body parameter: %w", apperror.PrefixTarget(err, fmt.Sprintf("body_parameters[%d]", i)))
			}
			bodyParameters = append(bodyParameters, mapped)
		}
		body = &domain.RequestBodyInput{
			Required:    value.BodyRequired,
			ContentType: value.GetBodyContentType(),
			Template:    value.GetBodyTemplate(),
			Parameters:  bodyParameters,
		}
	}
	return domain.GenericWebServiceConfigurationInput{
		TemplateURL: value.GetTemplateUrl(),
		Method:      method,
		Body:        body,
		Parameters:  parameters,
		Format:      value.GetFormat(),
	}, nil
}

func toDomainWFSWebServiceConfigurationInput(value *metadatalifecyclev1.WFSWebServiceConfigurationInput) (domain.WFSConfigurationInput, error) {
	parameters, err := toDomainWebServiceParameters(value.GetParameters(), "parameters")
	if err != nil {
		return domain.WFSConfigurationInput{}, err
	}
	return domain.WFSConfigurationInput{
		GetFeatureTemplateURL: value.GetGetFeatureTemplateUrl(),
		Version:               value.GetVersion(),
		FeatureTypes:          value.GetFeatureTypes(),
		DefaultFeatureType:    value.GetDefaultFeatureType(),
		OutputFormats:         value.GetOutputFormats(),
		DefaultOutputFormat:   value.GetDefaultOutputFormat(),
		ResultLimit:           value.ResultLimit,
		ResultLimitEditable:   value.GetResultLimitEditable(),
		Parameters:            parameters,
	}, nil
}

func toDomainWMTSWebServiceConfigurationInput(value *metadatalifecyclev1.WMTSWebServiceConfigurationInput) (domain.WMTSConfigurationInput, error) {
	layers := make([]domain.WMTSLayer, 0, len(value.GetLayers()))
	for i, layer := range value.GetLayers() {
		if layer == nil {
			return domain.WMTSConfigurationInput{}, apperror.Field(fmt.Sprintf("layers[%d]", i), "WMTS layer is required.", nil)
		}
		layers = append(layers, domain.WMTSLayer{Name: layer.GetName(), DefaultStyle: layer.DefaultStyle, Styles: layer.GetStyles()})
	}
	tile, err := toDomainWebServiceParameters(value.GetGetTileParameters(), "get_tile_parameters")
	if err != nil {
		return domain.WMTSConfigurationInput{}, err
	}
	info, err := toDomainWebServiceParameters(value.GetGetFeatureInfoParameters(), "get_feature_info_parameters")
	if err != nil {
		return domain.WMTSConfigurationInput{}, err
	}
	return domain.WMTSConfigurationInput{CapabilitiesURL: value.GetCapabilitiesUrl(), DefaultLayer: value.GetDefaultLayer(), Layers: layers, GetTileParameters: tile, GetFeatureInfoParameters: info}, nil
}

func toDomainWMSWebServiceConfigurationInput(value *metadatalifecyclev1.WMSWebServiceConfigurationInput) (domain.WMSConfigurationInput, error) {
	layers := make([]domain.WMSLayer, 0, len(value.GetLayers()))
	for i, layer := range value.GetLayers() {
		if layer == nil {
			return domain.WMSConfigurationInput{}, apperror.Field(fmt.Sprintf("layers[%d]", i), "WMS layer is required.", nil)
		}
		layers = append(layers, domain.WMSLayer{
			Name:         layer.GetName(),
			DefaultStyle: layer.GetDefaultStyle(),
			Styles:       layer.GetStyles(),
		})
	}
	getMapParameters, err := toDomainWebServiceParameters(value.GetGetMapParameters(), "get_map_parameters")
	if err != nil {
		return domain.WMSConfigurationInput{}, err
	}
	getFeatureInfoParameters, err := toDomainWebServiceParameters(value.GetGetFeatureInfoParameters(), "get_feature_info_parameters")
	if err != nil {
		return domain.WMSConfigurationInput{}, err
	}
	return domain.WMSConfigurationInput{
		CapabilitiesURL:          value.GetCapabilitiesUrl(),
		DefaultLayer:             value.GetDefaultLayer(),
		Layers:                   layers,
		TransparencyEditable:     value.GetTransparencyEditable(),
		DefaultTransparency:      value.GetDefaultTransparency(),
		GetMapParameters:         getMapParameters,
		GetFeatureInfoParameters: getFeatureInfoParameters,
	}, nil
}

func toDomainWebServiceParameters(values []*metadatalifecyclev1.ParameterInput, target string) ([]domain.WebServiceParameterInput, error) {
	result := make([]domain.WebServiceParameterInput, 0, len(values))
	for i, value := range values {
		mapped, err := webServiceParameterInput(value)
		if err != nil {
			return nil, fmt.Errorf("map parameter: %w", apperror.PrefixTarget(err, fmt.Sprintf("%s[%d]", target, i)))
		}
		result = append(result, mapped)
	}
	return result, nil
}
