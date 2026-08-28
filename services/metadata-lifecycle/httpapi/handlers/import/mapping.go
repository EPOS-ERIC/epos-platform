package imports

import "github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"

func toResponse(value domain.ImportResult) resultResponse {
	return resultResponse{
		Groups:               value.Groups,
		Categories:           value.Categories,
		Organizations:        value.Organizations,
		ContactPoints:        value.ContactPoints,
		Datasets:             value.Datasets,
		SoftwareApplications: value.SoftwareApplications,
		SoftwareSourceCodes:  value.SoftwareSourceCodes,
		Warnings:             toDiagnostics(value.Warnings),
	}
}

func toDiagnostics(values []domain.ImportDiagnostic) []diagnostic {
	result := make([]diagnostic, 0, len(values))
	for _, value := range values {
		result = append(result, diagnostic{
			Severity:  value.Severity,
			Subject:   value.Subject,
			Predicate: value.Predicate,
			Message:   value.Message,
		})
	}
	return result
}
