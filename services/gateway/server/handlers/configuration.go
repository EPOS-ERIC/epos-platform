package handlers

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) CreateConfiguration(ctx context.Context, request *gatewayapi.ConfigurationWriteRequest) (*gatewayapi.ConfigurationDetail, error) {
	user, authenticated := authcontext.UserFromContext(ctx)
	protoRequest, err := toProtoCreateConfigurationRequest(request)
	if err != nil {
		return nil, fmt.Errorf("map create configuration request: %w", err)
	}
	response, err := h.sharing.CreateConfiguration(ctx, user, authenticated, protoRequest)
	if err != nil {
		return nil, fmt.Errorf("create configuration: %w", err)
	}
	mapped, err := toAPIConfiguration(response.GetConfiguration(), h.sharing.CanManageConfigurations(user, authenticated))
	if err != nil {
		return nil, err
	}
	return &mapped, nil
}

func (h *ogenHandler) GetConfiguration(ctx context.Context, params gatewayapi.GetConfigurationParams) (*gatewayapi.ConfigurationDetail, error) {
	user, authenticated := authcontext.UserFromContext(ctx)
	response, err := h.sharing.GetConfiguration(ctx, user, authenticated, &configurationv1.GetConfigurationRequest{Id: params.ConfigurationId.String()})
	if err != nil {
		return nil, fmt.Errorf("get configuration: %w", err)
	}
	mapped, err := toAPIConfiguration(response.GetConfiguration(), h.sharing.CanManageConfigurations(user, authenticated))
	if err != nil {
		return nil, err
	}
	return &mapped, nil
}

func (h *ogenHandler) ListConfigurations(ctx context.Context, params gatewayapi.ListConfigurationsParams) ([]gatewayapi.ConfigurationSummary, error) {
	user, authenticated := authcontext.UserFromContext(ctx)
	request, err := toProtoListConfigurationsRequest(params)
	if err != nil {
		return nil, err
	}
	response, err := h.sharing.ListConfigurations(ctx, user, authenticated, request)
	if err != nil {
		return nil, fmt.Errorf("list configurations: %w", err)
	}
	return toAPIConfigurationSummaries(response, h.sharing.CanManageConfigurations(user, authenticated))
}

func (h *ogenHandler) ReplaceConfiguration(ctx context.Context, request *gatewayapi.ConfigurationWriteRequest, params gatewayapi.ReplaceConfigurationParams) (*gatewayapi.ConfigurationDetail, error) {
	user, authenticated := authcontext.UserFromContext(ctx)
	protoRequest, err := toProtoReplaceConfigurationRequest(params.ConfigurationId.String(), request)
	if err != nil {
		return nil, fmt.Errorf("map replace configuration request: %w", err)
	}
	response, err := h.sharing.UpdateConfiguration(ctx, user, authenticated, protoRequest)
	if err != nil {
		return nil, fmt.Errorf("replace configuration: %w", err)
	}
	mapped, err := toAPIConfiguration(response.GetConfiguration(), h.sharing.CanManageConfigurations(user, authenticated))
	if err != nil {
		return nil, err
	}
	return &mapped, nil
}

func (h *ogenHandler) DeleteConfiguration(ctx context.Context, params gatewayapi.DeleteConfigurationParams) error {
	user, authenticated := authcontext.UserFromContext(ctx)
	_, err := h.sharing.DeleteConfiguration(ctx, user, authenticated, &configurationv1.DeleteConfigurationRequest{Id: params.ConfigurationId.String()})
	if err != nil {
		return fmt.Errorf("delete configuration: %w", err)
	}
	return nil
}

func (h *ogenHandler) ListOfficialConfigurations(ctx context.Context) ([]gatewayapi.OfficialConfigurationSummary, error) {
	user, authenticated := authcontext.UserFromContext(ctx)
	response, err := h.sharing.ListConfigurations(ctx, user, authenticated, &configurationv1.ListConfigurationsRequest{
		Scope: configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OFFICIAL,
	})
	if err != nil {
		return nil, fmt.Errorf("list official configurations: %w", err)
	}
	return toAPIOfficialConfigurationSummaries(response)
}

func (h *ogenHandler) SetConfigurationOfficialStatus(ctx context.Context, request *gatewayapi.SetConfigurationOfficialStatusRequest, params gatewayapi.SetConfigurationOfficialStatusParams) (*gatewayapi.ConfigurationDetail, error) {
	user, authenticated := authcontext.UserFromContext(ctx)
	protoRequest, err := toProtoSetConfigurationOfficialStatusRequest(params.ConfigurationId.String(), request)
	if err != nil {
		return nil, fmt.Errorf("map configuration status request: %w", err)
	}
	response, err := h.sharing.SetConfigurationOfficialStatus(ctx, user, authenticated, protoRequest)
	if err != nil {
		return nil, fmt.Errorf("set configuration official status: %w", err)
	}
	mapped, err := toAPIConfiguration(response.GetConfiguration(), h.sharing.CanManageConfigurations(user, authenticated))
	if err != nil {
		return nil, err
	}
	return &mapped, nil
}

func toProtoListConfigurationsRequest(params gatewayapi.ListConfigurationsParams) (*configurationv1.ListConfigurationsRequest, error) {
	ownerID, hasOwnerID := params.OwnerID.Get()
	scope, hasScope := params.Scope.Get()
	if hasOwnerID && hasScope {
		return nil, apperror.Field("owner_id", "Specify owner_id or scope=all, not both.", nil)
	}
	request := &configurationv1.ListConfigurationsRequest{}
	switch {
	case hasOwnerID:
		request.Scope = configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWNER
		request.OwnerId = &ownerID
	case hasScope && scope == gatewayapi.ListConfigurationsScopeAll:
		request.Scope = configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_ALL
	default:
		request.Scope = configurationv1.ConfigurationListScope_CONFIGURATION_LIST_SCOPE_OWN
	}
	return request, nil
}
