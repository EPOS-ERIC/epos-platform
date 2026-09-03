//nolint:dupl,exhaustruct_v5,wrapcheck
package handlers

import (
	"context"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) ImportTurtleMetadata(ctx context.Context, body gatewayapi.ImportTurtleMetadataReq) (*gatewayapi.ImportResult, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	v, err := h.imports.Turtle(ctx, actor, body)
	if err != nil {
		return nil, err
	}
	result := &gatewayapi.ImportResult{Groups: int(v.Groups), Categories: int(v.Categories), Organizations: int(v.Organizations), ContactPoints: int(v.ContactPoints), Datasets: int(v.Datasets), SoftwareApplications: int(v.SoftwareApplications), SoftwareSourceCodes: int(v.SoftwareSourceCodes)}
	for _, warning := range v.Warnings {
		result.Warnings = append(result.Warnings, gatewayapi.ImportDiagnostic{Severity: warning.Severity, Subject: warning.Subject, Predicate: warning.Predicate, Message: warning.Message})
	}
	return result, nil
}
