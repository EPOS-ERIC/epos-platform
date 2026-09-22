package handlers

import (
	"fmt"

	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/go-faster/jx"
)

func toProtoCreateConfigurationRequest(request *gatewayapi.CreateConfigurationRequest) (*configurationv1.CreateConfigurationRequest, error) {
	if request == nil {
		return nil, fmt.Errorf("request is required")
	}
	result := &configurationv1.CreateConfigurationRequest{
		Configuration:  []byte(request.Configuration),
		ListOfServices: append([]string(nil), request.ListOfServices...),
		IsOfficial:     request.IsOfficial.Value,
	}
	if request.ID.Set && !request.ID.Null {
		id := request.ID.Value.String()
		result.Id = &id
	}
	if title := toClientStringPtr(request.Title); title != nil {
		result.Title = title
	}
	if description := toClientStringPtr(request.Description); description != nil {
		result.Description = description
	}
	if request.IsPublic.Set && !request.IsPublic.Null {
		isPublic := request.IsPublic.Value
		result.IsPublic = &isPublic
	}
	return result, nil
}

func toProtoReplaceConfigurationRequest(id string, request *gatewayapi.ReplaceConfigurationRequest) (*configurationv1.UpdateConfigurationRequest, error) {
	if request == nil {
		return nil, fmt.Errorf("request is required")
	}
	result := &configurationv1.UpdateConfigurationRequest{
		Id:             id,
		Configuration:  []byte(request.Configuration),
		ListOfServices: append([]string(nil), request.ListOfServices...),
		IsOfficial:     request.IsOfficial.Value,
	}
	if title := toClientStringPtr(request.Title); title != nil {
		result.Title = title
	}
	if description := toClientStringPtr(request.Description); description != nil {
		result.Description = description
	}
	if request.IsPublic.Set && !request.IsPublic.Null {
		isPublic := request.IsPublic.Value
		result.IsPublic = &isPublic
	}
	return result, nil
}

func toProtoListConfigurationsRequest(params gatewayapi.ListConfigurationsParams) *configurationv1.ListConfigurationsRequest {
	result := &configurationv1.ListConfigurationsRequest{}
	if params.PageSize.Set {
		pageSize := int32(params.PageSize.Value)
		result.PageSize = &pageSize
	}
	if params.Offset.Set {
		offset := int32(params.Offset.Value)
		result.Offset = &offset
	}
	if params.Official.Set {
		official := params.Official.Value
		result.Official = &official
	}
	return result
}

func toAPIConfiguration(record *configurationv1.Configuration) (gatewayapi.Configuration, error) {
	if record == nil {
		return gatewayapi.Configuration{}, fmt.Errorf("decode sharing configuration: response is empty")
	}
	id, err := parseUUID("configuration.id", record.GetId())
	if err != nil {
		return gatewayapi.Configuration{}, err
	}
	return gatewayapi.Configuration{
		ID:             id,
		Configuration:  jx.Raw(record.GetConfiguration()),
		Title:          optionalString(record.Title),
		Description:    optionalString(record.Description),
		ListOfServices: append([]string(nil), record.GetListOfServices()...),
		IsOfficial:     record.GetIsOfficial(),
		IsPublic:       record.GetIsPublic(),
		OwnerID:        record.GetOwnerId(),
		CreatedAt:      record.GetCreatedAt().AsTime(),
		UpdatedAt:      record.GetUpdatedAt().AsTime(),
	}, nil
}

func toAPIConfigurationList(response *configurationv1.ListConfigurationsResponse) (*gatewayapi.ConfigurationList, error) {
	if response == nil {
		return nil, fmt.Errorf("decode sharing configuration list: response is empty")
	}
	result := &gatewayapi.ConfigurationList{Total: response.GetTotal()}
	result.Configurations = make([]gatewayapi.Configuration, 0, len(response.GetConfigurations()))
	for _, record := range response.GetConfigurations() {
		mapped, err := toAPIConfiguration(record)
		if err != nil {
			return nil, err
		}
		result.Configurations = append(result.Configurations, mapped)
	}
	return result, nil
}
