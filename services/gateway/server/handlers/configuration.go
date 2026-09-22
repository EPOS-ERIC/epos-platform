package handlers

import (
	"context"
	"fmt"

	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

// Configuration endpoints are pure proxies with gateway-owned authentication:
// writes require a bearer token, reads accept anonymous callers (public
// configurations only). The handler forwards the actor identity and admin
// flag; visibility and ownership policy is enforced by the sharing service.

// caller resolves the optional actor: empty actor means anonymous.
func (h *ogenHandler) caller(ctx context.Context) (actor string, admin bool) {
	actor, ok := actorID(ctx)
	if !ok {
		return "", false
	}
	return actor, isAdmin(ctx)
}

func (h *ogenHandler) CreateConfiguration(ctx context.Context, request *gatewayapi.CreateConfigurationRequest) (*gatewayapi.Configuration, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	protoRequest, err := toProtoCreateConfigurationRequest(request)
	if err != nil {
		return nil, fmt.Errorf("map create configuration request: %w", err)
	}
	response, err := h.sharing.CreateConfiguration(ctx, actor, isAdmin(ctx), protoRequest)
	if err != nil {
		return nil, fmt.Errorf("create configuration: %w", err)
	}
	mapped, err := toAPIConfiguration(response.GetConfiguration())
	if err != nil {
		return nil, err
	}
	return &mapped, nil
}

func (h *ogenHandler) GetConfiguration(ctx context.Context, params gatewayapi.GetConfigurationParams) (*gatewayapi.Configuration, error) {
	actor, admin := h.caller(ctx)
	response, err := h.sharing.GetConfiguration(ctx, actor, admin, &configurationv1.GetConfigurationRequest{Id: params.ConfigurationId.String()})
	if err != nil {
		return nil, fmt.Errorf("get configuration: %w", err)
	}
	mapped, err := toAPIConfiguration(response.GetConfiguration())
	if err != nil {
		return nil, err
	}
	return &mapped, nil
}

func (h *ogenHandler) ListConfigurations(ctx context.Context, params gatewayapi.ListConfigurationsParams) (*gatewayapi.ConfigurationList, error) {
	actor, admin := h.caller(ctx)
	response, err := h.sharing.ListConfigurations(ctx, actor, admin, toProtoListConfigurationsRequest(params))
	if err != nil {
		return nil, fmt.Errorf("list configurations: %w", err)
	}
	return toAPIConfigurationList(response)
}

func (h *ogenHandler) ReplaceConfiguration(ctx context.Context, request *gatewayapi.ReplaceConfigurationRequest, params gatewayapi.ReplaceConfigurationParams) (*gatewayapi.Configuration, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	protoRequest, err := toProtoReplaceConfigurationRequest(params.ConfigurationId.String(), request)
	if err != nil {
		return nil, fmt.Errorf("map replace configuration request: %w", err)
	}
	response, err := h.sharing.UpdateConfiguration(ctx, actor, isAdmin(ctx), protoRequest)
	if err != nil {
		return nil, fmt.Errorf("replace configuration: %w", err)
	}
	mapped, err := toAPIConfiguration(response.GetConfiguration())
	if err != nil {
		return nil, err
	}
	return &mapped, nil
}

func (h *ogenHandler) DeleteConfiguration(ctx context.Context, params gatewayapi.DeleteConfigurationParams) error {
	actor, ok := actorID(ctx)
	if !ok {
		return authMissing()
	}
	if _, err := h.sharing.DeleteConfiguration(ctx, actor, isAdmin(ctx), &configurationv1.DeleteConfigurationRequest{Id: params.ConfigurationId.String()}); err != nil {
		return fmt.Errorf("delete configuration: %w", err)
	}
	return nil
}
