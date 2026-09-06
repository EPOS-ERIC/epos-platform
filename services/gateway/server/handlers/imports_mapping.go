package handlers

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func toGatewayImportResult(response *metadatalifecyclev1.ImportTurtleResponse) (*gatewayapi.ImportResult, error) {
	if response == nil || response.Result == nil {
		return nil, fmt.Errorf("metadata-lifecycle import response is empty")
	}

	result := &gatewayapi.ImportResult{
		Groups:               int(response.Result.Groups),
		Categories:           int(response.Result.Categories),
		Organizations:        int(response.Result.Organizations),
		ContactPoints:        int(response.Result.ContactPoints),
		Datasets:             int(response.Result.Datasets),
		SoftwareApplications: int(response.Result.SoftwareApplications),
		SoftwareSourceCodes:  int(response.Result.SoftwareSourceCodes),
	}
	for i, warning := range response.Result.Warnings {
		if warning == nil {
			return nil, fmt.Errorf("metadata-lifecycle import response warning %d is empty", i)
		}
		severity, err := toGatewayImportSeverity(warning.Severity)
		if err != nil {
			return nil, fmt.Errorf("map metadata-lifecycle import warning %d: %w", i, err)
		}
		result.Warnings = append(result.Warnings, gatewayapi.ImportDiagnostic{
			Severity:  severity,
			Subject:   warning.Subject,
			Predicate: warning.Predicate,
			Message:   warning.Message,
		})
	}
	return result, nil
}

func toGatewayImportSeverity(value metadatalifecyclev1.ImportSeverity) (string, error) {
	switch value {
	case metadatalifecyclev1.ImportSeverity_IMPORT_SEVERITY_WARNING:
		return "warning", nil
	case metadatalifecyclev1.ImportSeverity_IMPORT_SEVERITY_ERROR:
		return "error", nil
	default:
		return "", fmt.Errorf("unknown import severity: %q", value)
	}
}
