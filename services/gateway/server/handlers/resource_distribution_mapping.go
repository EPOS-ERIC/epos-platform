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
		return gatewayapi.NewWebServiceDistributionDistribution(gatewayapi.WebServiceDistribution{
			ID:          id,
			Title:       value.Title,
			Description: value.Description,
			License:     toAPINilString(value.License),
			Type:        gatewayapi.WebServiceDistributionType(value.Type),
			Issued:      toAPINilTime(value.Issued),
			Modified:    toAPINilTime(value.Modified),
			ByteSize:    toAPINilInt64(value.ByteSize),
			WebService:  webService,
		}), nil
	default:
		return gatewayapi.Distribution{}, fmt.Errorf("unsupported upstream distribution type")
	}
}

func toAPIWebService(value *metadatalifecyclev1.WebService) (gatewayapi.WebService, error) {
	id, err := parseUUID("web_service.id", value.Id)
	if err != nil {
		return gatewayapi.WebService{}, err
	}
	result := gatewayapi.WebService{
		ID:               id,
		Name:             value.Name,
		Description:      value.Description,
		TemplateURL:      value.TemplateUrl,
		Format:           value.Format,
		SpatialCoverage:  toAPINilString(value.SpatialCoverage),
		TemporalCoverage: toAPITemporalCoverage(value.TemporalCoverage),
		ContactPoints:    make([]uuid.UUID, 0, len(value.ContactPoints)),
	}
	for _, parameter := range value.Parameters {
		if parameter == nil {
			continue
		}
		mapped, err := toAPIParameter(parameter)
		if err != nil {
			return gatewayapi.WebService{}, err
		}
		result.Parameters = append(result.Parameters, mapped)
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

type apiParameterMetadata interface {
	GetId() string
	GetVariable() string
	GetProperty() string
	GetLabel() string
	GetRequired() bool
	GetMultipleValues() bool
	GetReadOnly() bool
}

func toAPIParameterMetadata(value apiParameterMetadata) (gatewayapi.ParameterMetadata, error) {
	id, err := parseUUID("parameter.metadata.id", value.GetId())
	if err != nil {
		return gatewayapi.ParameterMetadata{}, err
	}
	return gatewayapi.ParameterMetadata{
		ID:             id,
		Variable:       value.GetVariable(),
		Property:       value.GetProperty(),
		Label:          value.GetLabel(),
		Required:       value.GetRequired(),
		MultipleValues: value.GetMultipleValues(),
		ReadOnly:       value.GetReadOnly(),
	}, nil
}

func toAPIParameter(value *metadatalifecyclev1.Parameter) (gatewayapi.Parameter, error) {
	switch {
	case value.GetString_() != nil:
		v := value.GetString_()
		metadata, err := toAPIParameterMetadata(value.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewStringParameterParameter(gatewayapi.StringParameter{
			Kind:          gatewayapi.StringParameterKindSTRING,
			Metadata:      metadata,
			Default:       toAPINilStringValue(v.Default),
			AllowedValues: v.AllowedValues,
			Pattern:       v.Pattern,
		}), nil
	case value.GetBoolean() != nil:
		v := value.GetBoolean()
		metadata, err := toAPIParameterMetadata(value.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewBooleanParameterParameter(gatewayapi.BooleanParameter{
			Kind:     gatewayapi.BooleanParameterKindBOOLEAN,
			Metadata: metadata,
			Default:  toAPINilBoolValue(v.Default),
		}), nil
	case value.GetInteger() != nil:
		v := value.GetInteger()
		metadata, err := toAPIParameterMetadata(value.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewIntegerParameterParameter(gatewayapi.IntegerParameter{
			Kind:          gatewayapi.IntegerParameterKindINTEGER,
			Metadata:      metadata,
			Minimum:       toAPINilInt64Value(v.Minimum),
			Maximum:       toAPINilInt64Value(v.Maximum),
			Default:       toAPINilInt64Value(v.Default),
			AllowedValues: v.AllowedValues,
		}), nil
	case value.GetFloat() != nil:
		v := value.GetFloat()
		metadata, err := toAPIParameterMetadata(value.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewFloatParameterParameter(gatewayapi.FloatParameter{
			Kind:          gatewayapi.FloatParameterKindFLOAT,
			Metadata:      metadata,
			Minimum:       toAPINilFloat64Value(v.Minimum),
			Maximum:       toAPINilFloat64Value(v.Maximum),
			Default:       toAPINilFloat64Value(v.Default),
			AllowedValues: v.AllowedValues,
		}), nil
	case value.GetDateTime() != nil:
		v := value.GetDateTime()
		metadata, err := toAPIParameterMetadata(value.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewDateTimeParameterParameter(gatewayapi.DateTimeParameter{
			Kind:     gatewayapi.DateTimeParameterKindDATETIME,
			Metadata: metadata,
			Minimum:  toAPINilTimeValue(v.Minimum),
			Maximum:  toAPINilTimeValue(v.Maximum),
			Default:  toAPINilTimeValue(v.Default),
			Pattern:  v.Pattern,
		}), nil
	default:
		return gatewayapi.Parameter{}, fmt.Errorf("unsupported upstream parameter type")
	}
}

func toAPINilStringValue(value *string) gatewayapi.NilString {
	result := gatewayapi.NilString{}
	if value != nil {
		result.SetTo(*value)
	} else {
		result.SetToNull()
	}
	return result
}

func toAPINilBoolValue(value *bool) gatewayapi.NilBool {
	result := gatewayapi.NilBool{}
	if value != nil {
		result.SetTo(*value)
	} else {
		result.SetToNull()
	}
	return result
}

func toAPINilInt64Value(value *int64) gatewayapi.NilInt64 {
	result := gatewayapi.NilInt64{}
	if value != nil {
		result.SetTo(*value)
	} else {
		result.SetToNull()
	}
	return result
}

func toAPINilFloat64Value(value *float64) gatewayapi.NilFloat64 {
	result := gatewayapi.NilFloat64{}
	if value != nil {
		result.SetTo(*value)
	} else {
		result.SetToNull()
	}
	return result
}

func toAPINilTimeValue(value *timestamppb.Timestamp) gatewayapi.NilDateTime {
	result := gatewayapi.NilDateTime{}
	if value != nil {
		result.SetTo(value.AsTime())
	} else {
		result.SetToNull()
	}
	return result
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
