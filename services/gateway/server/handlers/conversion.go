package handlers

import (
	"context"
	"slices"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	converterv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/converter/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func requireConverterManager(ctx context.Context) (string, error) {
	user, ok := authcontext.UserFromContext(ctx)
	if !ok || user.ID == "" {
		return "", apperror.Default(apperror.CodeUnauthenticated)
	}

	if !slices.Contains(user.GlobalGrantedPermissions, domain.GlobalPermissionConverterManage) &&
		!slices.Contains(user.GlobalRolePermissions, domain.GlobalPermissionConverterManage) {
		return "", apperror.Default(apperror.CodePermissionDenied)
	}

	return user.ID, nil
}

func (h *ogenHandler) ListDistributionConversionRelations(ctx context.Context, params gatewayapi.ListDistributionConversionRelationsParams) ([]gatewayapi.DistributionConversionPlugin, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok || actor == "" {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}

	relations, err := h.distributionAccess.DistributionRelations(ctx, actor, params.DistributionId)
	if err != nil {
		return nil, err
	}

	return toAPIDistributionConversionPlugins(relations), nil
}

func (h *ogenHandler) CreateConversionPlugin(ctx context.Context, request *gatewayapi.CreateConversionPluginRequest) (gatewayapi.CreateConversionPluginRes, error) {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return nil, err
	}

	runtime, err := toProtoConversionRuntime(request.Runtime)
	if err != nil {
		return nil, err
	}

	response, err := h.converter.CreatePlugin(ctx, actor, &converterv1.CreatePluginRequest{
		Input: &converterv1.PluginInput{
			Name:        request.Name,
			Description: request.Description.Value,
			Repository:  request.Repository,
			Branch:      request.Branch,
			Runtime:     runtime,
			Arguments:   request.Arguments,
		},
	})
	if err != nil {
		return nil, err
	}

	p, err := toAPIManagedConversionPlugin(response.GetPlugin())
	if err != nil {
		return nil, err
	}

	if response.GetSyncFailed() {
		result := gatewayapi.CreateConversionPluginAccepted(p)
		return &result, nil
	}

	return new(gatewayapi.CreateConversionPluginCreated(p)), nil
}

func (h *ogenHandler) GetConversionPlugin(ctx context.Context, params gatewayapi.GetConversionPluginParams) (*gatewayapi.ManagedConversionPlugin, error) {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return nil, err
	}

	response, err := h.converter.GetPlugin(ctx, actor, &converterv1.GetPluginRequest{Id: params.PluginId.String()})
	if err != nil {
		return nil, err
	}

	p, err := toAPIManagedConversionPlugin(response.GetPlugin())
	return &p, err
}

func (h *ogenHandler) ListConversionPlugins(ctx context.Context) ([]gatewayapi.ManagedConversionPlugin, error) {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return nil, err
	}

	response, err := h.converter.ListPlugins(ctx, actor)
	if err != nil {
		return nil, err
	}

	result := make([]gatewayapi.ManagedConversionPlugin, 0, len(response.GetPlugins()))
	for _, value := range response.GetPlugins() {
		p, err := toAPIManagedConversionPlugin(value)
		if err != nil {
			return nil, err
		}

		result = append(result, p)
	}

	return result, nil
}

func (h *ogenHandler) ReplaceConversionPlugin(ctx context.Context, request *gatewayapi.ReplaceConversionPluginRequest, params gatewayapi.ReplaceConversionPluginParams) (gatewayapi.ReplaceConversionPluginRes, error) {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return nil, err
	}

	runtime, err := toProtoConversionRuntime(request.Runtime)
	if err != nil {
		return nil, err
	}

	response, err := h.converter.UpdatePlugin(ctx, actor, &converterv1.UpdatePluginRequest{
		Id:      params.PluginId.String(),
		Enabled: request.Enabled,
		Input: &converterv1.PluginInput{
			Name:        request.Name,
			Description: request.Description.Value,
			Repository:  request.Repository,
			Branch:      request.Branch,
			Runtime:     runtime,
			Arguments:   request.Arguments,
		},
	})
	if err != nil {
		return nil, err
	}

	p, err := toAPIManagedConversionPlugin(response.GetPlugin())
	if err != nil {
		return nil, err
	}

	if response.GetSyncFailed() {
		result := gatewayapi.ReplaceConversionPluginAccepted(p)
		return &result, nil
	}

	return new(gatewayapi.ReplaceConversionPluginOK(p)), nil
}

func (h *ogenHandler) DeleteConversionPlugin(ctx context.Context, params gatewayapi.DeleteConversionPluginParams) error {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return err
	}

	return h.converter.DeletePlugin(ctx, actor, &converterv1.DeletePluginRequest{
		Id: params.PluginId.String(),
	})
}

func (h *ogenHandler) SyncConversionPlugin(ctx context.Context, params gatewayapi.SyncConversionPluginParams) (*gatewayapi.ManagedConversionPlugin, error) {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return nil, err
	}

	response, err := h.converter.SyncPlugin(ctx, actor, &converterv1.SyncPluginRequest{
		Id: params.PluginId.String(),
	})
	if err != nil {
		return nil, err
	}

	p, err := toAPIManagedConversionPlugin(response.GetPlugin())
	return &p, err
}

func (h *ogenHandler) validateConversionRelation(ctx context.Context, actor string, input *gatewayapi.ConversionRelationInput) error {
	formats, err := h.metadataLifecycle.DistributionFormats(ctx, actor, input.DistributionID)
	if err != nil {
		return err
	}

	if !slices.Contains(formats, input.InputFormat) {
		return apperror.Field("input_format", "Input format must match a native format of the distribution.", nil)
	}

	return nil
}

func (h *ogenHandler) CreateConversionRelation(ctx context.Context, request *gatewayapi.ConversionRelationInput) (*gatewayapi.ConversionRelation, error) {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.validateConversionRelation(ctx, actor, request); err != nil {
		return nil, err
	}

	response, err := h.converter.CreateRelation(ctx, actor, &converterv1.CreateRelationRequest{
		Input: toProtoConversionRelation(request),
	})
	if err != nil {
		return nil, err
	}

	relation, err := toAPIConversionRelation(response.GetRelation())
	return &relation, err
}

func (h *ogenHandler) GetConversionRelation(ctx context.Context, params gatewayapi.GetConversionRelationParams) (*gatewayapi.ConversionRelation, error) {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return nil, err
	}
	response, err := h.converter.GetRelation(ctx, actor, &converterv1.GetRelationRequest{Id: params.ConversionId.String()})
	if err != nil {
		return nil, err
	}
	relation, err := toAPIConversionRelation(response.GetRelation())
	return &relation, err
}

func (h *ogenHandler) ListConversionRelations(ctx context.Context) ([]gatewayapi.ConversionRelation, error) {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return nil, err
	}
	response, err := h.converter.ListRelations(ctx, actor)
	if err != nil {
		return nil, err
	}
	result := make([]gatewayapi.ConversionRelation, 0, len(response.GetRelations()))
	for _, value := range response.GetRelations() {
		relation, err := toAPIConversionRelation(value)
		if err != nil {
			return nil, err
		}
		result = append(result, relation)
	}
	return result, nil
}

func (h *ogenHandler) ReplaceConversionRelation(ctx context.Context, request *gatewayapi.ConversionRelationInput, params gatewayapi.ReplaceConversionRelationParams) (*gatewayapi.ConversionRelation, error) {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.validateConversionRelation(ctx, actor, request); err != nil {
		return nil, err
	}
	response, err := h.converter.UpdateRelation(ctx, actor, &converterv1.UpdateRelationRequest{Id: params.ConversionId.String(), Input: toProtoConversionRelation(request)})
	if err != nil {
		return nil, err
	}
	relation, err := toAPIConversionRelation(response.GetRelation())
	return &relation, err
}

func (h *ogenHandler) DeleteConversionRelation(ctx context.Context, params gatewayapi.DeleteConversionRelationParams) error {
	actor, err := requireConverterManager(ctx)
	if err != nil {
		return err
	}
	return h.converter.DeleteRelation(ctx, actor, &converterv1.DeleteRelationRequest{Id: params.ConversionId.String()})
}
