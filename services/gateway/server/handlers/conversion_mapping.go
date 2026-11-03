package handlers

import (
	"fmt"

	converterv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/converter/v1"
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

func toProtoConversionRuntime(runtime gatewayapi.ConversionRuntime) (converterv1.Runtime, error) {
	switch runtime {
	case gatewayapi.ConversionRuntimePython:
		return converterv1.Runtime_RUNTIME_PYTHON, nil
	case gatewayapi.ConversionRuntimeJava:
		return converterv1.Runtime_RUNTIME_JAVA, nil
	case gatewayapi.ConversionRuntimeBinary:
		return converterv1.Runtime_RUNTIME_BINARY, nil
	default:
		return 0, fmt.Errorf("unknown conversion runtime %q", runtime)
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
			web.AvailableFormats = append(web.AvailableFormats, gatewayapi.NewConvertedFormatAvailableFormat(gatewayapi.ConvertedFormat{
				Kind:         gatewayapi.ConvertedFormatKindCONVERTED,
				ConversionID: conversion.ID,
				InputFormat:  conversion.InputFormat,
				OutputFormat: conversion.OutputFormat,
				Plugin: gatewayapi.ConversionPlugin{
					ID:          conversion.PluginID,
					Name:        conversion.PluginName,
					Description: conversion.PluginDescription,
				},
			}))
		}

	}
	return dataset
}

func toProtoConversionRelation(input *gatewayapi.ConversionRelationInput) *converterv1.RelationInput {
	return &converterv1.RelationInput{
		DistributionId: input.DistributionID.String(),
		PluginId:       input.PluginID.String(),
		InputFormat:    input.InputFormat,
		OutputFormat:   input.OutputFormat,
	}
}

func toAPIManagedConversionPlugin(value *converterv1.Plugin) (gatewayapi.ManagedConversionPlugin, error) {
	if value == nil || value.GetInput() == nil {
		return gatewayapi.ManagedConversionPlugin{}, fmt.Errorf("converter returned an incomplete plugin")
	}

	id, err := parseUUID("plugin.id", value.GetId())
	if err != nil {
		return gatewayapi.ManagedConversionPlugin{}, err
	}

	input := value.GetInput()
	var runtime gatewayapi.ConversionRuntime
	switch input.GetRuntime() {
	case converterv1.Runtime_RUNTIME_PYTHON:
		runtime = gatewayapi.ConversionRuntimePython
	case converterv1.Runtime_RUNTIME_JAVA:
		runtime = gatewayapi.ConversionRuntimeJava
	case converterv1.Runtime_RUNTIME_BINARY:
		runtime = gatewayapi.ConversionRuntimeBinary
	default:
		return gatewayapi.ManagedConversionPlugin{}, fmt.Errorf("converter returned unknown runtime")
	}

	arguments := input.GetArguments()
	if arguments == nil {
		arguments = []string{}
	}

	return gatewayapi.ManagedConversionPlugin{
		ID:              id,
		Name:            input.GetName(),
		Description:     input.GetDescription(),
		Repository:      input.GetRepository(),
		Branch:          input.GetBranch(),
		Runtime:         runtime,
		Arguments:       arguments,
		Enabled:         value.GetEnabled(),
		Installed:       value.GetInstalled(),
		InstalledCommit: toAPIRequiredNilString(value.InstalledCommit),
		CreatedBy:       value.GetCreatedBy(),
		UpdatedBy:       value.GetUpdatedBy(),
	}, nil
}

func toAPIConversionRelation(value *converterv1.Relation) (gatewayapi.ConversionRelation, error) {
	if value == nil || value.GetInput() == nil {
		return gatewayapi.ConversionRelation{}, fmt.Errorf("converter returned an incomplete relation")
	}

	id, err := parseUUID("relation.id", value.GetId())
	if err != nil {
		return gatewayapi.ConversionRelation{}, err
	}

	distributionID, err := parseUUID("relation.distribution_id", value.GetInput().GetDistributionId())
	if err != nil {
		return gatewayapi.ConversionRelation{}, err
	}

	pluginID, err := parseUUID("relation.plugin_id", value.GetInput().GetPluginId())
	if err != nil {
		return gatewayapi.ConversionRelation{}, err
	}

	return gatewayapi.ConversionRelation{
		ID:             id,
		DistributionID: distributionID,
		PluginID:       pluginID,
		InputFormat:    value.GetInput().GetInputFormat(),
		OutputFormat:   value.GetInput().GetOutputFormat(),
		CreatedBy:      value.GetCreatedBy(),
		UpdatedBy:      value.GetUpdatedBy(),
	}, nil
}
