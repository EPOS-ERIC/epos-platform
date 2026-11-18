package handlers

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
)

func toAPIDistributionConversionPlugins(relations []domain.DistributionPluginRelation) []gatewayapi.DistributionConversionPlugin {
	result := make([]gatewayapi.DistributionConversionPlugin, 0)
	indices := make(map[uuid.UUID]int)
	for _, item := range relations {
		index, ok := indices[item.PluginID]
		if !ok {
			index = len(result)
			indices[item.PluginID] = index
			arguments := item.Arguments
			if arguments == nil {
				arguments = []string{}
			}

			result = append(result, gatewayapi.DistributionConversionPlugin{
				ID:          item.PluginID,
				Name:        item.Name,
				Description: item.Description,
				Repository:  item.Repository,
				Branch:      item.Branch,
				Runtime:     gatewayapi.ConversionRuntime(item.Runtime),
				Arguments:   arguments,
				Enabled:     item.Enabled,
				Installed:   item.Installed,
				Relations:   []gatewayapi.DistributionConversionRelation{},
			})
		}

		result[index].Relations = append(result[index].Relations, gatewayapi.DistributionConversionRelation{
			ID:             item.RelationID,
			DistributionID: item.DistributionID,
			InputFormat:    item.InputFormat,
			OutputFormat:   item.OutputFormat,
		})
	}

	return result
}

func toDomainConversionRuntime(runtime gatewayapi.ConversionRuntime) (domain.ConversionRuntime, error) {
	switch runtime {
	case gatewayapi.ConversionRuntimePython:
		return domain.ConversionRuntimePython, nil
	case gatewayapi.ConversionRuntimeJava:
		return domain.ConversionRuntimeJava, nil
	case gatewayapi.ConversionRuntimeBinary:
		return domain.ConversionRuntimeBinary, nil
	default:
		return "", fmt.Errorf("unknown conversion runtime %q", runtime)
	}
}

// withAvailableConversions appends conversions to their matching distributions.
func withAvailableConversions(dataset gatewayapi.DatasetResourceMetadata, byDistribution map[uuid.UUID][]domain.AvailableConversion) gatewayapi.DatasetResourceMetadata {
	for i := range dataset.Distributions {
		distribution := &dataset.Distributions[i]
		if !distribution.IsWebServiceDistribution() {
			continue
		}

		web := &distribution.WebServiceDistribution
		for _, conversion := range byDistribution[web.ID] {
			format := gatewayapi.NewConvertedFormatAvailableFormat(gatewayapi.ConvertedFormat{
				Kind:         gatewayapi.ConvertedFormatKindCONVERTED,
				ConversionID: conversion.ID,
				InputFormat:  conversion.InputFormat,
				OutputFormat: conversion.OutputFormat,
				Plugin: gatewayapi.ConversionPlugin{
					ID:          conversion.PluginID,
					Name:        conversion.PluginName,
					Description: conversion.PluginDescription,
				},
			})
			switch web.WebService.Type {
			case gatewayapi.HttpWebServiceDatasetWebService:
				web.WebService.HttpWebService.AvailableFormats = append(web.WebService.HttpWebService.AvailableFormats, format)
			case gatewayapi.DynamicWebServiceDatasetWebService:
				web.WebService.DynamicWebService.AvailableFormats = append(web.WebService.DynamicWebService.AvailableFormats, format)
			case gatewayapi.WfsWebServiceDatasetWebService:
				web.WebService.WfsWebService.AvailableFormats = append(web.WebService.WfsWebService.AvailableFormats, format)
			case gatewayapi.WmsWebServiceDatasetWebService, gatewayapi.WmtsWebServiceDatasetWebService:
				continue
			default:
				continue
			}
		}

	}
	return dataset
}

func toDomainConversionRelation(input *gatewayapi.ConversionRelationInput) domain.Relation {
	return domain.Relation{
		DistributionID: input.DistributionID,
		PluginID:       input.PluginID,
		InputFormat:    input.InputFormat,
		OutputFormat:   input.OutputFormat,
	}
}

func toAPIManagedConversionPlugin(value domain.Plugin) (gatewayapi.ManagedConversionPlugin, error) {
	var runtime gatewayapi.ConversionRuntime
	switch value.Runtime {
	case domain.ConversionRuntimePython:
		runtime = gatewayapi.ConversionRuntimePython
	case domain.ConversionRuntimeJava:
		runtime = gatewayapi.ConversionRuntimeJava
	case domain.ConversionRuntimeBinary:
		runtime = gatewayapi.ConversionRuntimeBinary
	default:
		return gatewayapi.ManagedConversionPlugin{}, fmt.Errorf("converter returned unknown runtime")
	}

	arguments := value.Arguments
	if arguments == nil {
		arguments = []string{}
	}

	return gatewayapi.ManagedConversionPlugin{
		ID:              value.ID,
		Name:            value.Name,
		Description:     value.Description,
		Repository:      value.Repository,
		Branch:          value.Branch,
		Runtime:         runtime,
		Arguments:       arguments,
		Enabled:         value.Enabled,
		Installed:       value.Installed,
		InstalledCommit: toAPIRequiredNilString(value.InstalledCommit),
		CreatedBy:       value.CreatedBy,
		UpdatedBy:       value.UpdatedBy,
	}, nil
}

func toAPIConversionRelation(value domain.Relation) gatewayapi.ConversionRelation {
	return gatewayapi.ConversionRelation{
		ID:             value.ID,
		DistributionID: value.DistributionID,
		PluginID:       value.PluginID,
		InputFormat:    value.InputFormat,
		OutputFormat:   value.OutputFormat,
		CreatedBy:      value.CreatedBy,
		UpdatedBy:      value.UpdatedBy,
	}
}
