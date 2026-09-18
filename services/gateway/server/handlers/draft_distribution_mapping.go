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

func toProtoDocumentation(v gatewayapi.OptDocumentation) *metadatalifecyclev1.Documentation {
	if !v.Set {
		return nil
	}
	r := &metadatalifecyclev1.Documentation{Uri: string(v.Value.URI)}
	if v.Value.Title.Set && !v.Value.Title.Null {
		r.Title = new(v.Value.Title.Value)
	}
	if v.Value.Description.Set && !v.Value.Description.Null {
		r.Description = new(v.Value.Description.Value)
	}
	return r
}

func toProtoWebServiceDraft(v gatewayapi.WebServiceDraft) *metadatalifecyclev1.WebServiceInput {
	r := &metadatalifecyclev1.WebServiceInput{
		Name:             string(v.Name),
		Description:      protoOptString(v.Description),
		TemplateUrl:      string(v.TemplateURL),
		Method:           toProtoWebServiceMethod(v.Method),
		Type:             toProtoWebServiceType(v.Type),
		VisibleOn:        toProtoWebServiceVisibleOn(v.VisibleOn),
		Format:           string(v.Format),
		TemporalCoverage: toProtoTemporal(v.TemporalCoverage),
		Documentation:    toProtoDocumentation(v.Documentation),
		ContactPoints:    uuidStrings(v.ContactPoints),
	}
	if v.SpatialCoverage.Set && !v.SpatialCoverage.Null {
		r.SpatialCoverage = new(v.SpatialCoverage.Value)
	}
	if v.Provider.Set {
		r.ProviderId = new(v.Provider.Value.String())
	}
	for _, parameter := range v.Parameters {
		r.Parameters = append(r.Parameters, toProtoParameterDraft(parameter))
	}
	if v.BodyContentType.Set || v.BodyTemplate.Set || len(v.BodyParameters) > 0 {
		if v.BodyContentType.Set {
			r.BodyContentType = new(string(v.BodyContentType.Value))
		}
		if v.BodyTemplate.Set {
			r.BodyTemplate = new(string(v.BodyTemplate.Value))
		}
		for _, parameter := range v.BodyParameters {
			r.BodyParameters = append(r.BodyParameters, toProtoScalarParameterDraft(parameter))
		}
	}
	return r
}

func toProtoDistributionDraft(v gatewayapi.DistributionDraft) *metadatalifecyclev1.DistributionInput {
	title, description, license, issued, modified, byteSize := distributionFields(v)
	r := &metadatalifecyclev1.DistributionInput{
		Type:        string(v.Type),
		Title:       title,
		Description: description,
		License:     license,
		Issued:      issued,
		Modified:    modified,
		ByteSize:    byteSize,
	}
	if v.IsWebServiceDistributionDraft() {
		x := v.WebServiceDistributionDraft
		r.Delivery = &metadatalifecyclev1.DistributionInput_WebService{WebService: toProtoWebServiceDraft(x.WebService)}
		return r
	}
	x := v.DownloadableFileDistributionDraft
	r.Delivery = &metadatalifecyclev1.DistributionInput_DownloadableFile{
		DownloadableFile: &metadatalifecyclev1.DownloadableFile{
			Url:    string(x.DownloadableFile.URL),
			Format: string(x.DownloadableFile.Format),
		},
	}
	return r
}

func toProtoDistribution(v gatewayapi.DistributionDraft) *metadatalifecyclev1.DistributionInput {
	title, description, license, issued, modified, byteSize := distributionFields(v)
	r := &metadatalifecyclev1.DistributionInput{
		Type:        string(v.Type),
		Title:       title,
		Description: description,
		License:     license,
		Issued:      issued,
		Modified:    modified,
		ByteSize:    byteSize,
	}
	if v.IsWebServiceDistributionDraft() {
		x := v.WebServiceDistributionDraft
		r.Delivery = &metadatalifecyclev1.DistributionInput_WebService{
			WebService: toProtoWebServiceDraft(x.WebService),
		}
		return r
	}

	x := v.DownloadableFileDistributionDraft
	r.Delivery = &metadatalifecyclev1.DistributionInput_DownloadableFile{
		DownloadableFile: &metadatalifecyclev1.DownloadableFile{
			Url:    string(x.DownloadableFile.URL),
			Format: string(x.DownloadableFile.Format),
		},
	}
	return r
}

func distributionFields(v gatewayapi.DistributionDraft) (string, string, *string, *timestamppb.Timestamp, *timestamppb.Timestamp, *int64) {
	if v.IsWebServiceDistributionDraft() {
		x := v.WebServiceDistributionDraft
		return x.Title, x.Description.Value, protoOptString(x.License), protoOptTime(x.Issued), protoOptTime(x.Modified), protoOptInt64(x.ByteSize)
	}
	x := v.DownloadableFileDistributionDraft
	return x.Title, x.Description.Value, protoOptString(x.License), protoOptTime(x.Issued), protoOptTime(x.Modified), protoOptInt64(x.ByteSize)
}

func protoOptString(v gatewayapi.OptNilString) *string {
	if !v.Set || v.Null {
		return nil
	}
	return new(v.Value)
}

func protoOptTime(v gatewayapi.OptNilDateTime) *timestamppb.Timestamp {
	if !v.Set || v.Null {
		return nil
	}
	return timestamppb.New(v.Value)
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
	case gatewayapi.WebServiceTypeGEOJSON:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_GEOJSON
	case gatewayapi.WebServiceTypeDYNAMIC:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_DYNAMIC
	case gatewayapi.WebServiceTypeWMTS:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WMTS
	case gatewayapi.WebServiceTypeWFS:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_WFS
	case gatewayapi.WebServiceTypeCOVJSON:
		return metadatalifecyclev1.WebServiceType_WEB_SERVICE_TYPE_COVJSON
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
