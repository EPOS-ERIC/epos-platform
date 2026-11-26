package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toAPIDistribution(value *metadatalifecyclev1.Distribution) (gatewayapi.Distribution, error) {
	resourceID, err := parseUUID("distribution.resource_id", value.GetResourceId())
	if err != nil {
		return gatewayapi.Distribution{}, err
	}
	switch {
	case value.GetDownloadableFile() != nil:
		if value.GetType() != metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_DOWNLOADABLE_FILE {
			return gatewayapi.Distribution{}, fmt.Errorf("upstream distribution type does not match downloadable-file delivery")
		}
		v := value.GetDownloadableFile()
		fileURL, err := parseURI(v.Url)
		if err != nil {
			return gatewayapi.Distribution{}, fmt.Errorf("decode upstream downloadable_file.url: %w", err)
		}
		id, err := parseUUID("downloadable_file_distribution.id", value.Id)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		return gatewayapi.NewDownloadableFileDistributionDistribution(gatewayapi.DownloadableFileDistribution{
			ID:          id,
			ResourceID:  resourceID,
			Title:       value.Title,
			Description: value.Description,
			License:     toAPINilString(value.License),
			Type:        gatewayapi.DownloadableFileDistributionTypeDOWNLOADABLEFILE,
			ByteSize:    toAPINilInt64(value.ByteSize),
			DownloadableFile: gatewayapi.DownloadableFile{
				URL:    fileURL,
				Format: v.Format,
			},
		}), nil
	case value.GetWebService() != nil:
		if value.GetType() != metadatalifecyclev1.DistributionType_DISTRIBUTION_TYPE_WEB_SERVICE {
			return gatewayapi.Distribution{}, fmt.Errorf("upstream distribution type does not match web-service delivery")
		}
		v := value.GetWebService()
		id, err := parseUUID("web_service_distribution.id", value.Id)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		webService, err := toAPIWebService(v)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		return gatewayapi.NewWebServiceDistributionDistribution(gatewayapi.WebServiceDistribution{
			ID:          id,
			ResourceID:  resourceID,
			Title:       value.Title,
			Description: value.Description,
			License:     toAPINilString(value.License),
			Type:        gatewayapi.WebServiceDistributionTypeWEBSERVICE,
			ByteSize:    toAPINilInt64(value.ByteSize),
			WebService:  webService,
		}), nil
	default:
		return gatewayapi.Distribution{}, fmt.Errorf("unsupported upstream distribution type")
	}
}

func toAPIAvailableFormats(values []*metadatalifecyclev1.NativeFormat) ([]gatewayapi.AvailableFormat, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("upstream web-service native formats are missing")
	}

	result := make([]gatewayapi.AvailableFormat, 0, len(values))
	for _, value := range values {
		if value == nil {
			return nil, fmt.Errorf("upstream native format is nil")
		}
		result = append(result, gatewayapi.NewOriginalFormatAvailableFormat(gatewayapi.OriginalFormat{
			Kind:         gatewayapi.OriginalFormatKindORIGINAL,
			OutputFormat: value.OutputFormat,
			Label:        value.Label,
		}))
	}

	// TODO: combine native capabilities with configured CONVERTED entries when conversion support exists.
	return result, nil
}

func toAPIWebServiceCommon(value *metadatalifecyclev1.WebServiceCommon) (gatewayapi.HttpWebService, error) {
	if value == nil {
		return gatewayapi.HttpWebService{}, fmt.Errorf("upstream web-service common fields are missing")
	}
	id, err := parseUUID("web_service.id", value.Id)
	if err != nil {
		return gatewayapi.HttpWebService{}, err
	}
	resourceID, err := parseUUID("web_service.resource_id", value.GetResourceId())
	if err != nil {
		return gatewayapi.HttpWebService{}, err
	}
	visibleOn, err := toAPIWebServiceVisibleOn(value.VisibleOn)
	if err != nil {
		return gatewayapi.HttpWebService{}, err
	}
	result := gatewayapi.HttpWebService{
		ID:            id,
		ResourceID:    resourceID,
		VisibleOn:     visibleOn,
		ContactPoints: make([]uuid.UUID, 0, len(value.ContactPoints)),
	}
	if len(value.Keywords) > 0 {
		result.Keywords = toAPIKeywords(value.Keywords)
	}
	if value.ProviderReference != nil {
		providerID, err := parseUUID("web_service.provider_id", value.ProviderReference.GetId())
		if err != nil {
			return gatewayapi.HttpWebService{}, err
		}
		result.Provider.SetTo(gatewayapi.ResourceVersionReference{
			ID:   providerID,
			Name: value.ProviderReference.GetName(),
		})
	}
	documentationURI, err := toAPIOptURI("web_service.documentation_uri", value.DocumentationUri)
	if err != nil {
		return gatewayapi.HttpWebService{}, err
	}
	result.DocumentationURI = documentationURI
	for _, contactPoint := range value.ContactPoints {
		id, err := parseUUID("web_service.contact_point_id", contactPoint)
		if err != nil {
			return gatewayapi.HttpWebService{}, err
		}
		result.ContactPoints = append(result.ContactPoints, id)
	}
	return result, nil
}

func toAPIWebService(value *metadatalifecyclev1.WebService) (gatewayapi.DatasetWebService, error) {
	switch service := value.GetService().(type) {
	case *metadatalifecyclev1.WebService_Http:
		if service.Http == nil {
			break
		}
		result, err := toAPIGenericWebService(service.Http.Common, service.Http.TemplateUrl, service.Http.Method, service.Http.Format, service.Http.Parameters, service.Http.Body, service.Http.NativeFormats)
		if err != nil {
			return gatewayapi.DatasetWebService{}, err
		}
		result.Type = gatewayapi.HttpWebServiceTypeHTTP
		return gatewayapi.NewHttpWebServiceDatasetWebService(result), nil
	case *metadatalifecyclev1.WebService_Dynamic:
		if service.Dynamic == nil {
			break
		}
		v := service.Dynamic
		generic, err := toAPIGenericWebService(v.Common, v.TemplateUrl, v.Method, v.Format, v.Parameters, v.Body, v.NativeFormats)
		if err != nil {
			return gatewayapi.DatasetWebService{}, err
		}
		result := gatewayapi.DynamicWebService{
			ID:               generic.ID,
			ResourceID:       generic.ResourceID,
			VisibleOn:        generic.VisibleOn,
			Provider:         generic.Provider,
			DocumentationURI: generic.DocumentationURI,
			ContactPoints:    generic.ContactPoints,
			Keywords:         generic.Keywords,
			Type:             gatewayapi.DynamicWebServiceTypeDYNAMIC,
			TemplateURL:      generic.TemplateURL,
			Method:           generic.Method,
			Parameters:       generic.Parameters,
			BodyContentType:  generic.BodyContentType,
			BodyTemplate:     generic.BodyTemplate,
			BodyRequired:     generic.BodyRequired,
			BodyParameters:   generic.BodyParameters,
			Format:           generic.Format,
			AvailableFormats: generic.AvailableFormats,
		}
		return gatewayapi.NewDynamicWebServiceDatasetWebService(result), nil
	case *metadatalifecyclev1.WebService_Wfs:
		if service.Wfs == nil {
			break
		}
		result, err := toAPIWFSWebServiceConfiguration(service.Wfs)
		if err != nil {
			return gatewayapi.DatasetWebService{}, err
		}
		return gatewayapi.NewWfsWebServiceDatasetWebService(result), nil
	case *metadatalifecyclev1.WebService_Wms:
		if service.Wms == nil {
			break
		}
		result, err := toAPIWMSWebServiceConfiguration(service.Wms)
		if err != nil {
			return gatewayapi.DatasetWebService{}, err
		}
		return gatewayapi.NewWmsWebServiceDatasetWebService(result), nil
	case *metadatalifecyclev1.WebService_Wmts:
		if service.Wmts == nil {
			break
		}
		result, err := toAPIWMTSWebServiceConfiguration(service.Wmts)
		if err != nil {
			return gatewayapi.DatasetWebService{}, err
		}
		return gatewayapi.NewWmtsWebServiceDatasetWebService(result), nil
	}
	return gatewayapi.DatasetWebService{}, fmt.Errorf("upstream web-service variant is missing")
}

func toAPIGenericWebService(common *metadatalifecyclev1.WebServiceCommon, template string, methodValue metadatalifecyclev1.WebServiceMethod, format string, parameters []*metadatalifecyclev1.Parameter, body *metadatalifecyclev1.RequestBody, nativeFormats []*metadatalifecyclev1.NativeFormat) (gatewayapi.HttpWebService, error) {
	result, err := toAPIWebServiceCommon(common)
	if err != nil {
		return gatewayapi.HttpWebService{}, err
	}
	result.Method, err = toAPIWebServiceMethod(methodValue)
	if err != nil {
		return gatewayapi.HttpWebService{}, err
	}
	result.AvailableFormats, err = toAPIAvailableFormats(nativeFormats)
	if err != nil {
		return gatewayapi.HttpWebService{}, err
	}
	result.TemplateURL = template
	result.Format = format
	for _, parameter := range parameters {
		if parameter == nil {
			continue
		}
		mapped, err := toAPIParameter(parameter)
		if err != nil {
			return gatewayapi.HttpWebService{}, err
		}
		result.Parameters = append(result.Parameters, mapped)
	}
	if body == nil {
		return result, nil
	}
	result.BodyContentType.SetTo(body.ContentType)
	result.BodyTemplate.SetTo(body.Template)
	result.BodyRequired.SetTo(body.Required)
	for _, parameter := range body.Parameters {
		if parameter == nil {
			continue
		}
		mapped, err := toAPIScalarParameter(parameter)
		if err != nil {
			return gatewayapi.HttpWebService{}, err
		}
		result.BodyParameters = append(result.BodyParameters, mapped)
	}
	return result, nil
}

func toAPIWFSWebServiceConfiguration(value *metadatalifecyclev1.WFSWebService) (gatewayapi.WfsWebService, error) {
	common, err := toAPIWebServiceCommon(value.Common)
	if err != nil {
		return gatewayapi.WfsWebService{}, err
	}
	formats, err := toAPIAvailableFormats(value.NativeFormats)
	if err != nil {
		return gatewayapi.WfsWebService{}, err
	}
	version := gatewayapi.WfsWebServiceVersion(value.GetVersion())
	if err := version.Validate(); err != nil {
		return gatewayapi.WfsWebService{}, fmt.Errorf("upstream WFS version is invalid: %w", err)
	}
	result := gatewayapi.WfsWebService{
		ID:                    common.ID,
		ResourceID:            common.ResourceID,
		VisibleOn:             common.VisibleOn,
		Provider:              common.Provider,
		DocumentationURI:      common.DocumentationURI,
		ContactPoints:         common.ContactPoints,
		Keywords:              common.Keywords,
		AvailableFormats:      formats,
		Type:                  gatewayapi.WfsWebServiceTypeWFS,
		GetFeatureTemplateURL: value.GetGetFeatureTemplateUrl(),
		Version:               version,
		DefaultFeatureType:    value.GetDefaultFeatureType(),
		DefaultOutputFormat:   value.GetDefaultOutputFormat(),
		ResultLimit:           toAPINilInt64(value.ResultLimit),
		ResultLimitEditable:   value.GetResultLimitEditable(),
	}
	result.FeatureTypes = value.GetFeatureTypes()
	result.OutputFormats = value.GetOutputFormats()
	result.Parameters, err = toAPIParameters(value.GetParameters())
	if err != nil {
		return gatewayapi.WfsWebService{}, err
	}
	return result, nil
}

func toAPIWMTSWebServiceConfiguration(value *metadatalifecyclev1.WMTSWebService) (gatewayapi.WmtsWebService, error) {
	common, err := toAPIWebServiceCommon(value.Common)
	if err != nil {
		return gatewayapi.WmtsWebService{}, err
	}
	capabilitiesURL, err := parseURI(value.GetCapabilitiesUrl())
	if err != nil {
		return gatewayapi.WmtsWebService{}, fmt.Errorf("decode upstream wmts.capabilities_url: %w", err)
	}
	result := gatewayapi.WmtsWebService{
		ID:               common.ID,
		ResourceID:       common.ResourceID,
		VisibleOn:        common.VisibleOn,
		Provider:         common.Provider,
		DocumentationURI: common.DocumentationURI,
		ContactPoints:    common.ContactPoints,
		Keywords:         common.Keywords,
		Type:             gatewayapi.WmtsWebServiceTypeWMTS,
		CapabilitiesURL:  capabilitiesURL,
		DefaultLayer:     value.GetDefaultLayer(),
	}
	for _, layer := range value.GetLayers() {
		if layer == nil {
			return gatewayapi.WmtsWebService{}, fmt.Errorf("upstream WMTS layer is nil")
		}
		id, err := parseUUID("wmts.layer.id", layer.GetId())
		if err != nil {
			return gatewayapi.WmtsWebService{}, err
		}
		resourceID, err := parseUUID("wmts.layer.resource_id", layer.GetResourceId())
		if err != nil {
			return gatewayapi.WmtsWebService{}, err
		}

		mapped := gatewayapi.WmtsLayer{
			ID:         id,
			ResourceID: resourceID,
			Name:       layer.GetName(),
		}

		if layer.DefaultStyle != nil {
			mapped.DefaultStyle.SetTo(*layer.DefaultStyle)
		}
		mapped.Styles = layer.GetStyles()
		result.Layers = append(result.Layers, mapped)
	}
	result.GetTileParameters, err = toAPIParameters(value.GetGetTileParameters())
	if err != nil {
		return gatewayapi.WmtsWebService{}, err
	}
	result.GetFeatureInfoParameters, err = toAPIParameters(value.GetGetFeatureInfoParameters())
	if err != nil {
		return gatewayapi.WmtsWebService{}, err
	}
	return result, nil
}

func toAPIWMSWebServiceConfiguration(value *metadatalifecyclev1.WMSWebService) (gatewayapi.WmsWebService, error) {
	common, err := toAPIWebServiceCommon(value.Common)
	if err != nil {
		return gatewayapi.WmsWebService{}, err
	}
	capabilitiesURL, err := parseURI(value.GetCapabilitiesUrl())
	if err != nil {
		return gatewayapi.WmsWebService{}, fmt.Errorf("decode upstream wms.capabilities_url: %w", err)
	}
	result := gatewayapi.WmsWebService{
		ID:                   common.ID,
		ResourceID:           common.ResourceID,
		VisibleOn:            common.VisibleOn,
		Provider:             common.Provider,
		DocumentationURI:     common.DocumentationURI,
		ContactPoints:        common.ContactPoints,
		Keywords:             common.Keywords,
		Type:                 gatewayapi.WmsWebServiceTypeWMS,
		CapabilitiesURL:      capabilitiesURL,
		DefaultLayer:         value.GetDefaultLayer(),
		TransparencyEditable: value.GetTransparencyEditable(),
		DefaultTransparency:  value.GetDefaultTransparency(),
	}
	for _, layer := range value.GetLayers() {
		if layer == nil {
			return gatewayapi.WmsWebService{}, fmt.Errorf("upstream WMS layer is nil")
		}
		id, err := parseUUID("wms.layer.id", layer.GetId())
		if err != nil {
			return gatewayapi.WmsWebService{}, err
		}
		resourceID, err := parseUUID("wms.layer.resource_id", layer.GetResourceId())
		if err != nil {
			return gatewayapi.WmsWebService{}, err
		}
		mapped := gatewayapi.WmsLayer{
			ID:           id,
			ResourceID:   resourceID,
			Name:         layer.GetName(),
			DefaultStyle: layer.GetDefaultStyle(),
			Styles:       layer.GetStyles(),
		}
		result.Layers = append(result.Layers, mapped)
	}
	result.GetMapParameters, err = toAPIParameters(value.GetGetMapParameters())
	if err != nil {
		return gatewayapi.WmsWebService{}, err
	}
	result.GetFeatureInfoParameters, err = toAPIParameters(value.GetGetFeatureInfoParameters())
	if err != nil {
		return gatewayapi.WmsWebService{}, err
	}
	return result, nil
}

func toAPIParameters(values []*metadatalifecyclev1.Parameter) ([]gatewayapi.Parameter, error) {
	result := make([]gatewayapi.Parameter, 0, len(values))
	for _, value := range values {
		if value == nil {
			return nil, fmt.Errorf("upstream parameter is nil")
		}
		mapped, err := toAPIParameter(value)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func toAPIWebServiceMethod(value metadatalifecyclev1.WebServiceMethod) (gatewayapi.WebServiceMethod, error) {
	switch value {
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET:
		return gatewayapi.WebServiceMethodGET, nil
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_POST:
		return gatewayapi.WebServiceMethodPOST, nil
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_PUT:
		return gatewayapi.WebServiceMethodPUT, nil
	case metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_DELETE:
		return gatewayapi.WebServiceMethodDELETE, nil
	default:
		return "", fmt.Errorf("unsupported upstream web service method %q", value)
	}
}

func toAPIWebServiceType(value metadatalifecyclev1.WebServiceType) (gatewayapi.WebServiceType, error) {
	switch value {
	case metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WMS:
		return gatewayapi.WebServiceTypeWMS, nil
	case metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_HTTP:
		return gatewayapi.WebServiceTypeHTTP, nil
	case metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WMTS:
		return gatewayapi.WebServiceTypeWMTS, nil
	case metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WFS:
		return gatewayapi.WebServiceTypeWFS, nil
	case metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_DYNAMIC:
		return gatewayapi.WebServiceTypeDYNAMIC, nil
	default:
		return "", fmt.Errorf("unsupported upstream web service type %q", value)
	}
}

func toAPIWebServiceVisibleOn(values []metadatalifecyclev1.WebServiceVisibleOn) (gatewayapi.OptNilWebServiceVisibleOnArray, error) {
	if len(values) == 0 {
		return gatewayapi.OptNilWebServiceVisibleOnArray{}, nil
	}
	result := make([]gatewayapi.WebServiceVisibleOn, len(values))
	for i, value := range values {
		switch value {
		case metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH:
			result[i] = gatewayapi.WebServiceVisibleOnGRAPH
		case metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_TABLE:
			result[i] = gatewayapi.WebServiceVisibleOnTABLE
		case metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_MAP:
			result[i] = gatewayapi.WebServiceVisibleOnMAP
		default:
			return gatewayapi.OptNilWebServiceVisibleOnArray{}, fmt.Errorf("unsupported upstream web service visible-on value %q", value)
		}
	}
	return gatewayapi.NewOptNilWebServiceVisibleOnArray(result), nil
}

func toAPIParameter(value *metadatalifecyclev1.Parameter) (gatewayapi.Parameter, error) {
	if value == nil {
		return gatewayapi.Parameter{}, fmt.Errorf("upstream parameter is nil")
	}
	id, err := parseUUID("web_service.parameter.id", value.GetId())
	if err != nil {
		return gatewayapi.Parameter{}, err
	}
	resourceID, err := parseUUID("web_service.parameter.resource_id", value.GetResourceId())
	if err != nil {
		return gatewayapi.Parameter{}, err
	}
	description := toAPINilString(value.Description)
	role, err := toAPIParameterRole(value.Role)
	if err != nil {
		return gatewayapi.Parameter{}, err
	}
	location, err := toAPIParameterLocation(value.Location)
	if err != nil {
		return gatewayapi.Parameter{}, err
	}
	switch {
	case value.GetString_() != nil:
		v := value.GetString_()
		enum, err := toAPIStringEnumOptions(v.Enum)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewStringParameterParameter(gatewayapi.StringParameter{
			ID:             id,
			ResourceID:     resourceID,
			Kind:           gatewayapi.StringParameterKindSTRING,
			Position:       value.Position,
			Name:           value.Name,
			Label:          value.Label,
			Description:    description,
			Required:       value.Required,
			Role:           role,
			Location:       location,
			Explode:        value.Explode,
			MultipleValues: v.MultipleValues,
			ReadOnly:       value.ReadOnly,
			Default:        toAPINilStringArray(v.Default),
			Enum:           enum,
			Pattern:        toAPINilString(v.Pattern),
		}), nil
	case value.GetBoolean() != nil:
		v := value.GetBoolean()
		return gatewayapi.NewBooleanParameterParameter(gatewayapi.BooleanParameter{
			ID:          id,
			ResourceID:  resourceID,
			Kind:        gatewayapi.BooleanParameterKindBOOLEAN,
			Position:    value.Position,
			Name:        value.Name,
			Label:       value.Label,
			Description: description,
			Required:    value.Required,
			Role:        role,
			Location:    location,
			Explode:     value.Explode,
			ReadOnly:    value.ReadOnly,
			Default:     toAPINilBool(v.Default),
		}), nil
	case value.GetInteger() != nil:
		v := value.GetInteger()
		enum, err := toAPIIntegerEnumOptions(v.Enum)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewIntegerParameterParameter(gatewayapi.IntegerParameter{
			ID:             id,
			ResourceID:     resourceID,
			Kind:           gatewayapi.IntegerParameterKindINTEGER,
			Position:       value.Position,
			Name:           value.Name,
			Label:          value.Label,
			Description:    description,
			Required:       value.Required,
			Role:           role,
			Location:       location,
			Explode:        value.Explode,
			MultipleValues: v.MultipleValues,
			ReadOnly:       value.ReadOnly,
			Minimum:        toAPINilInt64(v.Minimum),
			Maximum:        toAPINilInt64(v.Maximum),
			Default:        toAPINilInt64Array(v.Default),
			Enum:           enum,
		}), nil
	case value.GetFloat() != nil:
		v := value.GetFloat()
		enum, err := toAPIFloatEnumOptions(v.Enum)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewFloatParameterParameter(gatewayapi.FloatParameter{
			ID:             id,
			ResourceID:     resourceID,
			Kind:           gatewayapi.FloatParameterKindFLOAT,
			Position:       value.Position,
			Name:           value.Name,
			Label:          value.Label,
			Description:    description,
			Required:       value.Required,
			Role:           role,
			Location:       location,
			Explode:        value.Explode,
			MultipleValues: v.MultipleValues,
			ReadOnly:       value.ReadOnly,
			Minimum:        toAPINilFloat64(v.Minimum),
			Maximum:        toAPINilFloat64(v.Maximum),
			Default:        toAPINilFloat64Array(v.Default),
			Enum:           enum,
		}), nil
	case value.GetDateTime() != nil:
		v := value.GetDateTime()
		return gatewayapi.NewDateTimeParameterParameter(gatewayapi.DateTimeParameter{
			ID:          id,
			ResourceID:  resourceID,
			Kind:        gatewayapi.DateTimeParameterKindDATETIME,
			Position:    value.Position,
			Name:        value.Name,
			Label:       value.Label,
			Description: description,
			Required:    value.Required,
			Role:        role,
			Location:    location,
			Explode:     value.Explode,
			ReadOnly:    value.ReadOnly,
			Minimum:     toAPINilTime(v.Minimum),
			Maximum:     toAPINilTime(v.Maximum),
			Default:     toAPINilTime(v.Default),
			Layout:      toAPINilString(v.Layout),
		}), nil
	case value.GetTemplate() != nil:
		v := value.GetTemplate()
		parameters := make([]gatewayapi.ScalarParameter, 0, len(v.Parameters))
		for _, parameter := range v.Parameters {
			mapped, err := toAPIScalarParameter(parameter)
			if err != nil {
				return gatewayapi.Parameter{}, err
			}
			parameters = append(parameters, mapped)
		}
		return gatewayapi.NewTemplateParameterParameter(gatewayapi.TemplateParameter{
			ID:          id,
			ResourceID:  resourceID,
			Kind:        gatewayapi.TemplateParameterKindTEMPLATE,
			Position:    value.Position,
			Name:        value.Name,
			Label:       value.Label,
			Description: description,
			Required:    value.Required,
			Role:        role,
			Location:    location,
			Explode:     value.Explode,
			ReadOnly:    value.ReadOnly,
			Template:    v.Template,
			Parameters:  parameters,
		}), nil
	default:
		return gatewayapi.Parameter{}, fmt.Errorf("unsupported upstream parameter type")
	}
}

func toAPIScalarParameter(value *metadatalifecyclev1.ScalarParameter) (gatewayapi.ScalarParameter, error) {
	if value == nil {
		return gatewayapi.ScalarParameter{}, fmt.Errorf("upstream scalar parameter is nil")
	}
	id, err := parseUUID("web_service.scalar_parameter.id", value.Id)
	if err != nil {
		return gatewayapi.ScalarParameter{}, err
	}
	resourceID, err := parseUUID("web_service.scalar_parameter.resource_id", value.GetResourceId())
	if err != nil {
		return gatewayapi.ScalarParameter{}, err
	}
	description := toAPINilString(value.Description)
	role, err := toAPIParameterRole(value.Role)
	if err != nil {
		return gatewayapi.ScalarParameter{}, err
	}
	switch {
	case value.GetString_() != nil:
		v := value.GetString_()
		enum, err := toAPIStringEnumOptions(v.Enum)
		if err != nil {
			return gatewayapi.ScalarParameter{}, err
		}
		return gatewayapi.NewStringScalarParameterScalarParameter(gatewayapi.StringScalarParameter{
			Role:        role,
			ID:          id,
			ResourceID:  resourceID,
			Kind:        gatewayapi.StringScalarParameterKindSTRING,
			Position:    value.Position,
			Name:        value.Name,
			Label:       value.Label,
			Description: description,
			Required:    value.Required,
			ReadOnly:    value.ReadOnly,
			Default:     toAPINilStringArray(v.Default),
			Enum:        enum,
			Pattern:     toAPINilString(v.Pattern),
		}), nil
	case value.GetBoolean() != nil:
		v := value.GetBoolean()
		return gatewayapi.NewBooleanScalarParameterScalarParameter(gatewayapi.BooleanScalarParameter{
			Role:        role,
			ID:          id,
			ResourceID:  resourceID,
			Kind:        gatewayapi.BooleanScalarParameterKindBOOLEAN,
			Position:    value.Position,
			Name:        value.Name,
			Label:       value.Label,
			Description: description,
			Required:    value.Required,
			ReadOnly:    value.ReadOnly,
			Default:     toAPINilBool(v.Default),
		}), nil
	case value.GetInteger() != nil:
		v := value.GetInteger()
		enum, err := toAPIIntegerEnumOptions(v.Enum)
		if err != nil {
			return gatewayapi.ScalarParameter{}, err
		}
		return gatewayapi.NewIntegerScalarParameterScalarParameter(gatewayapi.IntegerScalarParameter{
			Role:        role,
			ID:          id,
			ResourceID:  resourceID,
			Kind:        gatewayapi.IntegerScalarParameterKindINTEGER,
			Position:    value.Position,
			Name:        value.Name,
			Label:       value.Label,
			Description: description,
			Required:    value.Required,
			ReadOnly:    value.ReadOnly,
			Minimum:     toAPINilInt64(v.Minimum),
			Maximum:     toAPINilInt64(v.Maximum),
			Default:     toAPINilInt64Array(v.Default),
			Enum:        enum,
		}), nil
	case value.GetFloat() != nil:
		v := value.GetFloat()
		enum, err := toAPIFloatEnumOptions(v.Enum)
		if err != nil {
			return gatewayapi.ScalarParameter{}, err
		}
		return gatewayapi.NewFloatScalarParameterScalarParameter(gatewayapi.FloatScalarParameter{
			Role:        role,
			ID:          id,
			ResourceID:  resourceID,
			Kind:        gatewayapi.FloatScalarParameterKindFLOAT,
			Position:    value.Position,
			Name:        value.Name,
			Label:       value.Label,
			Description: description,
			Required:    value.Required,
			ReadOnly:    value.ReadOnly,
			Minimum:     toAPINilFloat64(v.Minimum),
			Maximum:     toAPINilFloat64(v.Maximum),
			Default:     toAPINilFloat64Array(v.Default),
			Enum:        enum,
		}), nil
	case value.GetDateTime() != nil:
		v := value.GetDateTime()
		return gatewayapi.NewDateTimeScalarParameterScalarParameter(gatewayapi.DateTimeScalarParameter{
			Role:        role,
			ID:          id,
			ResourceID:  resourceID,
			Kind:        gatewayapi.DateTimeScalarParameterKindDATETIME,
			Position:    value.Position,
			Name:        value.Name,
			Label:       value.Label,
			Description: description,
			Required:    value.Required,
			ReadOnly:    value.ReadOnly,
			Minimum:     toAPINilTime(v.Minimum),
			Maximum:     toAPINilTime(v.Maximum),
			Default:     toAPINilTime(v.Default),
			Layout:      toAPINilString(v.Layout),
		}), nil
	default:
		return gatewayapi.ScalarParameter{}, fmt.Errorf("unsupported upstream scalar parameter type")
	}
}

func toAPINilTime(value *timestamppb.Timestamp) gatewayapi.OptNilDateTime {
	result := gatewayapi.OptNilDateTime{}
	if value != nil {
		result.SetTo(value.AsTime())
	}
	return result
}

func toAPINilInt64(value *int64) gatewayapi.OptNilInt64 {
	result := gatewayapi.OptNilInt64{}
	if value != nil {
		result.SetTo(*value)
	}
	return result
}

func toAPINilFloat64(value *float64) gatewayapi.OptNilFloat64 {
	result := gatewayapi.OptNilFloat64{}
	if value != nil {
		result.SetTo(*value)
	}
	return result
}

func toAPINilStringArray(value []string) gatewayapi.OptNilStringArray {
	result := gatewayapi.OptNilStringArray{}
	if value != nil {
		result.SetTo(value)
	}
	return result
}

func toAPINilInt64Array(value []int64) gatewayapi.OptNilInt64Array {
	result := gatewayapi.OptNilInt64Array{}
	if value != nil {
		result.SetTo(value)
	}
	return result
}

func toAPINilFloat64Array(value []float64) gatewayapi.OptNilFloat64Array {
	result := gatewayapi.OptNilFloat64Array{}
	if value != nil {
		result.SetTo(value)
	}
	return result
}

func toAPINilBool(value *bool) gatewayapi.OptNilBool {
	result := gatewayapi.OptNilBool{}
	if value != nil {
		result.SetTo(*value)
	}
	return result
}

func toAPIParameterLocation(value metadatalifecyclev1.WebServiceParameterLocation) (gatewayapi.ParameterLocation, error) {
	switch value {
	case metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_PATH:
		return gatewayapi.ParameterLocationPATH, nil
	case metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_QUERY:
		return gatewayapi.ParameterLocationQUERY, nil
	case metadatalifecyclev1.WebServiceParameterLocation_WEB_SERVICE_PARAMETER_LOCATION_HEADER:
		return gatewayapi.ParameterLocationHEADER, nil
	default:
		return "", fmt.Errorf("unsupported upstream parameter location %q", value)
	}
}

func toAPIParameterRole(value metadatalifecyclev1.WebServiceParameterRole) (gatewayapi.OptNilParameterRole, error) {
	result := gatewayapi.OptNilParameterRole{}
	switch value {
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_UNSPECIFIED:
		return result, nil
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_FORMAT:
		result.SetTo(gatewayapi.ParameterRoleFORMAT)
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_START_DATE:
		result.SetTo(gatewayapi.ParameterRoleSTARTDATE)
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_END_DATE:
		result.SetTo(gatewayapi.ParameterRoleENDDATE)
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MIN_LON:
		result.SetTo(gatewayapi.ParameterRoleMINLON)
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MAX_LON:
		result.SetTo(gatewayapi.ParameterRoleMAXLON)
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MIN_LAT:
		result.SetTo(gatewayapi.ParameterRoleMINLAT)
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_MAX_LAT:
		result.SetTo(gatewayapi.ParameterRoleMAXLAT)
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_LONGITUDE:
		result.SetTo(gatewayapi.ParameterRoleLONGITUDE)
	case metadatalifecyclev1.WebServiceParameterRole_WEB_SERVICE_PARAMETER_ROLE_LATITUDE:
		result.SetTo(gatewayapi.ParameterRoleLATITUDE)
	default:
		return gatewayapi.OptNilParameterRole{}, fmt.Errorf("unsupported upstream parameter role %q", value)
	}
	return result, nil
}
