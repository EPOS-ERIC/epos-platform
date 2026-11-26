package clients

import (
	"fmt"

	converterv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/converter/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

func toProtoPluginInput(input domain.PluginInput) (*converterv1.PluginInput, error) {
	var runtime converterv1.Runtime
	switch input.Runtime {
	case domain.ConversionRuntimePython:
		runtime = converterv1.Runtime_RUNTIME_PYTHON
	case domain.ConversionRuntimeJava:
		runtime = converterv1.Runtime_RUNTIME_JAVA
	case domain.ConversionRuntimeBinary:
		runtime = converterv1.Runtime_RUNTIME_BINARY
	default:
		return nil, fmt.Errorf("unknown plugin runtime %q", input.Runtime)
	}
	return &converterv1.PluginInput{
		Name:        input.Name,
		Description: input.Description,
		Repository:  input.Repository,
		Branch:      input.Branch,
		Runtime:     runtime,
		Arguments:   input.Arguments,
	}, nil
}

func fromProtoPlugin(value *converterv1.Plugin) (domain.Plugin, error) {
	if value == nil || value.GetInput() == nil {
		return domain.Plugin{}, fmt.Errorf("converter returned incomplete plugin")
	}
	id, err := uuid.Parse(value.GetId())
	if err != nil {
		return domain.Plugin{}, fmt.Errorf("invalid plugin id: %w", err)
	}
	runtime, err := fromProtoRuntime(value.GetInput().GetRuntime())
	if err != nil {
		return domain.Plugin{}, err
	}
	input := value.GetInput()
	return domain.Plugin{
		Name:            input.GetName(),
		Description:     input.GetDescription(),
		Repository:      input.GetRepository(),
		Branch:          input.GetBranch(),
		Runtime:         runtime,
		Arguments:       input.GetArguments(),
		ID:              id,
		Enabled:         value.GetEnabled(),
		Installed:       value.GetInstalled(),
		InstalledCommit: value.InstalledCommit,
		CreatedBy:       value.GetCreatedBy(),
		UpdatedBy:       value.GetUpdatedBy(),
	}, nil
}

func fromProtoRuntime(value converterv1.Runtime) (domain.ConversionRuntime, error) {
	switch value {
	case converterv1.Runtime_RUNTIME_PYTHON:
		return domain.ConversionRuntimePython, nil
	case converterv1.Runtime_RUNTIME_JAVA:
		return domain.ConversionRuntimeJava, nil
	case converterv1.Runtime_RUNTIME_BINARY:
		return domain.ConversionRuntimeBinary, nil
	default:
		return "", fmt.Errorf("converter returned unknown runtime")
	}
}

func toProtoRelationInput(relation domain.Relation) *converterv1.RelationInput {
	return &converterv1.RelationInput{
		DistributionId: relation.DistributionID.String(),
		PluginId:       relation.PluginID.String(),
		InputFormat:    relation.InputFormat,
		OutputFormat:   relation.OutputFormat,
	}
}

func fromProtoRelation(value *converterv1.Relation) (domain.Relation, error) {
	if value == nil || value.GetInput() == nil {
		return domain.Relation{}, fmt.Errorf("converter returned incomplete relation")
	}
	id, err := uuid.Parse(value.GetId())
	if err != nil {
		return domain.Relation{}, fmt.Errorf("invalid relation id: %w", err)
	}
	distributionID, err := uuid.Parse(value.GetInput().GetDistributionId())
	if err != nil {
		return domain.Relation{}, fmt.Errorf("invalid relation distribution id: %w", err)
	}
	pluginID, err := uuid.Parse(value.GetInput().GetPluginId())
	if err != nil {
		return domain.Relation{}, fmt.Errorf("invalid relation plugin id: %w", err)
	}
	return domain.Relation{
		ID:             id,
		DistributionID: distributionID,
		PluginID:       pluginID,
		InputFormat:    value.GetInput().GetInputFormat(),
		OutputFormat:   value.GetInput().GetOutputFormat(),
		CreatedBy:      value.GetCreatedBy(),
		UpdatedBy:      value.GetUpdatedBy(),
	}, nil
}
