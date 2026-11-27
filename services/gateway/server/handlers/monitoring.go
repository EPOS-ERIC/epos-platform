package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) ListMonitoringDistributions(ctx context.Context) ([]gatewayapi.MonitoringConcept, error) {
	actorID, ok := authcontext.ActorID(ctx)
	if !ok {
		return nil, apperror.Default(apperror.CodeUnauthenticated)
	}

	response, err := h.metadataLifecycleClient.ListMonitoringDistributions(ctx, actorID)
	if err != nil {
		return nil, err
	}

	concepts, err := toAPIMonitoringConcepts(response.GetConcepts())
	if err != nil {
		return nil, err
	}

	return concepts, nil
}
