package grpcapi

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
)

func toProtoImportResponse(value domain.ImportResult) (*metadatalifecyclev1.ImportTurtleResponse, error) {
	warnings := make([]*metadatalifecyclev1.ImportDiagnostic, len(value.Warnings))
	for i, warning := range value.Warnings {
		severity, err := toProtoImportSeverity(warning.Severity)
		if err != nil {
			return nil, err
		}
		warnings[i] = &metadatalifecyclev1.ImportDiagnostic{
			Severity:  severity,
			Subject:   warning.Subject,
			Predicate: warning.Predicate,
			Message:   warning.Message,
		}
	}

	return &metadatalifecyclev1.ImportTurtleResponse{
		Result: &metadatalifecyclev1.ImportResult{
			Groups:               int32(value.Groups),
			Categories:           int32(value.Categories),
			Organizations:        int32(value.Organizations),
			ContactPoints:        int32(value.ContactPoints),
			Datasets:             int32(value.Datasets),
			SoftwareApplications: int32(value.SoftwareApplications),
			SoftwareSourceCodes:  int32(value.SoftwareSourceCodes),
			Warnings:             warnings,
		},
	}, nil
}

func toProtoImportSeverity(value domain.ImportSeverity) (metadatalifecyclev1.ImportSeverity, error) {
	switch value {
	case domain.ImportSeverityWarning:
		return metadatalifecyclev1.ImportSeverity_IMPORT_SEVERITY_WARNING, nil
	case domain.ImportSeverityError:
		return metadatalifecyclev1.ImportSeverity_IMPORT_SEVERITY_ERROR, nil
	default:
		return 0, fmt.Errorf("unknown import severity: %q", value)
	}
}
