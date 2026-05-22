package discovery

import (
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func mapDatasetItem(dataset metadataclient.DatasetSearchItem) (api.DatasetItem, error) {
	name, err := getField[string](dataset.Fields, "name")
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("get dataset name: %w", err)
	}

	return api.DatasetItem{
		BaseSearchItem: api.BaseSearchItem{
			Type: api.DatasetType,
			ID:   dataset.Id,
			Name: name,
			Href: buildGatewayHref(api.DatasetType, dataset.Id),
		},
	}, nil
}

func mapGenericItem(item metadataclient.GenericSearchItem) (api.SearchItem, error) {
	name, err := getField[string](item.Fields, "name")
	if err != nil {
		return nil, fmt.Errorf("get generic item name: %w", err)
	}

	resourceType, err := fromMetadataResourceType(metadataclient.ResourceType(item.GetType()))
	if err != nil {
		return nil, err
	}

	baseItem := api.BaseSearchItem{
		Type: resourceType,
		ID:   item.Id,
		Name: name,
		Href: buildGatewayHref(resourceType, item.Id),
	}

	switch resourceType {
	case api.OrganizationType:
		return api.OrganizationItem{BaseSearchItem: baseItem}, nil
	case api.SoftwareApplicationType:
		return api.SoftwareApplicationItem{BaseSearchItem: baseItem}, nil
	case api.SoftwareSourceCodeType:
		return api.SoftwareSourceCodeItem{BaseSearchItem: baseItem}, nil
	case api.WebserviceType:
		return api.WebServiceItem{BaseSearchItem: baseItem}, nil
	default:
		return nil, fmt.Errorf("unsupported gateway resource type: %q", resourceType)
	}
}

// TODO: this should probably be done better?
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
