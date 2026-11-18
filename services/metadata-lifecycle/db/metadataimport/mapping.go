package metadataimport

import (
	"fmt"
	"reflect"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

func identifierInputs(values []domain.Identifier) []domain.IdentifierInput {
	result := make([]domain.IdentifierInput, len(values))
	for i, value := range values {
		result[i] = domain.IdentifierInput{ResourceID: &value.ResourceID, Type: value.Type, Value: value.Value}
	}
	return result
}

func organizationIDs(values []domain.OrganizationMetadata) []uuid.UUID {
	result := make([]uuid.UUID, len(values))
	for i, value := range values {
		result[i] = value.ID
	}
	return result
}

func contactPointIDs(values []domain.ContactPoint) []uuid.UUID {
	result := make([]uuid.UUID, len(values))
	for i, value := range values {
		result[i] = value.ID
	}
	return result
}

func categoryIDs(values []domain.CategoryMetadata) []uuid.UUID {
	result := make([]uuid.UUID, len(values))
	for i, value := range values {
		result[i] = value.ID
	}
	return result
}

func toApplicationInput(value domain.SoftwareApplicationMetadata) domain.SoftwareApplicationInput {
	return domain.SoftwareApplicationInput{
		Name:             value.Name,
		Description:      value.Description,
		MainEntityOfPage: value.MainEntityOfPage,
		License:          value.License,
		Version:          value.Version,
		Keywords:         value.Keywords,
		DownloadURL:      value.DownloadURL,
		Requirements:     value.Requirements,
		OperatingSystem:  value.OperatingSystem,
		CreatorIDs:       organizationIDs(value.Creators),
		ContactPointIDs:  contactPointIDs(value.ContactPoints),
		CategoryIDs:      categoryIDs(value.Categories),
		Identifiers:      identifierInputs(value.Identifiers),
	}
}

func toSourceCodeInput(value domain.SoftwareSourceCodeMetadata) domain.SoftwareSourceCodeInput {
	return domain.SoftwareSourceCodeInput{
		Name:                 value.Name,
		Description:          value.Description,
		ProgrammingLanguage:  value.ProgrammingLanguage,
		Keywords:             value.Keywords,
		Repository:           value.Repository,
		License:              value.License,
		SoftwareRequirements: value.SoftwareRequirements,
		RuntimePlatform:      value.RuntimePlatform,
		MainEntityOfPage:     value.MainEntityOfPage,
		Version:              value.Version,
		CreatorIDs:           organizationIDs(value.Creators),
		ContactPointIDs:      contactPointIDs(value.ContactPoints),
		CategoryIDs:          categoryIDs(value.Categories),
		Identifiers:          identifierInputs(value.Identifiers),
	}
}

func toDatasetInput(value domain.DatasetMetadata) (domain.DatasetInput, error) {
	distributions := make([]domain.DistributionInput, len(value.Distributions))
	for i, distribution := range value.Distributions {
		input := domain.DistributionInput{
			ResourceID:       &distribution.ResourceID,
			Title:            distribution.Title,
			Description:      distribution.Description,
			License:          distribution.License,
			Type:             distribution.Type,
			ByteSize:         distribution.ByteSize,
			DownloadableFile: distribution.DownloadableFile,
		}
		if distribution.WebService != nil {
			service, err := toWebServiceInput(distribution.WebService)
			if err != nil {
				return domain.DatasetInput{}, fmt.Errorf("map distribution %d: %w", i, err)
			}
			input.WebService = service
		}
		distributions[i] = input
	}
	return domain.DatasetInput{
		Title:                      value.Title,
		Description:                value.Description,
		Keywords:                   value.Keywords,
		AccrualPeriodicity:         value.AccrualPeriodicity,
		QualityAnnotation:          value.QualityAnnotation,
		AccessRight:                value.AccessRight,
		Distributions:              distributions,
		SpatialCoverage:            value.SpatialCoverage,
		TemporalCoverage:           value.TemporalCoverage,
		GeologicalTemporalCoverage: value.GeologicalTemporalCoverage,
		PersistentIdentifiers:      identifierInputs(value.PersistentIdentifiers),
		ContactPointIDs:            contactPointIDs(value.ContactPoints),
		ProviderIDs:                organizationIDs(value.Providers),
		CategoryIDs:                categoryIDs(value.Categories),
		SourceCreated:              value.SourceCreated,
		SourceUpdated:              value.SourceUpdated,
	}, nil
}

func parameterInputs(values []domain.WebServiceParameter) []domain.WebServiceParameterInput {
	result := make([]domain.WebServiceParameterInput, len(values))
	for i, value := range values {
		result[i] = domain.WebServiceParameterInput{
			ParameterInput: value.Input(),
			Location:       value.Location,
			Explode:        value.Explode,
		}
	}
	return result
}

func genericInput(value domain.GenericWebService, common domain.WebServiceCommonInput) domain.GenericWebServiceInput {
	input := domain.GenericWebServiceInput{
		WebServiceCommonInput: common,
		TemplateURL:           value.TemplateURL,
		Method:                value.Method,
		Format:                value.Format,
		Parameters:            parameterInputs(value.Parameters),
	}
	if value.Body != nil {
		parameters := make([]domain.ParameterInput, len(value.Body.Parameters))
		for i, parameter := range value.Body.Parameters {
			parameters[i] = parameter.Input()
		}
		input.Body = &domain.RequestBodyInput{
			Required:    new(value.Body.Required),
			ContentType: value.Body.ContentType,
			Template:    value.Body.Template,
			Parameters:  parameters,
		}
	}
	return input
}

func toWebServiceInput(value domain.WebService) (domain.WebServiceInput, error) {
	commonValue := webServiceCommon(value)
	if commonValue == nil {
		return nil, apperror.Default(apperror.CodeInternal)
	}
	var providerID *uuid.UUID
	if commonValue.Provider != nil {
		providerID = &commonValue.Provider.ID
	}
	common := domain.WebServiceCommonInput{
		ResourceID:       &commonValue.ResourceID,
		VisibleOn:        commonValue.VisibleOn,
		ProviderID:       providerID,
		DocumentationURI: commonValue.DocumentationURI,
		ContactPointIDs:  contactPointIDs(commonValue.ContactPoints),
	}
	switch service := value.(type) {
	case *domain.HTTPWebService:
		return &domain.HTTPWebServiceInput{GenericWebServiceInput: genericInput(service.GenericWebService, common)}, nil
	case *domain.DynamicWebService:
		return &domain.DynamicWebServiceInput{GenericWebServiceInput: genericInput(service.GenericWebService, common)}, nil
	case *domain.WFSWebService:
		return &domain.WFSWebServiceInput{
			WebServiceCommonInput: common,
			GetFeatureTemplateURL: service.GetFeatureTemplateURL,
			Version:               service.Version,
			FeatureTypes:          service.FeatureTypes,
			DefaultFeatureType:    service.DefaultFeatureType,
			OutputFormats:         service.OutputFormats,
			DefaultOutputFormat:   service.DefaultOutputFormat,
			ResultLimit:           service.ResultLimit,
			ResultLimitEditable:   service.ResultLimitEditable,
			Parameters:            parameterInputs(service.Parameters),
		}, nil
	case *domain.WMSWebService:
		layers := make([]domain.WMSLayerInput, len(service.Layers))
		for i, layer := range service.Layers {
			layers[i] = domain.WMSLayerInput{ResourceID: &layer.ResourceID, Name: layer.Name, DefaultStyle: layer.DefaultStyle, Styles: layer.Styles}
		}
		return &domain.WMSWebServiceInput{
			WebServiceCommonInput:    common,
			CapabilitiesURL:          service.CapabilitiesURL,
			DefaultLayer:             service.DefaultLayer,
			Layers:                   layers,
			TransparencyEditable:     service.TransparencyEditable,
			DefaultTransparency:      service.DefaultTransparency,
			GetMapParameters:         parameterInputs(service.GetMapParameters),
			GetFeatureInfoParameters: parameterInputs(service.GetFeatureInfoParameters),
		}, nil
	case *domain.WMTSWebService:
		layers := make([]domain.WMTSLayerInput, len(service.Layers))
		for i, layer := range service.Layers {
			layers[i] = domain.WMTSLayerInput{ResourceID: &layer.ResourceID, Name: layer.Name, DefaultStyle: layer.DefaultStyle, Styles: layer.Styles}
		}
		return &domain.WMTSWebServiceInput{
			WebServiceCommonInput:    common,
			CapabilitiesURL:          service.CapabilitiesURL,
			DefaultLayer:             service.DefaultLayer,
			Layers:                   layers,
			GetTileParameters:        parameterInputs(service.GetTileParameters),
			GetFeatureInfoParameters: parameterInputs(service.GetFeatureInfoParameters),
		}, nil
	default:
		return nil, apperror.Wrap(apperror.CodeInternal, "", fmt.Errorf("unsupported web service %T", value))
	}
}

func webServiceCommon(value domain.WebService) *domain.WebServiceCommon {
	if value == nil {
		return nil
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil
	}
	return value.Common()
}
