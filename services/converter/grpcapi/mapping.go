package grpcapi

import (
	"net/url"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	converterv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/converter/v1"
	"github.com/EPOS-ERIC/epos-platform/services/converter/domain"
	"github.com/EPOS-ERIC/epos-platform/services/converter/plugin"
	"github.com/google/uuid"
)

func parseID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperror.Field(field, "A valid UUID is required.", err)
	}
	return id, nil
}

func toDomainPluginInput(input *converterv1.PluginInput) (domain.PluginInput, error) {
	if input == nil {
		return domain.PluginInput{}, apperror.Field("input", "Plugin input is required.", nil)
	}
	repository, err := url.Parse(strings.TrimSpace(input.GetRepository()))
	if err != nil {
		return domain.PluginInput{}, apperror.Field("repository", "A valid Git repository URL is required.", nil)
	}
	var runtime domain.Runtime
	switch input.GetRuntime() {
	case converterv1.Runtime_RUNTIME_PYTHON:
		runtime = domain.RuntimePython
	case converterv1.Runtime_RUNTIME_JAVA:
		runtime = domain.RuntimeJava
	case converterv1.Runtime_RUNTIME_BINARY:
		runtime = domain.RuntimeBinary
	default:
		return domain.PluginInput{}, apperror.Field("runtime", "Runtime must be python, java, or binary.", nil)
	}
	return domain.PluginInput{
		Name: input.GetName(), Description: input.GetDescription(), Repository: *repository,
		Branch: input.GetBranch(), Runtime: runtime, Arguments: input.GetArguments(),
	}, nil
}

func toProtoPlugin(record domain.Plugin, status plugin.Status) (*converterv1.Plugin, error) {
	var runtime converterv1.Runtime
	switch record.Runtime {
	case domain.RuntimePython:
		runtime = converterv1.Runtime_RUNTIME_PYTHON
	case domain.RuntimeJava:
		runtime = converterv1.Runtime_RUNTIME_JAVA
	case domain.RuntimeBinary:
		runtime = converterv1.Runtime_RUNTIME_BINARY
	default:
		return nil, apperror.Default(apperror.CodeInternal)
	}
	result := &converterv1.Plugin{
		Id: record.ID.String(), Enabled: record.Enabled, Installed: status.Installed,
		CreatedBy: record.CreatedBy, UpdatedBy: record.UpdatedBy,
		Input: &converterv1.PluginInput{
			Name: record.Name, Description: record.Description, Repository: record.Repository.String(),
			Branch: record.Branch, Runtime: runtime, Arguments: record.Arguments,
		},
	}
	if status.Commit != "" {
		result.InstalledCommit = new(status.Commit)
	}
	return result, nil
}

func toDomainRelationInput(input *converterv1.RelationInput) (domain.RelationInput, error) {
	if input == nil {
		return domain.RelationInput{}, apperror.Field("input", "Relation input is required.", nil)
	}
	distributionID, err := parseID("distribution_id", input.GetDistributionId())
	if err != nil {
		return domain.RelationInput{}, err
	}
	pluginID, err := parseID("plugin_id", input.GetPluginId())
	if err != nil {
		return domain.RelationInput{}, err
	}
	return domain.RelationInput{
		DistributionID: distributionID, PluginID: pluginID,
		InputFormat: input.GetInputFormat(), OutputFormat: input.GetOutputFormat(),
	}, nil
}

func toProtoRelation(record domain.Relation) *converterv1.Relation {
	return &converterv1.Relation{
		Id: record.ID.String(), CreatedBy: record.CreatedBy, UpdatedBy: record.UpdatedBy,
		Input: &converterv1.RelationInput{
			DistributionId: record.DistributionID.String(), PluginId: record.PluginID.String(),
			InputFormat: record.InputFormat, OutputFormat: record.OutputFormat,
		},
	}
}
