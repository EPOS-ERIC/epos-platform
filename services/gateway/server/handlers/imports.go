package handlers

import (
	"context"
	"fmt"
	"io"

	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func (h *ogenHandler) ImportTurtleMetadata(ctx context.Context, body gatewayapi.ImportTurtleMetadataReq) (*gatewayapi.ImportResult, error) {
	actor, ok := actorID(ctx)
	if !ok {
		return nil, authMissing()
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read import body: %w", err)
	}
	v, err := h.metadataLifecycle.ImportTurtle(ctx, actor, data)
	if err != nil {
		return nil, fmt.Errorf("import turtle metadata: %w", err)
	}
	result := &gatewayapi.ImportResult{
		Groups:               int(v.Groups),
		Categories:           int(v.Categories),
		Organizations:        int(v.Organizations),
		ContactPoints:        int(v.ContactPoints),
		Datasets:             int(v.Datasets),
		SoftwareApplications: int(v.SoftwareApplications),
		SoftwareSourceCodes:  int(v.SoftwareSourceCodes),
	}
	for _, warning := range v.Warnings {
		result.Warnings = append(result.Warnings, gatewayapi.ImportDiagnostic{
			Severity:  warning.Severity,
			Subject:   warning.Subject,
			Predicate: warning.Predicate,
			Message:   warning.Message,
		})
	}
	return result, nil
}
