package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	api "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) ListConcepts(ctx context.Context) ([]api.Concept, error) {
	response, err := h.metadataLifecycleClient.ListConcepts(ctx)
	if err != nil {
		return nil, err
	}

	values := make([]api.Concept, 0, len(response.Concepts))
	for _, value := range response.Concepts {
		ref, err := toAPIConcept(value)
		if err != nil {
			return nil, err
		}

		values = append(values, *ref)
	}

	return values, nil
}

func (h *ogenHandler) GetConcept(ctx context.Context, params api.GetConceptParams) (*api.Concept, error) {
	response, err := h.metadataLifecycleClient.GetConcept(ctx, params.ConceptId.String())
	if err != nil {
		return nil, err
	}

	return toAPIConcept(response.Concept)
}

func (h *ogenHandler) CreateConcept(ctx context.Context, req *api.ConceptInput) (*api.Concept, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}

	response, err := h.metadataLifecycleClient.CreateConcept(ctx, actor, req.Name)
	if err != nil {
		return nil, err
	}

	return toAPIConcept(response.Concept)
}

func (h *ogenHandler) ReplaceConcept(ctx context.Context, req *api.ConceptInput, params api.ReplaceConceptParams) (*api.Concept, error) {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}

	response, err := h.metadataLifecycleClient.ReplaceConcept(ctx, actor, params.ConceptId.String(), req.Name)
	if err != nil {
		return nil, err
	}

	return toAPIConcept(response.Concept)
}

func (h *ogenHandler) DeleteConcept(ctx context.Context, params api.DeleteConceptParams) error {
	actor, ok := authcontext.ActorID(ctx)
	if !ok {
		return apperror.Default(apperror.CodeUnauthenticated)
	}

	return h.metadataLifecycleClient.DeleteConcept(ctx, actor, params.ConceptId.String())
}
