package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/go-faster/jx"
)

func toProtoCreateConfigurationRequest(request *gatewayapi.ConfigurationWriteRequest) (*configurationv1.CreateConfigurationRequest, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	return &configurationv1.CreateConfigurationRequest{
		Configuration:  []byte(request.Configuration),
		Title:          request.Title,
		Description:    toClientStringPtr(request.Description),
		ListOfServices: append([]string{}, request.ListOfServices...),
		IsPublic:       request.IsPublic.Or(false),
	}, nil
}

func toProtoReplaceConfigurationRequest(id string, request *gatewayapi.ConfigurationWriteRequest) (*configurationv1.UpdateConfigurationRequest, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	return &configurationv1.UpdateConfigurationRequest{
		Id:             id,
		Configuration:  []byte(request.Configuration),
		Title:          request.Title,
		Description:    toClientStringPtr(request.Description),
		ListOfServices: append([]string{}, request.ListOfServices...),
		IsPublic:       protoOptionalBool(request.IsPublic),
	}, nil
}

func toProtoSetConfigurationOfficialStatusRequest(id string, request *gatewayapi.SetConfigurationOfficialStatusRequest) (*configurationv1.SetConfigurationOfficialStatusRequest, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	isOfficial := request.IsOfficial
	return &configurationv1.SetConfigurationOfficialStatusRequest{Id: id, IsOfficial: &isOfficial}, nil
}

func toAPIConfiguration(record *configurationv1.Configuration, includeOwner bool) (gatewayapi.ConfigurationDetail, error) {
	if record == nil {
		return gatewayapi.ConfigurationDetail{}, fmt.Errorf("decode sharing configuration: response is empty")
	}
	if !json.Valid(record.GetConfiguration()) {
		return gatewayapi.ConfigurationDetail{}, fmt.Errorf("decode sharing configuration: configuration is invalid JSON")
	}
	id, err := parseUUID("configuration.id", record.GetId())
	if err != nil {
		return gatewayapi.ConfigurationDetail{}, err
	}
	createdAt, err := fromProtoTimestamp("configuration.created_at", record.GetCreatedAt())
	if err != nil {
		return gatewayapi.ConfigurationDetail{}, err
	}
	updatedAt, err := fromProtoTimestamp("configuration.updated_at", record.GetUpdatedAt())
	if err != nil {
		return gatewayapi.ConfigurationDetail{}, err
	}
	ownerID, err := toAPIOwnerID(record.GetOwnerId(), includeOwner)
	if err != nil {
		return gatewayapi.ConfigurationDetail{}, err
	}
	return gatewayapi.ConfigurationDetail{
		ID:             id,
		Configuration:  jx.Raw(record.GetConfiguration()),
		Title:          record.GetTitle(),
		Description:    toAPIRequiredNilString(record.Description),
		ListOfServices: append([]string{}, record.GetListOfServices()...),
		IsOfficial:     record.GetIsOfficial(),
		IsPublic:       record.GetIsPublic(),
		OwnerID:        ownerID,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

func toAPIConfigurationSummary(record *configurationv1.ConfigurationSummary, includeOwner bool) (gatewayapi.ConfigurationSummary, error) {
	if record == nil {
		return gatewayapi.ConfigurationSummary{}, fmt.Errorf("decode sharing configuration summary: response is empty")
	}
	id, err := parseUUID("configuration.id", record.GetId())
	if err != nil {
		return gatewayapi.ConfigurationSummary{}, err
	}
	createdAt, err := fromProtoTimestamp("configuration.created_at", record.GetCreatedAt())
	if err != nil {
		return gatewayapi.ConfigurationSummary{}, err
	}
	updatedAt, err := fromProtoTimestamp("configuration.updated_at", record.GetUpdatedAt())
	if err != nil {
		return gatewayapi.ConfigurationSummary{}, err
	}
	ownerID, err := toAPIOwnerID(record.GetOwnerId(), includeOwner)
	if err != nil {
		return gatewayapi.ConfigurationSummary{}, err
	}
	return gatewayapi.ConfigurationSummary{
		ID:             id,
		Title:          record.GetTitle(),
		Description:    toAPIRequiredNilString(record.Description),
		ListOfServices: append([]string{}, record.GetListOfServices()...),
		IsOfficial:     record.GetIsOfficial(),
		IsPublic:       record.GetIsPublic(),
		OwnerID:        ownerID,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

func toAPIConfigurationSummaries(response *configurationv1.ListConfigurationsResponse, includeOwner bool) ([]gatewayapi.ConfigurationSummary, error) {
	if response == nil {
		return nil, fmt.Errorf("decode sharing configuration list: response is empty")
	}
	result := make([]gatewayapi.ConfigurationSummary, 0, len(response.GetConfigurations()))
	for _, record := range response.GetConfigurations() {
		mapped, err := toAPIConfigurationSummary(record, includeOwner)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func toAPIOfficialConfigurationSummary(record *configurationv1.ConfigurationSummary) (gatewayapi.OfficialConfigurationSummary, error) {
	summary, err := toAPIConfigurationSummary(record, false)
	if err != nil {
		return gatewayapi.OfficialConfigurationSummary{}, err
	}
	return gatewayapi.OfficialConfigurationSummary{
		ID:             summary.ID,
		Title:          summary.Title,
		Description:    summary.Description,
		ListOfServices: summary.ListOfServices,
		IsOfficial:     summary.IsOfficial,
		IsPublic:       summary.IsPublic,
		CreatedAt:      summary.CreatedAt,
		UpdatedAt:      summary.UpdatedAt,
	}, nil
}

func toAPIOfficialConfigurationSummaries(response *configurationv1.ListConfigurationsResponse) ([]gatewayapi.OfficialConfigurationSummary, error) {
	if response == nil {
		return nil, fmt.Errorf("decode official configuration list: response is empty")
	}
	result := make([]gatewayapi.OfficialConfigurationSummary, 0, len(response.GetConfigurations()))
	for _, record := range response.GetConfigurations() {
		mapped, err := toAPIOfficialConfigurationSummary(record)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

func toAPIOwnerID(value string, includeOwner bool) (gatewayapi.OptString, error) {
	if !includeOwner {
		return gatewayapi.OptString{}, nil
	}
	if value == "" {
		return gatewayapi.OptString{}, fmt.Errorf("decode sharing configuration: owner ID is empty")
	}
	return gatewayapi.NewOptString(value), nil
}
