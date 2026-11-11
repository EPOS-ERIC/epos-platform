package handlers

import (
	"fmt"
	"math"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	semanticaggregatorv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/semantic_aggregator/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
)

func toProtoImportResourcesRequest(request *gatewayapi.ImportResourcesRequest) (*semanticaggregatorv1.ImportResourcesRequest, error) {
	if request == nil {
		return nil, apperror.Field("request", "Request is required.", nil)
	}
	if !request.BaseUri.IsAbs() {
		return nil, apperror.Field("baseUri", "Value must be an absolute URI.", nil)
	}
	if request.MappingVersion > math.MaxInt32 {
		return nil, apperror.Field("mappingVersion", "Value is too large.", nil)
	}

	result := &semanticaggregatorv1.ImportResourcesRequest{
		Turtle:         []byte(request.Turtle),
		BaseUri:        request.BaseUri.String(),
		MappingId:      request.MappingId,
		MappingVersion: int32(request.MappingVersion),
		References:     make([]*semanticaggregatorv1.ReferenceBinding, 0, len(request.References)),
	}
	for i, reference := range request.References {
		if !reference.Iri.IsAbs() {
			return nil, apperror.Field(fmt.Sprintf("references[%d].iri", i), "Value must be an absolute URI.", nil)
		}
		kind, err := toProtoImportReferenceKind(reference.Kind)
		if err != nil {
			return nil, apperror.Field(fmt.Sprintf("references[%d].kind", i), "Value is invalid.", err)
		}
		result.References = append(result.References, &semanticaggregatorv1.ReferenceBinding{
			Kind: kind,
			Iri:  reference.Iri.String(),
			Id:   reference.ID.String(),
		})
	}
	return result, nil
}

func toProtoImportReferenceKind(value gatewayapi.ImportReferenceBindingKind) (semanticaggregatorv1.ReferenceKind, error) {
	switch value {
	case gatewayapi.ImportReferenceBindingKindOrganization:
		return semanticaggregatorv1.ReferenceKind_REFERENCE_KIND_ORGANIZATION, nil
	case gatewayapi.ImportReferenceBindingKindContactPoint:
		return semanticaggregatorv1.ReferenceKind_REFERENCE_KIND_CONTACT_POINT, nil
	case gatewayapi.ImportReferenceBindingKindCategory:
		return semanticaggregatorv1.ReferenceKind_REFERENCE_KIND_CATEGORY, nil
	default:
		return semanticaggregatorv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED, fmt.Errorf("unknown reference kind: %q", value)
	}
}

func toAPIImportResourcesResult(response *semanticaggregatorv1.ImportResourcesResponse) (*gatewayapi.ImportResourcesResult, error) {
	if response == nil {
		return nil, fmt.Errorf("semantic-aggregator import response is empty")
	}
	result := &gatewayapi.ImportResourcesResult{
		Imported: make([]gatewayapi.ImportedResource, 0, len(response.Imported)),
		Warnings: make([]gatewayapi.ImportResourceWarning, 0, len(response.Warnings)),
		Errors:   make([]gatewayapi.ImportResourceFailure, 0, len(response.Errors)),
	}
	for i, imported := range response.Imported {
		value, err := toAPIImportedResource(imported)
		if err != nil {
			return nil, fmt.Errorf("map semantic-aggregator imported resource %d: %w", i, err)
		}
		result.Imported = append(result.Imported, value)
	}
	if response.Failure != nil {
		failure, err := toAPIImportResourceFailure(response.Failure)
		if err != nil {
			return nil, fmt.Errorf("map semantic-aggregator import failure: %w", err)
		}
		result.Failure = gatewayapi.NewOptImportResourceFailure(failure)
	}
	for i, warning := range response.Warnings {
		value, err := toAPIImportResourceWarning(warning)
		if err != nil {
			return nil, fmt.Errorf("map semantic-aggregator import warning %d: %w", i, err)
		}
		result.Warnings = append(result.Warnings, value)
	}
	for i, importError := range response.Errors {
		value, err := toAPIImportResourceFailure(importError)
		if err != nil {
			return nil, fmt.Errorf("map semantic-aggregator import error %d: %w", i, err)
		}
		result.Errors = append(result.Errors, value)
	}
	return result, nil
}

func toAPIImportResourceWarning(value *semanticaggregatorv1.ImportWarning) (gatewayapi.ImportResourceWarning, error) {
	if value == nil || value.Subject == "" || value.Message == "" {
		return gatewayapi.ImportResourceWarning{}, fmt.Errorf("import warning is incomplete")
	}
	return gatewayapi.ImportResourceWarning{Subject: value.Subject, Message: value.Message}, nil
}

func toAPIImportedResource(value *semanticaggregatorv1.ImportedResource) (gatewayapi.ImportedResource, error) {
	if value == nil {
		return gatewayapi.ImportedResource{}, fmt.Errorf("resource is empty")
	}
	resourceID, err := parseUUID("resource_id", value.ResourceId)
	if err != nil {
		return gatewayapi.ImportedResource{}, err
	}
	versionID, err := parseUUID("version_id", value.VersionId)
	if err != nil {
		return gatewayapi.ImportedResource{}, err
	}
	kind, err := toAPIImportedResourceKind(value.Kind)
	if err != nil {
		return gatewayapi.ImportedResource{}, err
	}
	return gatewayapi.ImportedResource{Subject: value.Subject, ResourceId: resourceID, VersionId: versionID, Kind: kind}, nil
}

func toAPIImportedResourceKind(value semanticaggregatorv1.ImportedResourceKind) (gatewayapi.ImportedResourceKind, error) {
	switch value {
	case semanticaggregatorv1.ImportedResourceKind_IMPORTED_RESOURCE_KIND_DATASET:
		return gatewayapi.ImportedResourceKindDataset, nil
	case semanticaggregatorv1.ImportedResourceKind_IMPORTED_RESOURCE_KIND_SOFTWARE_APPLICATION:
		return gatewayapi.ImportedResourceKindSoftwareApplication, nil
	case semanticaggregatorv1.ImportedResourceKind_IMPORTED_RESOURCE_KIND_SOFTWARE_SOURCE_CODE:
		return gatewayapi.ImportedResourceKindSoftwareSourceCode, nil
	default:
		return "", fmt.Errorf("unknown imported resource kind: %q", value)
	}
}

func toAPIImportResourceFailure(value *semanticaggregatorv1.ImportFailure) (gatewayapi.ImportResourceFailure, error) {
	result := gatewayapi.ImportResourceFailure{Subject: value.Subject, Code: value.Code, OutcomeUnknown: value.OutcomeUnknown}
	if value.Message != "" {
		result.Message = gatewayapi.NewOptString(value.Message)
	}
	if value.Phase == semanticaggregatorv1.ImportPhase_IMPORT_PHASE_UNSPECIFIED {
		return result, nil
	}
	phase, err := toAPIImportResourceFailurePhase(value.Phase)
	if err != nil {
		return gatewayapi.ImportResourceFailure{}, err
	}
	result.Phase = gatewayapi.NewOptImportResourceFailurePhase(phase)
	return result, nil
}

func toAPIImportResourceFailurePhase(value semanticaggregatorv1.ImportPhase) (gatewayapi.ImportResourceFailurePhase, error) {
	switch value {
	case semanticaggregatorv1.ImportPhase_IMPORT_PHASE_CONTACT_RESOLUTION:
		return gatewayapi.ImportResourceFailurePhaseContactResolution, nil
	case semanticaggregatorv1.ImportPhase_IMPORT_PHASE_DATASET_CREATION:
		return gatewayapi.ImportResourceFailurePhaseDatasetCreation, nil
	case semanticaggregatorv1.ImportPhase_IMPORT_PHASE_CATEGORY_CREATION:
		return gatewayapi.ImportResourceFailurePhaseCategoryCreation, nil
	case semanticaggregatorv1.ImportPhase_IMPORT_PHASE_ORGANIZATION_CREATION:
		return gatewayapi.ImportResourceFailurePhaseOrganizationCreation, nil
	default:
		return "", fmt.Errorf("unknown import failure phase: %q", value)
	}
}
