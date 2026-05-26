package discovery

import (
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func mapDatasetItem(item metadataclient.SearchItem) (api.DatasetItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.DatasetItem{}, err
	}

	themes, err := getOptionalStringSliceField(item.Fields, "theme")
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("get dataset concepts: %w", err)
	}

	publishers, err := getOptionalStringSliceField(item.Fields, "publishers")
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("get dataset providers: %w", err)
	}

	return api.DatasetItem{
		BaseSearchItem: baseItem,
		Provider:       mapProviders(publishers),
		Concepts:       mapConcepts(themes),
		Distributions:  []api.DatasetDistributionSearchItem{},
	}, nil
}

func mapOrganizationItem(item metadataclient.SearchItem) (api.OrganizationItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.OrganizationItem{}, err
	}

	return api.OrganizationItem{BaseSearchItem: baseItem}, nil
}

func mapSoftwareApplicationItem(item metadataclient.SearchItem) (api.SoftwareApplicationItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.SoftwareApplicationItem{}, err
	}

	publishers, err := getOptionalStringSliceField(item.Fields, "publishers")
	if err != nil {
		return api.SoftwareApplicationItem{}, fmt.Errorf("get software application providers: %w", err)
	}

	return api.SoftwareApplicationItem{
		BaseSearchItem: baseItem,
		Provider:       mapProviders(publishers),
	}, nil
}

func mapSoftwareSourceCodeItem(item metadataclient.SearchItem) (api.SoftwareSourceCodeItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.SoftwareSourceCodeItem{}, err
	}

	publishers, err := getOptionalStringSliceField(item.Fields, "publishers")
	if err != nil {
		return api.SoftwareSourceCodeItem{}, fmt.Errorf("get software source code providers: %w", err)
	}

	return api.SoftwareSourceCodeItem{
		BaseSearchItem: baseItem,
		Provider:       mapProviders(publishers),
	}, nil
}

func mapWebServiceItem(item metadataclient.SearchItem) (api.WebServiceItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.WebServiceItem{}, err
	}

	publishers, err := getOptionalStringSliceField(item.Fields, "publishers")
	if err != nil {
		return api.WebServiceItem{}, fmt.Errorf("get webservice providers: %w", err)
	}

	themes, err := getOptionalStringSliceField(item.Fields, "theme")
	if err != nil {
		return api.WebServiceItem{}, fmt.Errorf("get webservice concepts: %w", err)
	}

	return api.WebServiceItem{
		BaseSearchItem: baseItem,
		Provider:       mapProviders(publishers),
		Concepts:       mapConcepts(themes),
	}, nil
}

func mapBaseItem(item metadataclient.SearchItem) (api.BaseSearchItem, error) {
	name, err := getField[string](item.Fields, "name")
	if err != nil {
		return api.BaseSearchItem{}, fmt.Errorf("get item name: %w", err)
	}

	resourceType, err := fromMetadataResourceType(item.GetType())
	if err != nil {
		return api.BaseSearchItem{}, err
	}

	return api.BaseSearchItem{
		Type: resourceType,
		ID:   item.Id,
		Name: name,
		Href: buildGatewayHref(resourceType, item.Id),
	}, nil
}

func mapProviders(providerIDs []string) []api.ResourceRef {
	providers := make([]api.ResourceRef, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		providers = append(providers, api.ResourceRef{ID: providerID})
	}

	return providers
}

func mapConcepts(conceptIDs []string) []api.ResourceRef {
	concepts := make([]api.ResourceRef, 0, len(conceptIDs))
	for _, conceptID := range conceptIDs {
		concepts = append(concepts, api.ResourceRef{ID: conceptID})
	}

	return concepts
}

func buildGatewayHref(resourceType api.ResourceType, id string) string {
	return strings.TrimPrefix(fmt.Sprintf("v1/%s/%s", resourceType, id), "/")
}

func getField[T any](fields map[string]any, key string) (T, error) {
	var zero T
	raw, ok := fields[key]
	if !ok {
		return zero, fmt.Errorf("missing field %q", key)
	}

	val, ok := raw.(T)
	if !ok {
		return zero, fmt.Errorf("field %q: expected %T, got %T", key, zero, raw)
	}

	return val, nil
}

func getOptionalStringSliceField(fields map[string]any, key string) ([]string, error) {
	raw, ok := fields[key]
	if !ok {
		return nil, nil
	}

	if val, ok := raw.([]string); ok {
		return val, nil
	}

	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("field %q: expected []string or []interface{}, got %T", key, raw)
	}

	result := make([]string, 0, len(values))
	for i, value := range values {
		strValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("field %q[%d]: expected string, got %T", key, i, value)
		}

		result = append(result, strValue)
	}

	return result, nil
}
