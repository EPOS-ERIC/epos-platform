package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

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

func (h *ogenHandler) CreateConversionPlugin(ctx context.Context, request *gatewayapi.CreateConversionPluginRequest) (*gatewayapi.ManagedConversionPlugin, error) {
	user, authenticated := authcontext.UserFromContext(ctx)

	runtime, err := toDomainConversionRuntime(request.Runtime)
	if err != nil {
		return nil, err
	}

	response, err := h.converter.CreatePlugin(ctx, user, authenticated, domain.PluginInput{
		Name:        request.Name,
		Description: request.Description.Value,
		Repository:  request.Repository,
		Branch:      request.Branch,
		Runtime:     runtime,
		Arguments:   request.Arguments,
	})
	if err != nil {
		return nil, err
	}

	p, err := toAPIManagedConversionPlugin(response)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (h *ogenHandler) GetConversionPlugin(ctx context.Context, params gatewayapi.GetConversionPluginParams) (*gatewayapi.ManagedConversionPlugin, error) {
	user, authenticated := authcontext.UserFromContext(ctx)

	response, err := h.converter.GetPlugin(ctx, user, authenticated, params.PluginId)
	if err != nil {
		return nil, err
	}

	p, err := toAPIManagedConversionPlugin(response)
	return &p, err
}

func (h *ogenHandler) ListConversionPlugins(ctx context.Context) ([]gatewayapi.ManagedConversionPlugin, error) {
	user, authenticated := authcontext.UserFromContext(ctx)

	response, err := h.converter.ListPlugins(ctx, user, authenticated)
	if err != nil {
		return nil, err
	}

	result := make([]gatewayapi.ManagedConversionPlugin, 0, len(response))
	for _, value := range response {
		p, err := toAPIManagedConversionPlugin(value)
		if err != nil {
			return nil, err
		}

		result = append(result, p)
	}

	return result, nil
}

func (h *ogenHandler) ReplaceConversionPlugin(ctx context.Context, request *gatewayapi.ReplaceConversionPluginRequest, params gatewayapi.ReplaceConversionPluginParams) (*gatewayapi.ManagedConversionPlugin, error) {
	user, authenticated := authcontext.UserFromContext(ctx)

	runtime, err := toDomainConversionRuntime(request.Runtime)
	if err != nil {
		return nil, err
	}

	response, err := h.converter.UpdatePlugin(ctx, user, authenticated, params.PluginId, domain.PluginInput{
		Name:        request.Name,
		Description: request.Description.Value,
		Repository:  request.Repository,
		Branch:      request.Branch,
		Runtime:     runtime,
		Arguments:   request.Arguments,
	}, request.Enabled)
	if err != nil {
		return nil, err
	}

	p, err := toAPIManagedConversionPlugin(response)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (h *ogenHandler) DeleteConversionPlugin(ctx context.Context, params gatewayapi.DeleteConversionPluginParams) error {
	user, authenticated := authcontext.UserFromContext(ctx)

	return h.converter.DeletePlugin(ctx, user, authenticated, params.PluginId)
}

func (h *ogenHandler) SyncConversionPlugin(ctx context.Context, params gatewayapi.SyncConversionPluginParams) (*gatewayapi.ManagedConversionPlugin, error) {
	user, authenticated := authcontext.UserFromContext(ctx)

	response, err := h.converter.SyncPlugin(ctx, user, authenticated, params.PluginId)
	if err != nil {
		return nil, err
	}

	p, err := toAPIManagedConversionPlugin(response)
	return &p, err
}

func (h *ogenHandler) CreateConversionRelation(ctx context.Context, request *gatewayapi.ConversionRelationInput) (*gatewayapi.ConversionRelation, error) {
	user, authenticated := authcontext.UserFromContext(ctx)

	response, err := h.converter.CreateRelation(ctx, user, authenticated, toDomainConversionRelation(request))
	if err != nil {
		return nil, err
	}

	relation := toAPIConversionRelation(response)
	return &relation, nil
}

func (h *ogenHandler) GetConversionRelation(ctx context.Context, params gatewayapi.GetConversionRelationParams) (*gatewayapi.ConversionRelation, error) {
	user, authenticated := authcontext.UserFromContext(ctx)
	response, err := h.converter.GetRelation(ctx, user, authenticated, params.ConversionId)
	if err != nil {
		return nil, err
	}
	relation := toAPIConversionRelation(response)
	return &relation, nil
}

func (h *ogenHandler) ListConversionRelations(ctx context.Context) ([]gatewayapi.ConversionRelation, error) {
	user, authenticated := authcontext.UserFromContext(ctx)
	response, err := h.converter.ListRelations(ctx, user, authenticated)
	if err != nil {
		return nil, err
	}
	result := make([]gatewayapi.ConversionRelation, 0, len(response))
	for _, value := range response {
		result = append(result, toAPIConversionRelation(value))
	}
	return result, nil
}

func (h *ogenHandler) ReplaceConversionRelation(ctx context.Context, request *gatewayapi.ConversionRelationInput, params gatewayapi.ReplaceConversionRelationParams) (*gatewayapi.ConversionRelation, error) {
	user, authenticated := authcontext.UserFromContext(ctx)
	response, err := h.converter.UpdateRelation(ctx, user, authenticated, params.ConversionId, toDomainConversionRelation(request))
	if err != nil {
		return nil, err
	}
	relation := toAPIConversionRelation(response)
	return &relation, nil
}

func (h *ogenHandler) DeleteConversionRelation(ctx context.Context, params gatewayapi.DeleteConversionRelationParams) error {
	user, authenticated := authcontext.UserFromContext(ctx)
	return h.converter.DeleteRelation(ctx, user, authenticated, params.ConversionId)
}
