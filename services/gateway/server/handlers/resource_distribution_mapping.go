package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toAPIDistribution(value *metadatalifecyclev1.Distribution) (gatewayapi.Distribution, error) {
	switch {
	case value.GetDownloadableFile() != nil:
		v := value.GetDownloadableFile()
		id, err := parseUUID("downloadable_file_distribution.id", value.Id)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		return gatewayapi.NewDownloadableFileDistributionDistribution(gatewayapi.DownloadableFileDistribution{
			ID:          id,
			Title:       value.Title,
			Description: value.Description,
			License:     toAPINilString(value.License),
			Type:        gatewayapi.DownloadableFileDistributionType(value.Type),
			Issued:      toAPINilTime(value.Issued),
			Modified:    toAPINilTime(value.Modified),
			ByteSize:    toAPINilInt64(value.ByteSize),
			DownloadableFile: gatewayapi.DownloadableFile{
				URL:    gatewayapi.NonBlankString(v.Url),
				Format: gatewayapi.NonBlankString(v.Format),
			},
		}), nil
	case value.GetWebService() != nil:
		v := value.GetWebService()
		id, err := parseUUID("web_service_distribution.id", value.Id)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		webService, err := toAPIWebService(v)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		formats, err := toAPIAvailableFormats(value.NativeFormats)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		return gatewayapi.NewWebServiceDistributionDistribution(gatewayapi.WebServiceDistribution{
			ID:               id,
			Title:            value.Title,
			Description:      value.Description,
			License:          toAPINilString(value.License),
			Type:             gatewayapi.WebServiceDistributionType(value.Type),
			Issued:           toAPINilTime(value.Issued),
			Modified:         toAPINilTime(value.Modified),
			ByteSize:         toAPINilInt64(value.ByteSize),
			WebService:       webService,
			AvailableFormats: formats,
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

func toAPIWebService(value *metadatalifecyclev1.WebService) (gatewayapi.WebService, error) {
	id, err := parseUUID("web_service.id", value.Id)
	if err != nil {
		return gatewayapi.WebService{}, err
	}
	visibleOn, err := toAPIWebServiceVisibleOn(value.VisibleOn)
	if err != nil {
		return gatewayapi.WebService{}, err
	}
	configuration, err := toAPIWebServiceConfiguration(value)
	if err != nil {
		return gatewayapi.WebService{}, err
	}
	result := gatewayapi.WebService{
		ID:               id,
		Name:             value.Name,
		Description:      toAPINilString(value.Description),
		VisibleOn:        visibleOn,
		Configuration:    configuration,
		SpatialCoverage:  toAPINilString(value.SpatialCoverage),
		TemporalCoverage: toAPITemporalCoverage(value.TemporalCoverage),
		ContactPoints:    make([]uuid.UUID, 0, len(value.ContactPoints)),
	}
	if value.ProviderReference != nil {
		providerID, err := parseUUID("web_service.provider_id", value.ProviderReference.GetId())
		if err != nil {
			return gatewayapi.WebService{}, err
		}
		result.Provider.SetTo(gatewayapi.ResourceVersionReference{
			ID:   providerID,
			Name: value.ProviderReference.GetName(),
		})
	}
	if value.Documentation != nil {
		result.Documentation = gatewayapi.NewOptDocumentation(gatewayapi.Documentation{
			URI:         gatewayapi.NonBlankString(value.Documentation.GetUri()),
			Title:       toAPINilString(value.Documentation.Title),
			Description: toAPINilString(value.Documentation.Description),
		})
	}
	for _, contactPoint := range value.ContactPoints {
		id, err := parseUUID("web_service.contact_point_id", contactPoint)
		if err != nil {
			return gatewayapi.WebService{}, err
		}
		result.ContactPoints = append(result.ContactPoints, id)
	}
	return result, nil
}

func toAPIWebServiceConfiguration(value *metadatalifecyclev1.WebService) (gatewayapi.WebServiceConfiguration, error) {
	switch configuration := value.GetConfiguration().(type) {
	case *metadatalifecyclev1.WebService_Generic:
		if configuration.Generic == nil {
			return gatewayapi.WebServiceConfiguration{}, fmt.Errorf("upstream generic web-service configuration is nil")
		}
		generic, err := toAPIGenericWebServiceConfiguration(configuration.Generic)
		if err != nil {
			return gatewayapi.WebServiceConfiguration{}, err
		}
		serviceType, err := toAPIWebServiceType(value.GetType())
		if err != nil {
			return gatewayapi.WebServiceConfiguration{}, err
		}
		switch serviceType {
		case gatewayapi.WebServiceTypeWMS, gatewayapi.WebServiceTypeWFS:
			return gatewayapi.WebServiceConfiguration{}, fmt.Errorf("upstream generic configuration has incompatible web-service type %q", serviceType)
		case gatewayapi.WebServiceTypeHTTP:
			return gatewayapi.NewWebServiceConfigurationHTTPWebServiceConfiguration(generic), nil
		case gatewayapi.WebServiceTypeWMTS:
			return gatewayapi.NewWebServiceConfigurationWMTSWebServiceConfiguration(generic), nil
		default:
			return gatewayapi.WebServiceConfiguration{}, fmt.Errorf("unsupported generic upstream web-service type %q", value.GetType())
		}
	case *metadatalifecyclev1.WebService_Wfs:
		if configuration.Wfs == nil {
			return gatewayapi.WebServiceConfiguration{}, fmt.Errorf("upstream WFS configuration is nil")
		}
		if value.GetType() != metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WFS {
			return gatewayapi.WebServiceConfiguration{}, fmt.Errorf("upstream WFS configuration has incompatible web-service type %q", value.GetType())
		}
		wfs, err := toAPIWFSWebServiceConfiguration(configuration.Wfs)
		if err != nil {
			return gatewayapi.WebServiceConfiguration{}, err
		}
		return gatewayapi.NewWfsWebServiceConfigurationWebServiceConfiguration(wfs), nil
	case *metadatalifecyclev1.WebService_Wms:
		if configuration.Wms == nil {
			return gatewayapi.WebServiceConfiguration{}, fmt.Errorf("upstream WMS configuration is nil")
		}
		if value.GetType() != metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WMS {
			return gatewayapi.WebServiceConfiguration{}, fmt.Errorf("upstream WMS configuration has incompatible web-service type %q", value.GetType())
		}
		wms, err := toAPIWMSWebServiceConfiguration(configuration.Wms)
		if err != nil {
			return gatewayapi.WebServiceConfiguration{}, err
		}
		return gatewayapi.NewWmsWebServiceConfigurationWebServiceConfiguration(wms), nil
	default:
		return gatewayapi.WebServiceConfiguration{}, fmt.Errorf("upstream web-service configuration is missing")
	}
}

func toAPIGenericWebServiceConfiguration(value *metadatalifecyclev1.GenericWebServiceConfiguration) (gatewayapi.GenericWebServiceConfiguration, error) {
	method, err := toAPIWebServiceMethod(value.GetMethod())
	if err != nil {
		return gatewayapi.GenericWebServiceConfiguration{}, err
	}
	result := gatewayapi.GenericWebServiceConfiguration{
		TemplateURL: value.GetTemplateUrl(),
		Method:      method,
		Format:      value.GetFormat(),
	}
	for _, parameter := range value.GetParameters() {
		if parameter == nil {
			continue
		}
		mapped, err := toAPIParameter(parameter)
		if err != nil {
			return gatewayapi.GenericWebServiceConfiguration{}, err
		}
		result.Parameters = append(result.Parameters, mapped)
	}
	result.BodyContentType = toAPINilString(value.BodyContentType)
	result.BodyTemplate = toAPINilString(value.BodyTemplate)
	if value.BodyRequired != nil {
		result.BodyRequired.SetTo(*value.BodyRequired)
	}
	for _, parameter := range value.GetBodyParameters() {
		if parameter == nil {
			continue
		}
		mapped, err := toAPIScalarParameter(parameter)
		if err != nil {
			return gatewayapi.GenericWebServiceConfiguration{}, err
		}
		result.BodyParameters = append(result.BodyParameters, mapped)
	}
	return result, nil
}

func toAPIWFSWebServiceConfiguration(value *metadatalifecyclev1.WFSWebServiceConfiguration) (gatewayapi.WfsWebServiceConfiguration, error) {
	version := gatewayapi.WfsWebServiceConfigurationVersion(value.GetVersion())
	if err := version.Validate(); err != nil {
		return gatewayapi.WfsWebServiceConfiguration{}, fmt.Errorf("upstream WFS version is invalid: %w", err)
	}
	result := gatewayapi.WfsWebServiceConfiguration{
		Type:                  gatewayapi.WfsWebServiceConfigurationTypeWFS,
		GetFeatureTemplateURL: value.GetGetFeatureTemplateUrl(),
		Version:               version,
		DefaultFeatureType:    gatewayapi.NonBlankString(value.GetDefaultFeatureType()),
		DefaultOutputFormat:   gatewayapi.NonBlankString(value.GetDefaultOutputFormat()),
		ResultLimit:           toAPINilInt64(value.ResultLimit),
		ResultLimitEditable:   value.GetResultLimitEditable(),
		Parameters:            make([]gatewayapi.Parameter, 0, len(value.GetParameters())),
	}
	for _, featureType := range value.GetFeatureTypes() {
		result.FeatureTypes = append(result.FeatureTypes, gatewayapi.NonBlankString(featureType))
	}
	for _, format := range value.GetOutputFormats() {
		result.OutputFormats = append(result.OutputFormats, gatewayapi.NonBlankString(format))
	}
	for _, parameter := range value.GetParameters() {
		if parameter == nil {
			return gatewayapi.WfsWebServiceConfiguration{}, fmt.Errorf("upstream WFS parameter is nil")
		}
		mapped, err := toAPIParameter(parameter)
		if err != nil {
			return gatewayapi.WfsWebServiceConfiguration{}, err
		}
		result.Parameters = append(result.Parameters, mapped)
	}
	return result, nil
}

func toAPIWMSWebServiceConfiguration(value *metadatalifecyclev1.WMSWebServiceConfiguration) (gatewayapi.WmsWebServiceConfiguration, error) {
	result := gatewayapi.WmsWebServiceConfiguration{
		Type:                 gatewayapi.WmsWebServiceConfigurationTypeWMS,
		CapabilitiesURL:      value.GetCapabilitiesUrl(),
		DefaultLayer:         gatewayapi.NonBlankString(value.GetDefaultLayer()),
		TransparencyEditable: value.GetTransparencyEditable(),
		DefaultTransparency:  value.GetDefaultTransparency(),
	}
	for _, layer := range value.GetLayers() {
		if layer == nil {
			return gatewayapi.WmsWebServiceConfiguration{}, fmt.Errorf("upstream WMS layer is nil")
		}
		mapped := gatewayapi.WmsLayer{
			Name:         gatewayapi.NonBlankString(layer.GetName()),
			DefaultStyle: layer.GetDefaultStyle(),
			Styles:       layer.GetStyles(),
		}
		result.Layers = append(result.Layers, mapped)
	}
	var err error
	result.GetMapParameters, err = toAPIWMSParameters(value.GetGetMapParameters())
	if err != nil {
		return gatewayapi.WmsWebServiceConfiguration{}, err
	}
	result.GetFeatureInfoParameters, err = toAPIWMSParameters(value.GetGetFeatureInfoParameters())
	if err != nil {
		return gatewayapi.WmsWebServiceConfiguration{}, err
	}
	return result, nil
}

func toAPIWMSParameters(values []*metadatalifecyclev1.Parameter) ([]gatewayapi.Parameter, error) {
	result := make([]gatewayapi.Parameter, 0, len(values))
	for _, value := range values {
		if value == nil {
			return nil, fmt.Errorf("upstream WMS parameter is nil")
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
	description := toAPINilString(value.Description)
	property := toAPINilString(value.Property)
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
			Kind:           gatewayapi.StringParameterKindSTRING,
			Position:       value.Position,
			Name:           gatewayapi.NonBlankString(value.Name),
			Label:          gatewayapi.NonBlankString(value.Label),
			Description:    description,
			Required:       value.Required,
			Property:       property,
			Role:           role,
			Location:       location,
			Explode:        value.Explode,
			MultipleValues: v.MultipleValues,
			ReadOnly:       value.ReadOnly,
			Default:        toAPINilString(v.Default),
			Enum:           enum,
			Pattern:        toAPINilString(v.Pattern),
		}), nil
	case value.GetBoolean() != nil:
		v := value.GetBoolean()
		return gatewayapi.NewBooleanParameterParameter(gatewayapi.BooleanParameter{
			ID:          id,
			Kind:        gatewayapi.BooleanParameterKindBOOLEAN,
			Position:    value.Position,
			Name:        gatewayapi.NonBlankString(value.Name),
			Label:       gatewayapi.NonBlankString(value.Label),
			Description: description,
			Required:    value.Required,
			Property:    property,
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
			Kind:           gatewayapi.IntegerParameterKindINTEGER,
			Position:       value.Position,
			Name:           gatewayapi.NonBlankString(value.Name),
			Label:          gatewayapi.NonBlankString(value.Label),
			Description:    description,
			Required:       value.Required,
			Property:       property,
			Role:           role,
			Location:       location,
			Explode:        value.Explode,
			MultipleValues: v.MultipleValues,
			ReadOnly:       value.ReadOnly,
			Minimum:        toAPINilInt64(v.Minimum),
			Maximum:        toAPINilInt64(v.Maximum),
			Default:        toAPINilInt64(v.Default),
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
			Kind:           gatewayapi.FloatParameterKindFLOAT,
			Position:       value.Position,
			Name:           gatewayapi.NonBlankString(value.Name),
			Label:          gatewayapi.NonBlankString(value.Label),
			Description:    description,
			Required:       value.Required,
			Property:       property,
			Role:           role,
			Location:       location,
			Explode:        value.Explode,
			MultipleValues: v.MultipleValues,
			ReadOnly:       value.ReadOnly,
			Minimum:        toAPINilFloat64(v.Minimum),
			Maximum:        toAPINilFloat64(v.Maximum),
			Default:        toAPINilFloat64(v.Default),
			Enum:           enum,
		}), nil
	case value.GetDateTime() != nil:
		v := value.GetDateTime()
		return gatewayapi.NewDateTimeParameterParameter(gatewayapi.DateTimeParameter{
			ID:          id,
			Kind:        gatewayapi.DateTimeParameterKindDATETIME,
			Position:    value.Position,
			Name:        gatewayapi.NonBlankString(value.Name),
			Label:       gatewayapi.NonBlankString(value.Label),
			Description: description,
			Required:    value.Required,
			Property:    property,
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
			Kind:        gatewayapi.TemplateParameterKindTEMPLATE,
			Position:    value.Position,
			Name:        gatewayapi.NonBlankString(value.Name),
			Label:       gatewayapi.NonBlankString(value.Label),
			Description: description,
			Required:    value.Required,
			Property:    property,
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
	description := toAPINilString(value.Description)
	switch {
	case value.GetString_() != nil:
		v := value.GetString_()
		enum, err := toAPIStringEnumOptions(v.Enum)
		if err != nil {
			return gatewayapi.ScalarParameter{}, err
		}
		return gatewayapi.NewStringScalarParameterScalarParameter(gatewayapi.StringScalarParameter{
			ID:          id,
			Kind:        gatewayapi.StringScalarParameterKindSTRING,
			Position:    value.Position,
			Name:        gatewayapi.NonBlankString(value.Name),
			Label:       gatewayapi.NonBlankString(value.Label),
			Description: description,
			Required:    value.Required,
			ReadOnly:    value.ReadOnly,
			Default:     toAPINilString(v.Default),
			Enum:        enum,
			Pattern:     toAPINilString(v.Pattern),
		}), nil
	case value.GetBoolean() != nil:
		v := value.GetBoolean()
		return gatewayapi.NewBooleanScalarParameterScalarParameter(gatewayapi.BooleanScalarParameter{
			ID:          id,
			Kind:        gatewayapi.BooleanScalarParameterKindBOOLEAN,
			Position:    value.Position,
			Name:        gatewayapi.NonBlankString(value.Name),
			Label:       gatewayapi.NonBlankString(value.Label),
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
			ID:          id,
			Kind:        gatewayapi.IntegerScalarParameterKindINTEGER,
			Position:    value.Position,
			Name:        gatewayapi.NonBlankString(value.Name),
			Label:       gatewayapi.NonBlankString(value.Label),
			Description: description,
			Required:    value.Required,
			ReadOnly:    value.ReadOnly,
			Minimum:     toAPINilInt64(v.Minimum),
			Maximum:     toAPINilInt64(v.Maximum),
			Default:     toAPINilInt64(v.Default),
			Enum:        enum,
		}), nil
	case value.GetFloat() != nil:
		v := value.GetFloat()
		enum, err := toAPIFloatEnumOptions(v.Enum)
		if err != nil {
			return gatewayapi.ScalarParameter{}, err
		}
		return gatewayapi.NewFloatScalarParameterScalarParameter(gatewayapi.FloatScalarParameter{
			ID:          id,
			Kind:        gatewayapi.FloatScalarParameterKindFLOAT,
			Position:    value.Position,
			Name:        gatewayapi.NonBlankString(value.Name),
			Label:       gatewayapi.NonBlankString(value.Label),
			Description: description,
			Required:    value.Required,
			ReadOnly:    value.ReadOnly,
			Minimum:     toAPINilFloat64(v.Minimum),
			Maximum:     toAPINilFloat64(v.Maximum),
			Default:     toAPINilFloat64(v.Default),
			Enum:        enum,
		}), nil
	case value.GetDateTime() != nil:
		v := value.GetDateTime()
		return gatewayapi.NewDateTimeScalarParameterScalarParameter(gatewayapi.DateTimeScalarParameter{
			ID:          id,
			Kind:        gatewayapi.DateTimeScalarParameterKindDATETIME,
			Position:    value.Position,
			Name:        gatewayapi.NonBlankString(value.Name),
			Label:       gatewayapi.NonBlankString(value.Label),
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

func toAPITemporalCoverage(value *metadatalifecyclev1.Temporal) gatewayapi.OptTemporalCoverage {
	if value == nil {
		return gatewayapi.OptTemporalCoverage{}
	}
	return gatewayapi.NewOptTemporalCoverage(gatewayapi.TemporalCoverage{
		Start: toAPINilTime(value.Start),
		End:   toAPINilTime(value.End),
	})
}

func toAPINilTime(value *timestamppb.Timestamp) gatewayapi.OptNilDateTime {
	result := gatewayapi.OptNilDateTime{}
	if value != nil {
		result.SetTo(value.AsTime())
	}
	return result
}

func toAPINilString(value *string) gatewayapi.OptNilString {
	result := gatewayapi.OptNilString{}
	if value != nil {
		result.SetTo(*value)
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
	default:
		return gatewayapi.OptNilParameterRole{}, fmt.Errorf("unsupported upstream parameter role %q", value)
	}
	return result, nil
}
