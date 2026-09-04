package handlers

import (
	"fmt"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
	"github.com/google/uuid"
)

func toAPIDistribution(value metadatalifecycleclient.Distribution) (gatewayapi.Distribution, error) {
	switch {
	case value.DownloadableFileDistribution != nil:
		v := value.DownloadableFileDistribution
		id, err := parseUUID("downloadable_file_distribution.id", v.Id)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		return gatewayapi.NewDownloadableFileDistributionDistribution(gatewayapi.DownloadableFileDistribution{
			ID:          id,
			Title:       v.Title,
			Description: v.Description,
			License:     toAPINilString(v.License),
			Type:        gatewayapi.DownloadableFileDistributionType(v.Type),
			Issued:      toAPINilTime(v.Issued),
			Modified:    toAPINilTime(v.Modified),
			ByteSize:    toAPINilInt64(v.ByteSize),
			DownloadableFile: gatewayapi.DownloadableFile{
				URL:    gatewayapi.NonBlankString(v.DownloadableFile.Url),
				Format: gatewayapi.NonBlankString(v.DownloadableFile.Format),
			},
		}), nil
	case value.WebServiceDistribution != nil:
		v := value.WebServiceDistribution
		id, err := parseUUID("web_service_distribution.id", v.Id)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		webService, err := toAPIWebService(v.WebService)
		if err != nil {
			return gatewayapi.Distribution{}, err
		}
		return gatewayapi.NewWebServiceDistributionDistribution(gatewayapi.WebServiceDistribution{
			ID:          id,
			Title:       v.Title,
			Description: v.Description,
			License:     toAPINilString(v.License),
			Type:        gatewayapi.WebServiceDistributionType(v.Type),
			Issued:      toAPINilTime(v.Issued),
			Modified:    toAPINilTime(v.Modified),
			ByteSize:    toAPINilInt64(v.ByteSize),
			WebService:  webService,
		}), nil
	default:
		return gatewayapi.Distribution{}, fmt.Errorf("unsupported upstream distribution type")
	}
}

func toAPIWebService(value metadatalifecycleclient.WebService) (gatewayapi.WebService, error) {
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
		mapped, err := toAPIParameter(parameter)
		if err != nil {
			return gatewayapi.WebService{}, err
		}
		result.Parameters = append(result.Parameters, mapped)
	}
	if value.Provider != nil {
		providerID, err := parseUUID("web_service.provider_id", value.Provider.Id)
		if err != nil {
			return gatewayapi.WebService{}, err
		}
		result.Provider.SetTo(gatewayapi.ResourceVersionReference{
			ID:   providerID,
			Name: value.Provider.Name,
		})
	}
	if value.Documentation != nil {
		result.Documentation = gatewayapi.NewOptDocumentation(gatewayapi.Documentation{
			URI:         gatewayapi.NonBlankString(value.Documentation.Uri),
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

func toAPIParameter(value metadatalifecycleclient.Parameter) (gatewayapi.Parameter, error) {
	switch {
	case value.StringParameter != nil:
		v := value.StringParameter
		metadata, err := toAPIParameterMetadata(&v.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewStringParameterParameter(gatewayapi.StringParameter{
			Kind:          gatewayapi.StringParameterKind(v.Kind),
			Metadata:      metadata,
			Default:       toAPINilStringValue(v.Default),
			AllowedValues: v.AllowedValues,
			Pattern:       v.Pattern,
		}), nil
	case value.BooleanParameter != nil:
		v := value.BooleanParameter
		metadata, err := toAPIParameterMetadata(&v.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewBooleanParameterParameter(gatewayapi.BooleanParameter{
			Kind:     gatewayapi.BooleanParameterKind(v.Kind),
			Metadata: metadata,
			Default:  toAPINilBoolValue(v.Default),
		}), nil
	case value.IntegerParameter != nil:
		v := value.IntegerParameter
		metadata, err := toAPIParameterMetadata(&v.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewIntegerParameterParameter(gatewayapi.IntegerParameter{
			Kind:          gatewayapi.IntegerParameterKind(v.Kind),
			Metadata:      metadata,
			Minimum:       toAPINilInt64Value(v.Minimum),
			Maximum:       toAPINilInt64Value(v.Maximum),
			Default:       toAPINilInt64Value(v.Default),
			AllowedValues: v.AllowedValues,
		}), nil
	case value.FloatParameter != nil:
		v := value.FloatParameter
		metadata, err := toAPIParameterMetadata(&v.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewFloatParameterParameter(gatewayapi.FloatParameter{
			Kind:          gatewayapi.FloatParameterKind(v.Kind),
			Metadata:      metadata,
			Minimum:       toAPINilFloat64Value(v.Minimum),
			Maximum:       toAPINilFloat64Value(v.Maximum),
			Default:       toAPINilFloat64Value(v.Default),
			AllowedValues: v.AllowedValues,
		}), nil
	case value.DateTimeParameter != nil:
		v := value.DateTimeParameter
		metadata, err := toAPIParameterMetadata(&v.Metadata)
		if err != nil {
			return gatewayapi.Parameter{}, err
		}
		return gatewayapi.NewDateTimeParameterParameter(gatewayapi.DateTimeParameter{
			Kind:     gatewayapi.DateTimeParameterKind(v.Kind),
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

func toAPINilStringValue(value metadatalifecycleclient.NullableString) gatewayapi.NilString {
	result := gatewayapi.NilString{}
	if value := value.Get(); value != nil {
		result.SetTo(*value)
	} else {
		result.SetToNull()
	}
	return result
}

func toAPINilBoolValue(value metadatalifecycleclient.NullableBool) gatewayapi.NilBool {
	result := gatewayapi.NilBool{}
	if value := value.Get(); value != nil {
		result.SetTo(*value)
	} else {
		result.SetToNull()
	}
	return result
}

func toAPINilInt64Value(value metadatalifecycleclient.NullableInt64) gatewayapi.NilInt64 {
	result := gatewayapi.NilInt64{}
	if value := value.Get(); value != nil {
		result.SetTo(*value)
	} else {
		result.SetToNull()
	}
	return result
}

func toAPINilFloat64Value(value metadatalifecycleclient.NullableFloat64) gatewayapi.NilFloat64 {
	result := gatewayapi.NilFloat64{}
	if value := value.Get(); value != nil {
		result.SetTo(*value)
	} else {
		result.SetToNull()
	}
	return result
}

func toAPINilTimeValue(value metadatalifecycleclient.NullableTime) gatewayapi.NilDateTime {
	result := gatewayapi.NilDateTime{}
	if value := value.Get(); value != nil {
		result.SetTo(*value)
	} else {
		result.SetToNull()
	}
	return result
}

func toAPITemporalCoverage(value *metadatalifecycleclient.Temporal) gatewayapi.OptTemporalCoverage {
	if value == nil {
		return gatewayapi.OptTemporalCoverage{}
	}
	return gatewayapi.NewOptTemporalCoverage(gatewayapi.TemporalCoverage{
		Start: toAPINilTime(value.Start),
		End:   toAPINilTime(value.End),
	})
}

func toAPINilTime(value metadatalifecycleclient.NullableTime) gatewayapi.OptNilDateTime {
	result := gatewayapi.OptNilDateTime{}
	if value := value.Get(); value != nil {
		result.SetTo(*value)
	}
	return result
}

func toAPINilString(value metadatalifecycleclient.NullableString) gatewayapi.OptNilString {
	result := gatewayapi.OptNilString{}
	if value := value.Get(); value != nil {
		result.SetTo(*value)
	}
	return result
}

func toAPINilInt64(value metadatalifecycleclient.NullableInt64) gatewayapi.OptNilInt64 {
	result := gatewayapi.OptNilInt64{}
	if value := value.Get(); value != nil {
		result.SetTo(*value)
	}
	return result
}
