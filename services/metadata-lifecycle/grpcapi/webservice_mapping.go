package grpcapi

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func toProtoWebService(value domain.WebService) *metadatalifecyclev1.WebService {
	common := value.Common()
	commonOutput := &metadatalifecyclev1.WebServiceCommon{
		Id:               common.ID.String(),
		ResourceId:       common.ResourceID.String(),
		VisibleOn:        webServiceVisibleOnOutput(common.VisibleOn),
		DocumentationUri: common.DocumentationURI,
		Keywords:         common.Keywords,
		AccessRights:     common.AccessRights,
		License:          common.License,
	}
	if common.Provider != nil {
		commonOutput.ProviderReference = referenceOutput(common.Provider.ID, common.Provider.Name)
	}
	for _, contact := range common.ContactPoints {
		commonOutput.ContactPoints = append(commonOutput.ContactPoints, contact.ID.String())
	}
	commonOutput.Categories = categoryReferences(common.Categories)
	result := &metadatalifecyclev1.WebService{}
	switch service := value.(type) {
	case *domain.HTTPWebService:
		mapped := toProtoHTTPWebService(service.GenericWebService)
		mapped.Common = commonOutput
		mapped.NativeFormats = toProtoNativeFormats(service.NativeFormats())
		result.Service = &metadatalifecyclev1.WebService_Http{Http: mapped}
	case *domain.DynamicWebService:
		generic := toProtoHTTPWebService(service.GenericWebService)
		result.Service = &metadatalifecyclev1.WebService_Dynamic{Dynamic: &metadatalifecyclev1.DynamicWebService{
			Common:        commonOutput,
			TemplateUrl:   generic.TemplateUrl,
			Format:        generic.Format,
			Method:        generic.Method,
			Parameters:    generic.Parameters,
			Body:          generic.Body,
			NativeFormats: toProtoNativeFormats(service.NativeFormats()),
		}}
	case *domain.WFSWebService:
		mapped := toProtoWFSWebServiceConfiguration(service)
		mapped.Common = commonOutput
		mapped.NativeFormats = toProtoNativeFormats(service.NativeFormats())
		result.Service = &metadatalifecyclev1.WebService_Wfs{Wfs: mapped}
	case *domain.WMSWebService:
		mapped := toProtoWMSWebServiceConfiguration(service)
		mapped.Common = commonOutput
		result.Service = &metadatalifecyclev1.WebService_Wms{Wms: mapped}
	case *domain.WMTSWebService:
		mapped := toProtoWMTSWebServiceConfiguration(service)
		mapped.Common = commonOutput
		result.Service = &metadatalifecyclev1.WebService_Wmts{Wmts: mapped}
	}
	return result
}

func toProtoHTTPWebService(value domain.GenericWebService) *metadatalifecyclev1.HTTPWebService {
	result := &metadatalifecyclev1.HTTPWebService{
		TemplateUrl: value.TemplateURL,
		Format:      value.Format,
		Method:      webServiceMethodOutput(value.Method),
	}
	for _, parameter := range value.Parameters {
		result.Parameters = append(result.Parameters, parameterOutput(parameter))
	}
	if value.Body != nil {
		result.Body = &metadatalifecyclev1.RequestBody{
			ContentType: value.Body.ContentType,
			Template:    value.Body.Template,
			Required:    value.Body.Required,
		}
		for _, parameter := range value.Body.Parameters {
			result.Body.Parameters = append(result.Body.Parameters, scalarParameterOutput(parameter))
		}
	}
	return result
}

func toProtoWFSWebServiceConfiguration(value *domain.WFSWebService) *metadatalifecyclev1.WFSWebService {
	result := &metadatalifecyclev1.WFSWebService{
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

func toProtoWMTSWebServiceConfiguration(value *domain.WMTSWebService) *metadatalifecyclev1.WMTSWebService {
	result := &metadatalifecyclev1.WMTSWebService{CapabilitiesUrl: value.CapabilitiesURL, DefaultLayer: value.DefaultLayer}
	for _, layer := range value.Layers {
		result.Layers = append(result.Layers, &metadatalifecyclev1.WMTSLayer{
			Id:           layer.ID.String(),
			ResourceId:   layer.ResourceID.String(),
			Name:         layer.Name,
			DefaultStyle: layer.DefaultStyle,
			Styles:       layer.Styles,
		})
	}
	for _, parameter := range value.GetTileParameters {
		result.GetTileParameters = append(result.GetTileParameters, parameterOutput(parameter))
	}
	for _, parameter := range value.GetFeatureInfoParameters {
		result.GetFeatureInfoParameters = append(result.GetFeatureInfoParameters, parameterOutput(parameter))
	}
	return result
}

func toProtoWMSWebServiceConfiguration(value *domain.WMSWebService) *metadatalifecyclev1.WMSWebService {
	result := &metadatalifecyclev1.WMSWebService{
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
			Id:           layer.ID.String(),
			ResourceId:   layer.ResourceID.String(),
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
		return nil, apperror.New(apperror.CodeValidationFailed, "web service is required") //nolint:wrapcheck
	}
	var commonValue *metadatalifecyclev1.WebServiceCommonInput
	var service domain.WebServiceInput
	var err error
	switch v := value.GetService().(type) {
	case *metadatalifecyclev1.WebServiceInput_Http:
		if v.Http == nil {
			return nil, apperror.Field("", "HTTP web service is required.", nil)
		}
		commonValue = v.Http.Common
		var generic domain.GenericWebServiceInput
		generic, err = toDomainGenericWebServiceInput(v.Http.GetTemplateUrl(), v.Http.GetMethod(), v.Http.GetFormat(), v.Http.GetParameters(), v.Http.GetBody())
		service = &domain.HTTPWebServiceInput{GenericWebServiceInput: generic}
	case *metadatalifecyclev1.WebServiceInput_Dynamic:
		if v.Dynamic == nil {
			return nil, apperror.Field("", "DYNAMIC web service is required.", nil)
		}
		commonValue = v.Dynamic.Common
		var generic domain.GenericWebServiceInput
		generic, err = toDomainGenericWebServiceInput(v.Dynamic.GetTemplateUrl(), v.Dynamic.GetMethod(), v.Dynamic.GetFormat(), v.Dynamic.GetParameters(), v.Dynamic.GetBody())
		service = &domain.DynamicWebServiceInput{GenericWebServiceInput: generic}
	case *metadatalifecyclev1.WebServiceInput_Wfs:
		if v.Wfs == nil {
			return nil, apperror.Field("", "WFS web service is required.", nil)
		}
		commonValue = v.Wfs.Common
		service, err = toDomainWFSWebServiceInput(v.Wfs)
	case *metadatalifecyclev1.WebServiceInput_Wms:
		if v.Wms == nil {
			return nil, apperror.Field("", "WMS web service is required.", nil)
		}
		commonValue = v.Wms.Common
		service, err = toDomainWMSWebServiceInput(v.Wms)
	case *metadatalifecyclev1.WebServiceInput_Wmts:
		if v.Wmts == nil {
			return nil, apperror.Field("", "WMTS web service is required.", nil)
		}
		commonValue = v.Wmts.Common
		service, err = toDomainWMTSWebServiceInput(v.Wmts)
	default:
		return nil, apperror.Field("", "Web service is required.", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("map web service: %w", err)
	}
	if commonValue == nil {
		return service, nil
	}
	var resourceID *uuid.UUID
	if commonValue.ResourceId != nil {
		id, err := parseUUIDAt(*commonValue.ResourceId, "web service resource ID", "resource_id")
		if err != nil {
			return nil, err
		}
		resourceID = &id
	}
	contactPoints, err := resourceParseUUIDs(commonValue.GetContactPoints(), "web service contact point", "contact_points")
	if err != nil {
		return nil, err
	}
	categories, err := resourceParseUUIDs(commonValue.GetCategories(), "web service category", "categories")
	if err != nil {
		return nil, err
	}
	var providerID *uuid.UUID
	if commonValue.ProviderId != nil {
		id, err := parseUUIDAt(commonValue.GetProviderId(), "web service provider", "provider")
		if err != nil {
			return nil, err
		}
		providerID = &id
	}
	visibleOn, err := webServiceVisibleOnInput(commonValue.GetVisibleOn())
	if err != nil {
		return nil, err
	}
	common := domain.WebServiceCommonInput{
		ResourceID:       resourceID,
		VisibleOn:        visibleOn,
		ProviderID:       providerID,
		DocumentationURI: commonValue.DocumentationUri,
		ContactPointIDs:  contactPoints,
		Keywords:         commonValue.Keywords,
		AccessRights:     commonValue.AccessRights,
		License:          commonValue.License,
		CategoryIDs:      categories,
	}
	*service.CommonInput() = common
	return service, nil
}

func toDomainGenericWebServiceInput(template string, methodValue metadatalifecyclev1.WebServiceMethod, format string, parameterValues []*metadatalifecyclev1.ParameterInput, bodyValue *metadatalifecyclev1.RequestBodyInput) (domain.GenericWebServiceInput, error) {
	method, err := webServiceMethodInput(methodValue)
	if err != nil {
		return domain.GenericWebServiceInput{}, err
	}
	parameters, err := toDomainWebServiceParameters(parameterValues, "parameters")
	if err != nil {
		return domain.GenericWebServiceInput{}, err
	}
	var body *domain.RequestBodyInput
	if bodyValue != nil {
		bodyParameters := make([]domain.ParameterInput, 0, len(bodyValue.GetParameters()))
		for i, parameter := range bodyValue.GetParameters() {
			mapped, err := scalarParameterInput(parameter)
			if err != nil {
				return domain.GenericWebServiceInput{}, fmt.Errorf("map body parameter: %w", apperror.PrefixTarget(err, fmt.Sprintf("body_parameters[%d]", i)))
			}
			bodyParameters = append(bodyParameters, mapped)
		}
		body = &domain.RequestBodyInput{
			Required:    bodyValue.Required,
			ContentType: bodyValue.GetContentType(),
			Template:    bodyValue.GetTemplate(),
			Parameters:  bodyParameters,
		}
	}
	return domain.GenericWebServiceInput{
		TemplateURL: template,
		Method:      method,
		Body:        body,
		Parameters:  parameters,
		Format:      format,
	}, nil
}

func toDomainWFSWebServiceInput(value *metadatalifecyclev1.WFSWebServiceInput) (*domain.WFSWebServiceInput, error) {
	parameters, err := toDomainWebServiceParameters(value.GetParameters(), "parameters")
	if err != nil {
		return nil, err
	}
	return &domain.WFSWebServiceInput{
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

func toDomainWMTSWebServiceInput(value *metadatalifecyclev1.WMTSWebServiceInput) (*domain.WMTSWebServiceInput, error) {
	layers := make([]domain.WMTSLayerInput, 0, len(value.GetLayers()))
	for i, layer := range value.GetLayers() {
		if layer == nil {
			return nil, apperror.Field(fmt.Sprintf("layers[%d]", i), "WMTS layer is required.", nil)
		}
		resourceID, err := layerResourceID(layer.ResourceId, i)
		if err != nil {
			return nil, err
		}
		layers = append(layers, domain.WMTSLayerInput{
			ResourceID:   resourceID,
			Name:         layer.GetName(),
			DefaultStyle: layer.DefaultStyle,
			Styles:       layer.GetStyles(),
		})
	}
	tile, err := toDomainWebServiceParameters(value.GetGetTileParameters(), "get_tile_parameters")
	if err != nil {
		return nil, err
	}
	info, err := toDomainWebServiceParameters(value.GetGetFeatureInfoParameters(), "get_feature_info_parameters")
	if err != nil {
		return nil, err
	}
	return &domain.WMTSWebServiceInput{CapabilitiesURL: value.GetCapabilitiesUrl(), DefaultLayer: value.GetDefaultLayer(), Layers: layers, GetTileParameters: tile, GetFeatureInfoParameters: info}, nil
}

func toDomainWMSWebServiceInput(value *metadatalifecyclev1.WMSWebServiceInput) (*domain.WMSWebServiceInput, error) {
	layers := make([]domain.WMSLayerInput, 0, len(value.GetLayers()))
	for i, layer := range value.GetLayers() {
		if layer == nil {
			return nil, apperror.Field(fmt.Sprintf("layers[%d]", i), "WMS layer is required.", nil)
		}
		resourceID, err := layerResourceID(layer.ResourceId, i)
		if err != nil {
			return nil, err
		}
		layers = append(layers, domain.WMSLayerInput{
			ResourceID:   resourceID,
			Name:         layer.GetName(),
			DefaultStyle: layer.GetDefaultStyle(),
			Styles:       layer.GetStyles(),
		})
	}
	getMapParameters, err := toDomainWebServiceParameters(value.GetGetMapParameters(), "get_map_parameters")
	if err != nil {
		return nil, err
	}
	getFeatureInfoParameters, err := toDomainWebServiceParameters(value.GetGetFeatureInfoParameters(), "get_feature_info_parameters")
	if err != nil {
		return nil, err
	}
	return &domain.WMSWebServiceInput{
		CapabilitiesURL:          value.GetCapabilitiesUrl(),
		DefaultLayer:             value.GetDefaultLayer(),
		Layers:                   layers,
		TransparencyEditable:     value.GetTransparencyEditable(),
		DefaultTransparency:      value.GetDefaultTransparency(),
		GetMapParameters:         getMapParameters,
		GetFeatureInfoParameters: getFeatureInfoParameters,
	}, nil
}

func layerResourceID(value *string, index int) (*uuid.UUID, error) {
	if value == nil {
		return nil, nil
	}
	id, err := parseUUIDAt(*value, "layer resource ID", fmt.Sprintf("layers[%d].resource_id", index))
	if err != nil {
		return nil, err
	}
	return &id, nil
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
