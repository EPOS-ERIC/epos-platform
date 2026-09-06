package handlers

import (
	"errors"
	"fmt"
	"time"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoSearchResourcesRequest(value gatewayapi.DiscoverResourcesParams) (*metadatalifecyclev1.SearchResourcesRequest, error) {
	result := &metadatalifecyclev1.SearchResourcesRequest{Keywords: value.Keywords}
	for _, provider := range value.Provider {
		result.ProviderIds = append(result.ProviderIds, provider.String())
	}
	if value.Q.Set {
		query := value.Q.Value
		result.Query = &query
	}
	if value.GroupId.Set {
		groupID := value.GroupId.Value.String()
		result.GroupId = &groupID
	}
	if value.EditorId.Set {
		editorID := value.EditorId.Value
		result.EditorId = &editorID
	}
	if value.SpatialFilter.Set {
		spatialFilter := value.SpatialFilter.Value
		result.SpatialFilter = &spatialFilter
	}
	if value.DateFrom.Set {
		result.DateFrom = timestamppb.New(value.DateFrom.Value)
	}
	if value.DateTo.Set {
		result.DateTo = timestamppb.New(value.DateTo.Value)
	}

	for _, resourceType := range value.Type {
		mapped, err := toProtoDiscoveryResourceType(resourceType)
		if err != nil {
			return nil, err
		}
		result.ResourceTypes = append(result.ResourceTypes, mapped)
	}
	for _, concept := range value.Concept {
		mapped, err := toProtoDiscoveryConcept(concept)
		if err != nil {
			return nil, err
		}
		result.Concepts = append(result.Concepts, mapped)
	}
	for _, status := range value.VersioningStatus {
		mapped, err := toProtoDiscoveryStatus(status)
		if err != nil {
			return nil, err
		}
		result.VersioningStatuses = append(result.VersioningStatuses, mapped)
	}
	return result, nil
}

func toProtoDiscoveryResourceType(value gatewayapi.DiscoveryResourceType) (metadatalifecyclev1.ResourceType, error) {
	switch value {
	case gatewayapi.DiscoveryResourceTypeDataset:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_DATASET, nil
	case gatewayapi.DiscoveryResourceTypeSoftwareApplication:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_APPLICATION, nil
	case gatewayapi.DiscoveryResourceTypeSoftwareSourceCode:
		return metadatalifecyclev1.ResourceType_RESOURCE_TYPE_SOFTWARE_SOURCE_CODE, nil
	default:
		return 0, fmt.Errorf("invalid discovery resource type %q", value)
	}
}

func toProtoDiscoveryConcept(value gatewayapi.DiscoveryConcept) (metadatalifecyclev1.TopLevelConcept, error) {
	switch value {
	case gatewayapi.DiscoveryConceptDATA:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_DATA, nil
	case gatewayapi.DiscoveryConceptFACILITIES:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_FACILITIES, nil
	case gatewayapi.DiscoveryConceptSOFTWARE:
		return metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_SOFTWARE, nil
	default:
		return 0, fmt.Errorf("invalid discovery concept %q", value)
	}
}

func toProtoDiscoveryStatus(value gatewayapi.VersioningStatus) (metadatalifecyclev1.VersioningStatus, error) {
	switch value {
	case gatewayapi.VersioningStatusPUBLISHED:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_PUBLISHED, nil
	case gatewayapi.VersioningStatusDRAFT:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_DRAFT, nil
	case gatewayapi.VersioningStatusSUBMITTED:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_SUBMITTED, nil
	case gatewayapi.VersioningStatusARCHIVED:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_ARCHIVED, nil
	case gatewayapi.VersioningStatusREJECTED:
		return metadatalifecyclev1.VersioningStatus_VERSIONING_STATUS_REJECTED, nil
	default:
		return 0, fmt.Errorf("invalid discovery versioning status %q", value)
	}
}

type apiDiscoveryCommon struct {
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	Name       string
	Status     gatewayapi.VersioningStatus
	UpdatedAt  time.Time
	Concept    gatewayapi.DiscoveryConcept
	CreatedBy  gatewayapi.OptReference
	Group      gatewayapi.Reference
}

func toAPIDiscoveryCommon(resourceID, versionID, name string, status metadatalifecyclev1.VersioningStatus, updatedAt *timestamppb.Timestamp, concept metadatalifecyclev1.TopLevelConcept, createdBy, group *metadatalifecyclev1.DiscoveryReference) (apiDiscoveryCommon, error) {
	parsedResourceID, err := parseUUID("discovery.resource_id", resourceID)
	if err != nil {
		return apiDiscoveryCommon{}, err
	}
	parsedVersionID, err := parseUUID("discovery.version_id", versionID)
	if err != nil {
		return apiDiscoveryCommon{}, err
	}
	if updatedAt == nil {
		return apiDiscoveryCommon{}, errors.New("discovery.updated_at is required")
	}
	if err := updatedAt.CheckValid(); err != nil {
		return apiDiscoveryCommon{}, fmt.Errorf("invalid discovery.updated_at: %w", err)
	}
	parsedGroup, err := toAPIDiscoveryReference(group)
	if err != nil {
		return apiDiscoveryCommon{}, err
	}
	parsedStatus, err := toAPIVersioningStatus(status)
	if err != nil {
		return apiDiscoveryCommon{}, err
	}
	parsedConcept, err := toAPIDiscoveryConceptValue(concept)
	if err != nil {
		return apiDiscoveryCommon{}, err
	}

	common := apiDiscoveryCommon{
		ResourceID: parsedResourceID,
		VersionID:  parsedVersionID,
		Name:       name,
		Status:     parsedStatus,
		UpdatedAt:  updatedAt.AsTime(),
		Concept:    parsedConcept,
		Group:      parsedGroup,
	}
	if createdBy != nil {
		value, err := toAPIDiscoveryReference(createdBy)
		if err != nil {
			return apiDiscoveryCommon{}, err
		}
		common.CreatedBy = gatewayapi.NewOptReference(value)
	}
	return common, nil
}

func toAPIDiscovery(value *metadatalifecyclev1.DiscoveryItem) (gatewayapi.DiscoveryItem, error) {
	if value == nil {
		return gatewayapi.DiscoveryItem{}, errors.New("discovery item is nil")
	}
	if item := value.GetDataset(); item != nil {
		common, err := toAPIDiscoveryCommon(item.ResourceId, item.VersionId, item.Name, item.Status, item.UpdatedAt, item.Concept, item.CreatedBy, item.Group)
		if err != nil {
			return gatewayapi.DiscoveryItem{}, err
		}
		providers, err := toAPIDiscoveryReferences(item.Providers)
		if err != nil {
			return gatewayapi.DiscoveryItem{}, err
		}
		categories, err := toAPIDiscoveryReferences(item.Categories)
		if err != nil {
			return gatewayapi.DiscoveryItem{}, err
		}
		distributions, err := toAPIDiscoveryReferences(item.Distributions)
		if err != nil {
			return gatewayapi.DiscoveryItem{}, err
		}
		return gatewayapi.NewDatasetDiscoveryItemDiscoveryItem(gatewayapi.DatasetDiscoveryItem{
			Type:          gatewayapi.DatasetDiscoveryItemTypeDataset,
			ResourceID:    common.ResourceID,
			VersionID:     common.VersionID,
			Name:          common.Name,
			Status:        common.Status,
			UpdatedAt:     common.UpdatedAt,
			Concept:       common.Concept,
			CreatedBy:     common.CreatedBy,
			Group:         common.Group,
			Providers:     providers,
			Categories:    categories,
			Distributions: distributions,
		}), nil
	}
	if item := value.GetSoftwareApplication(); item != nil {
		common, creators, categories, err := toAPISoftwareDiscoveryFields(item.ResourceId, item.VersionId, item.Name, item.Status, item.UpdatedAt, item.Concept, item.CreatedBy, item.Group, item.Creators, item.Categories)
		if err != nil {
			return gatewayapi.DiscoveryItem{}, err
		}
		return gatewayapi.NewSoftwareApplicationDiscoveryItemDiscoveryItem(gatewayapi.SoftwareApplicationDiscoveryItem{
			Type:       gatewayapi.SoftwareApplicationDiscoveryItemTypeSoftwareApplication,
			ResourceID: common.ResourceID,
			VersionID:  common.VersionID,
			Name:       common.Name,
			Status:     common.Status,
			UpdatedAt:  common.UpdatedAt,
			Concept:    common.Concept,
			CreatedBy:  common.CreatedBy,
			Group:      common.Group,
			Creators:   creators,
			Categories: categories,
		}), nil
	}
	if item := value.GetSoftwareSourceCode(); item != nil {
		common, creators, categories, err := toAPISoftwareDiscoveryFields(item.ResourceId, item.VersionId, item.Name, item.Status, item.UpdatedAt, item.Concept, item.CreatedBy, item.Group, item.Creators, item.Categories)
		if err != nil {
			return gatewayapi.DiscoveryItem{}, err
		}
		return gatewayapi.NewSoftwareSourceCodeDiscoveryItemDiscoveryItem(gatewayapi.SoftwareSourceCodeDiscoveryItem{
			Type:       gatewayapi.SoftwareSourceCodeDiscoveryItemTypeSoftwareSourceCode,
			ResourceID: common.ResourceID,
			VersionID:  common.VersionID,
			Name:       common.Name,
			Status:     common.Status,
			UpdatedAt:  common.UpdatedAt,
			Concept:    common.Concept,
			CreatedBy:  common.CreatedBy,
			Group:      common.Group,
			Creators:   creators,
			Categories: categories,
		}), nil
	}
	return gatewayapi.DiscoveryItem{}, errors.New("discovery item resource is required")
}

func toAPISoftwareDiscoveryFields(resourceID, versionID, name string, status metadatalifecyclev1.VersioningStatus, updatedAt *timestamppb.Timestamp, concept metadatalifecyclev1.TopLevelConcept, createdBy, group *metadatalifecyclev1.DiscoveryReference, creators, categories []*metadatalifecyclev1.DiscoveryReference) (apiDiscoveryCommon, []gatewayapi.Reference, []gatewayapi.Reference, error) {
	common, err := toAPIDiscoveryCommon(resourceID, versionID, name, status, updatedAt, concept, createdBy, group)
	if err != nil {
		return apiDiscoveryCommon{}, nil, nil, err
	}
	creatorReferences, err := toAPIDiscoveryReferences(creators)
	if err != nil {
		return apiDiscoveryCommon{}, nil, nil, err
	}
	categoryReferences, err := toAPIDiscoveryReferences(categories)
	if err != nil {
		return apiDiscoveryCommon{}, nil, nil, err
	}
	return common, creatorReferences, categoryReferences, nil
}

func toAPIDiscoveryConceptValue(value metadatalifecyclev1.TopLevelConcept) (gatewayapi.DiscoveryConcept, error) {
	switch value {
	case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_DATA:
		return gatewayapi.DiscoveryConceptDATA, nil
	case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_FACILITIES:
		return gatewayapi.DiscoveryConceptFACILITIES, nil
	case metadatalifecyclev1.TopLevelConcept_TOP_LEVEL_CONCEPT_SOFTWARE:
		return gatewayapi.DiscoveryConceptSOFTWARE, nil
	default:
		return "", fmt.Errorf("unsupported discovery concept %q", value)
	}
}

func toAPIDiscoveryReference(value *metadatalifecyclev1.DiscoveryReference) (gatewayapi.Reference, error) {
	if value == nil {
		return gatewayapi.Reference{}, errors.New("discovery reference is required")
	}
	return gatewayapi.Reference{ID: value.Id, Name: value.Name}, nil
}

func toAPIDiscoveryReferences(values []*metadatalifecyclev1.DiscoveryReference) ([]gatewayapi.Reference, error) {
	result := make([]gatewayapi.Reference, 0, len(values))
	for _, value := range values {
		mapped, err := toAPIDiscoveryReference(value)
		if err != nil {
			return nil, err
		}
		result = append(result, mapped)
	}
	return result, nil
}
