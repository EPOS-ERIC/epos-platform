package handlers

import (
	gatewayapi "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/gatewayapi"
	metadatalifecycleclient "github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadata-lifecycle"
)

func toAPIDiscovery(value metadatalifecycleclient.DiscoveryItem) (gatewayapi.DiscoveryItem, error) {
	resourceID, err := parseUUID("discovery.resource_id", value.ResourceId)
	if err != nil {
		return gatewayapi.DiscoveryItem{}, err
	}
	versionID, err := parseUUID("discovery.version_id", value.VersionId)
	if err != nil {
		return gatewayapi.DiscoveryItem{}, err
	}
	result := gatewayapi.DiscoveryItem{
		Type:       gatewayapi.DiscoveryResourceType(value.Type),
		ResourceID: resourceID,
		VersionID:  versionID,
		Name:       value.Name,
		Status:     gatewayapi.DiscoveryItemStatus(value.Status),
		UpdatedAt:  value.UpdatedAt,
		Concept:    gatewayapi.DiscoveryItemConcept(value.Concept),
		Group:      toAPIDiscoveryReference(value.Group),
	}
	if value.CreatedBy != nil {
		result.CreatedBy.SetTo(gatewayapi.Reference{
			ID:   value.CreatedBy.Id,
			Name: value.CreatedBy.Name,
		})
	}
	result.Providers = toAPIDiscoveryReferences(value.Providers)
	result.Creators = toAPIDiscoveryReferences(value.Creators)
	result.Categories = toAPIDiscoveryReferences(value.Categories)
	result.Distributions = toAPIDiscoveryReferences(value.Distributions)
	return result, nil
}

func toAPIDiscoveryReferences(values []metadatalifecycleclient.DiscoveryReference) []gatewayapi.Reference {
	result := make([]gatewayapi.Reference, 0, len(values))
	for _, value := range values {
		result = append(result, toAPIDiscoveryReference(value))
	}
	return result
}

func toAPIDiscoveryReference(value metadatalifecycleclient.DiscoveryReference) gatewayapi.Reference {
	return gatewayapi.Reference{
		ID:   value.Id,
		Name: value.Name,
	}
}
