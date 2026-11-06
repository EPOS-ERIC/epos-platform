package handlers

import (
	"errors"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoTemporal(v gatewayapi.OptTemporalCoverage) *metadatalifecyclev1.Temporal {
	if !v.Set {
		return nil
	}
	r := &metadatalifecyclev1.Temporal{}
	if v.Value.Start.Set && !v.Value.Start.Null {
		r.Start = timestamppb.New(v.Value.Start.Value)
	}
	if v.Value.End.Set && !v.Value.End.Null {
		r.End = timestamppb.New(v.Value.End.Value)
	}
	return r
}

type protoDatasetTemporalRange struct {
	temporal   *metadatalifecyclev1.Temporal
	geological *metadatalifecyclev1.GeologicalTemporalCoverage
}

func toProtoDatasetTemporalRange(
	temporal gatewayapi.OptTemporalCoverage,
	geological gatewayapi.OptGeologicalTemporalCoverage,
) (protoDatasetTemporalRange, error) {
	if temporal.Set && geological.Set {
		return protoDatasetTemporalRange{}, errors.New("dataset temporal coverage fields are mutually exclusive")
	}
	if temporal.Set {
		return protoDatasetTemporalRange{temporal: toProtoTemporal(temporal)}, nil
	}
	if geological.Set {
		return protoDatasetTemporalRange{geological: toProtoGeologicalTemporal(geological.Value)}, nil
	}
	return protoDatasetTemporalRange{}, nil
}

func toProtoGeologicalTemporal(v gatewayapi.GeologicalTemporalCoverage) *metadatalifecyclev1.GeologicalTemporalCoverage {
	result := &metadatalifecyclev1.GeologicalTemporalCoverage{}
	if v.StartMya.Set && !v.StartMya.Null {
		result.StartMya = new(v.StartMya.Value)
	}
	if v.EndMya.Set && !v.EndMya.Null {
		result.EndMya = new(v.EndMya.Value)
	}
	return result
}

func toProtoWebServiceDraft(v gatewayapi.WebServiceDraft) *metadatalifecyclev1.WebServiceInput {
	r := &metadatalifecyclev1.WebServiceInput{
		ResourceId:    protoOptionalUUID(v.ResourceID),
		VisibleOn:     toProtoWebServiceVisibleOn(v.VisibleOn),
		ContactPoints: uuidStrings(v.ContactPoints),
	}
	if v.DocumentationURI.Set {
		r.DocumentationUri = new(v.DocumentationURI.Value.String())
	}
	if v.Provider.Set {
		r.ProviderId = new(v.Provider.Value.String())
	}
	switch v.Configuration.Type {
	case gatewayapi.WmsWebServiceConfigurationDraftWebServiceConfigurationDraft:
		configuration := v.Configuration.WmsWebServiceConfigurationDraft
		r.Type = metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WMS
		r.Configuration = &metadatalifecyclev1.WebServiceInput_Wms{Wms: toProtoWMSWebServiceConfigurationDraft(configuration)}
	case gatewayapi.WmtsWebServiceConfigurationDraftWebServiceConfigurationDraft:
		r.Type = metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WMTS
		r.Configuration = &metadatalifecyclev1.WebServiceInput_Wmts{Wmts: toProtoWMTSWebServiceConfigurationDraft(v.Configuration.WmtsWebServiceConfigurationDraft)}
	case gatewayapi.GenericWebServiceConfigurationDraftWebServiceConfigurationDraft:
		configuration := v.Configuration.GenericWebServiceConfigurationDraft
		r.Type = toProtoWebServiceType(gatewayapi.WebServiceType(v.Configuration.Type))
		r.Configuration = &metadatalifecyclev1.WebServiceInput_Generic{Generic: toProtoGenericWebServiceConfigurationDraft(configuration)}
	case gatewayapi.DynamicWebServiceConfigurationDraftWebServiceConfigurationDraft:
		configuration := v.Configuration.DynamicWebServiceConfigurationDraft
		r.Type = metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_DYNAMIC
		r.Configuration = &metadatalifecyclev1.WebServiceInput_Dynamic{Dynamic: toProtoGenericWebServiceConfigurationDraft(gatewayapi.GenericWebServiceConfigurationDraft{
			TemplateURL:     configuration.TemplateURL,
			Method:          configuration.Method,
			Parameters:      configuration.Parameters,
			BodyContentType: configuration.BodyContentType,
			BodyTemplate:    configuration.BodyTemplate,
			BodyRequired:    configuration.BodyRequired,
			BodyParameters:  configuration.BodyParameters,
			Format:          configuration.Format,
		})}
	case gatewayapi.WfsWebServiceConfigurationDraftWebServiceConfigurationDraft:
		r.Type = metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WFS
		r.Configuration = &metadatalifecyclev1.WebServiceInput_Wfs{Wfs: toProtoWFSWebServiceConfigurationDraft(v.Configuration.WfsWebServiceConfigurationDraft)}
	default:
		r.Type = metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_UNSPECIFIED
	}
	return r
}

func toProtoGenericWebServiceConfigurationDraft(value gatewayapi.GenericWebServiceConfigurationDraft) *metadatalifecyclev1.GenericWebServiceConfigurationInput {
	result := &metadatalifecyclev1.GenericWebServiceConfigurationInput{
		TemplateUrl: string(value.TemplateURL),
		Method:      toProtoWebServiceMethod(value.Method),
		Format:      string(value.Format),
	}
	for _, parameter := range value.Parameters {
		result.Parameters = append(result.Parameters, toProtoParameterDraft(parameter))
	}
	if value.BodyRequired.Set {
		result.BodyRequired = new(value.BodyRequired.Value)
	}
	if value.BodyContentType.Set {
		result.BodyContentType = new(string(value.BodyContentType.Value))
	}
	if value.BodyTemplate.Set {
		result.BodyTemplate = new(string(value.BodyTemplate.Value))
	}
	for _, parameter := range value.BodyParameters {
		result.BodyParameters = append(result.BodyParameters, toProtoScalarParameterDraft(parameter))
	}
	return result
}

func toProtoWFSWebServiceConfigurationDraft(value gatewayapi.WfsWebServiceConfigurationDraft) *metadatalifecyclev1.WFSWebServiceConfigurationInput {
	result := &metadatalifecyclev1.WFSWebServiceConfigurationInput{
		GetFeatureTemplateUrl: value.GetFeatureTemplateURL,
		Version:               string(value.Version),
		DefaultFeatureType:    string(value.DefaultFeatureType),
		DefaultOutputFormat:   string(value.DefaultOutputFormat),
		ResultLimit:           protoOptInt64(value.ResultLimit),
		ResultLimitEditable:   value.ResultLimitEditable,
	}
	for _, featureType := range value.FeatureTypes {
		result.FeatureTypes = append(result.FeatureTypes, string(featureType))
	}
	for _, format := range value.OutputFormats {
		result.OutputFormats = append(result.OutputFormats, string(format))
	}
	for _, parameter := range value.Parameters {
		result.Parameters = append(result.Parameters, toProtoParameterDraft(parameter))
	}
	return result
}

func toProtoWMTSWebServiceConfigurationDraft(value gatewayapi.WmtsWebServiceConfigurationDraft) *metadatalifecyclev1.WMTSWebServiceConfigurationInput {
	result := &metadatalifecyclev1.WMTSWebServiceConfigurationInput{CapabilitiesUrl: value.CapabilitiesURL, DefaultLayer: string(value.DefaultLayer)}
	for _, layer := range value.Layers {
		mapped := &metadatalifecyclev1.WMTSLayerInput{
			Name:       string(layer.Name),
			ResourceId: protoOptionalUUID(layer.ResourceID),
		}

		if layer.DefaultStyle.Set {
			mapped.DefaultStyle = new(string(layer.DefaultStyle.Value))
		}
		for _, style := range layer.Styles {
			mapped.Styles = append(mapped.Styles, string(style))
		}
		result.Layers = append(result.Layers, mapped)
	}
	for _, parameter := range value.GetTileParameters {
		result.GetTileParameters = append(result.GetTileParameters, toProtoParameterDraft(parameter))
	}
	for _, parameter := range value.GetFeatureInfoParameters {
		result.GetFeatureInfoParameters = append(result.GetFeatureInfoParameters, toProtoParameterDraft(parameter))
	}
	return result
}

func toProtoWMSWebServiceConfigurationDraft(value gatewayapi.WmsWebServiceConfigurationDraft) *metadatalifecyclev1.WMSWebServiceConfigurationInput {
	result := &metadatalifecyclev1.WMSWebServiceConfigurationInput{
		CapabilitiesUrl:      value.CapabilitiesURL,
		DefaultLayer:         string(value.DefaultLayer),
		TransparencyEditable: value.TransparencyEditable,
		DefaultTransparency:  value.DefaultTransparency,
	}
	for _, layer := range value.Layers {
		mapped := &metadatalifecyclev1.WMSLayerInput{
			ResourceId:   protoOptionalUUID(layer.ResourceID),
			Name:         string(layer.Name),
			DefaultStyle: layer.DefaultStyle,
			Styles:       layer.Styles,
		}

		result.Layers = append(result.Layers, mapped)
	}
	for _, parameter := range value.GetMapParameters {
		result.GetMapParameters = append(result.GetMapParameters, toProtoParameterDraft(parameter))
	}
	for _, parameter := range value.GetFeatureInfoParameters {
		result.GetFeatureInfoParameters = append(result.GetFeatureInfoParameters, toProtoParameterDraft(parameter))
	}
	return result
}

func toProtoDistribution(v gatewayapi.DistributionDraft) (*metadatalifecyclev1.DistributionInput, error) {
	title, description, license, byteSize := distributionFields(v)
	distributionType, err := toProtoDistributionType(v.Type)
	if err != nil {
		return nil, err
	}
	r := &metadatalifecyclev1.DistributionInput{
		Type:        distributionType,
		Title:       title,
		Description: description,
		License:     license,
		ByteSize:    byteSize,
	}
	if v.IsWebServiceDistributionDraft() {
		x := v.WebServiceDistributionDraft
		r.ResourceId = protoOptionalUUID(x.ResourceID)
		r.Delivery = &metadatalifecyclev1.DistributionInput_WebService{
			WebService: toProtoWebServiceDraft(x.WebService),
		}
		return r, nil
	}

	x := v.DownloadableFileDistributionDraft
	r.ResourceId = protoOptionalUUID(x.ResourceID)
	r.Delivery = &metadatalifecyclev1.DistributionInput_DownloadableFile{
		DownloadableFile: &metadatalifecyclev1.DownloadableFile{
			Url:    string(x.DownloadableFile.URL),
			Format: string(x.DownloadableFile.Format),
		},
	}
	return r, nil
}

func distributionFields(v gatewayapi.DistributionDraft) (string, string, *string, *int64) {
	if v.IsWebServiceDistributionDraft() {
		x := v.WebServiceDistributionDraft
		return x.Title, x.Description.Value, protoOptString(x.License), protoOptInt64(x.ByteSize)
	}
	x := v.DownloadableFileDistributionDraft
	return x.Title, x.Description.Value, protoOptString(x.License), protoOptInt64(x.ByteSize)
}

func protoOptString(v gatewayapi.OptNilString) *string {
	if !v.Set || v.Null {
		return nil
	}
	return new(v.Value)
}

func protoOptInt64(v gatewayapi.OptNilInt64) *int64 {
	if !v.Set || v.Null {
		return nil
	}
	return new(v.Value)
}

func toProtoWebServiceMethod(value gatewayapi.WebServiceMethod) metadatalifecyclev1.WebServiceMethod {
	switch value {
	case gatewayapi.WebServiceMethodGET:
		return metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_GET
	case gatewayapi.WebServiceMethodPOST:
		return metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_POST
	case gatewayapi.WebServiceMethodPUT:
		return metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_PUT
	case gatewayapi.WebServiceMethodDELETE:
		return metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_DELETE
	default:
		return metadatalifecyclev1.WebServiceMethod_WEB_SERVICE_METHOD_UNSPECIFIED
	}
}

func toProtoWebServiceType(value gatewayapi.WebServiceType) metadatalifecyclev1.WebServiceType {
	switch value {
	case gatewayapi.WebServiceTypeWMS:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WMS
	case gatewayapi.WebServiceTypeHTTP:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_HTTP
	case gatewayapi.WebServiceTypeWMTS:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WMTS
	case gatewayapi.WebServiceTypeWFS:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WFS
	case gatewayapi.WebServiceTypeDYNAMIC:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_DYNAMIC
	default:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_UNSPECIFIED
	}
}

func toProtoWebServiceVisibleOn(value gatewayapi.OptNilWebServiceVisibleOnArray) []metadatalifecyclev1.WebServiceVisibleOn {
	values, ok := value.Get()
	if !ok {
		return nil
	}
	result := make([]metadatalifecyclev1.WebServiceVisibleOn, len(values))
	for i, item := range values {
		switch item {
		case gatewayapi.WebServiceVisibleOnGRAPH:
			result[i] = metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_GRAPH
		case gatewayapi.WebServiceVisibleOnTABLE:
			result[i] = metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_TABLE
		case gatewayapi.WebServiceVisibleOnMAP:
			result[i] = metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_MAP
		default:
			result[i] = metadatalifecyclev1.WebServiceVisibleOn_WEB_SERVICE_VISIBLE_ON_UNSPECIFIED
		}
	}
	return result
}
