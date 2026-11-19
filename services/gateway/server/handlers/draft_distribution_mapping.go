package handlers

import (
	"errors"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
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
	r := &metadatalifecyclev1.WebServiceInput{}
	switch v.Type {
	case gatewayapi.HttpWebServiceDraftWebServiceDraft:
		service := v.HttpWebServiceDraft
		mapped := toProtoHTTPWebServiceDraft(service)
		r.Service = &metadatalifecyclev1.WebServiceInput_Http{Http: mapped}
	case gatewayapi.DynamicWebServiceDraftWebServiceDraft:
		service := v.DynamicWebServiceDraft
		generic := toProtoHTTPWebServiceDraft(gatewayapi.HttpWebServiceDraft{
			ResourceID:       service.ResourceID,
			VisibleOn:        service.VisibleOn,
			Provider:         service.Provider,
			DocumentationURI: service.DocumentationURI,
			ContactPoints:    service.ContactPoints,
			Keywords:         service.Keywords,
			AccessRights:     service.AccessRights,
			License:          service.License,
			Categories:       service.Categories,
			TemplateURL:      service.TemplateURL,
			Method:           service.Method,
			Format:           service.Format,
			Parameters:       service.Parameters,
			BodyContentType:  service.BodyContentType,
			BodyTemplate:     service.BodyTemplate,
			BodyRequired:     service.BodyRequired,
			BodyParameters:   service.BodyParameters,
		})
		r.Service = &metadatalifecyclev1.WebServiceInput_Dynamic{Dynamic: &metadatalifecyclev1.DynamicWebServiceInput{
			Common:      generic.Common,
			TemplateUrl: generic.TemplateUrl,
			Method:      generic.Method,
			Format:      generic.Format,
			Parameters:  generic.Parameters,
			Body:        generic.Body,
		}}
	case gatewayapi.WfsWebServiceDraftWebServiceDraft:
		service := v.WfsWebServiceDraft
		mapped := toProtoWFSWebServiceConfigurationDraft(service)
		mapped.Common = toProtoWebServiceCommonInput(service.ResourceID, service.VisibleOn, service.Provider, service.DocumentationURI, service.ContactPoints, service.Keywords, service.AccessRights, service.License, service.Categories)
		r.Service = &metadatalifecyclev1.WebServiceInput_Wfs{Wfs: mapped}
	case gatewayapi.WmsWebServiceDraftWebServiceDraft:
		service := v.WmsWebServiceDraft
		mapped := toProtoWMSWebServiceConfigurationDraft(service)
		mapped.Common = toProtoWebServiceCommonInput(service.ResourceID, service.VisibleOn, service.Provider, service.DocumentationURI, service.ContactPoints, service.Keywords, service.AccessRights, service.License, service.Categories)
		r.Service = &metadatalifecyclev1.WebServiceInput_Wms{Wms: mapped}
	case gatewayapi.WmtsWebServiceDraftWebServiceDraft:
		service := v.WmtsWebServiceDraft
		mapped := toProtoWMTSWebServiceConfigurationDraft(service)
		mapped.Common = toProtoWebServiceCommonInput(service.ResourceID, service.VisibleOn, service.Provider, service.DocumentationURI, service.ContactPoints, service.Keywords, service.AccessRights, service.License, service.Categories)
		r.Service = &metadatalifecyclev1.WebServiceInput_Wmts{Wmts: mapped}
	default:
		return r
	}
	return r
}

func toProtoWebServiceCommonInput(resourceID gatewayapi.OptUUID, visibleOn gatewayapi.OptNilWebServiceVisibleOnArray, provider gatewayapi.OptUUID, documentation gatewayapi.OptURI, contacts []uuid.UUID, keywords []gatewayapi.Keyword, accessRights, license gatewayapi.OptNilString, categories []uuid.UUID) *metadatalifecyclev1.WebServiceCommonInput {
	result := &metadatalifecyclev1.WebServiceCommonInput{
		ResourceId:    protoOptionalUUID(resourceID),
		ProviderId:    protoOptionalUUID(provider),
		VisibleOn:     toProtoWebServiceVisibleOn(visibleOn),
		ContactPoints: uuidStrings(contacts),
		Keywords:      toProtoKeywords(keywords),
		AccessRights:  protoOptionalString(accessRights),
		License:       protoOptionalString(license),
		Categories:    uuidStrings(categories),
	}
	if documentation.Set {
		result.DocumentationUri = new(documentation.Value.String())
	}
	return result
}

func toProtoHTTPWebServiceDraft(value gatewayapi.HttpWebServiceDraft) *metadatalifecyclev1.HTTPWebServiceInput {
	result := &metadatalifecyclev1.HTTPWebServiceInput{
		Common:      toProtoWebServiceCommonInput(value.ResourceID, value.VisibleOn, value.Provider, value.DocumentationURI, value.ContactPoints, value.Keywords, value.AccessRights, value.License, value.Categories),
		TemplateUrl: string(value.TemplateURL),
		Method:      toProtoWebServiceMethod(value.Method),
		Format:      string(value.Format),
	}
	for _, parameter := range value.Parameters {
		result.Parameters = append(result.Parameters, toProtoParameterDraft(parameter))
	}
	if value.BodyRequired.Set || value.BodyContentType.Set || value.BodyTemplate.Set || len(value.BodyParameters) > 0 {
		result.Body = &metadatalifecyclev1.RequestBodyInput{
			ContentType: string(value.BodyContentType.Value),
			Template:    string(value.BodyTemplate.Value),
		}
		if value.BodyRequired.Set {
			result.Body.Required = new(value.BodyRequired.Value)
		}
		for _, parameter := range value.BodyParameters {
			result.Body.Parameters = append(result.Body.Parameters, toProtoScalarParameterDraft(parameter))
		}
	}
	return result
}

func toProtoWFSWebServiceConfigurationDraft(value gatewayapi.WfsWebServiceDraft) *metadatalifecyclev1.WFSWebServiceInput {
	result := &metadatalifecyclev1.WFSWebServiceInput{
		GetFeatureTemplateUrl: value.GetFeatureTemplateURL,
		Version:               string(value.Version),
		DefaultFeatureType:    string(value.DefaultFeatureType),
		DefaultOutputFormat:   string(value.DefaultOutputFormat),
		ResultLimit:           protoOptionalInt64(value.ResultLimit),
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

func toProtoWMTSWebServiceConfigurationDraft(value gatewayapi.WmtsWebServiceDraft) *metadatalifecyclev1.WMTSWebServiceInput {
	result := &metadatalifecyclev1.WMTSWebServiceInput{CapabilitiesUrl: value.CapabilitiesURL, DefaultLayer: string(value.DefaultLayer)}
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

func toProtoWMSWebServiceConfigurationDraft(value gatewayapi.WmsWebServiceDraft) *metadatalifecyclev1.WMSWebServiceInput {
	result := &metadatalifecyclev1.WMSWebServiceInput{
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
		return x.Title, x.Description.Value, protoOptionalString(x.License), protoOptionalInt64(x.ByteSize)
	}
	x := v.DownloadableFileDistributionDraft
	return x.Title, x.Description.Value, protoOptionalString(x.License), protoOptionalInt64(x.ByteSize)
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
