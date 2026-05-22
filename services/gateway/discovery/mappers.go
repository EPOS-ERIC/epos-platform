package discovery

import (
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func mapDatasetItem(dataset metadataclient.DatasetSearchItem) (api.DatasetItem, error) {
	name, err := getField[string](dataset.Fields, "name")
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("TODO: %w", err)
	}

	provider, err := getField[string](dataset.Fields, "publisher")
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("TODO: %w", err)
	}

	concepts, err := getField[string](dataset.Fields, "theme")
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("TODO: %w", err)
	}

	return api.DatasetItem{
		BaseSearchItem: api.BaseSearchItem{
			Type: api.DatasetType,
			ID:   dataset.Id,
			Name: name,
			// TODO: we have to build this in the right way, pointing to the details endpoint of the gateway, not the metadataclient
			Href: "TODO",
		},
		// TODO: check if this should be an array of strings or just a single string. from the metadataclient it looks we only get a single string but in theory the publishers can be more than 1?
		DataProviders: []string{provider},
		Concepts:      []string{concepts},
		// TODO: see how to do this, in theory we should get it from the search, but maybe we decided to fetch from the details? TBD
		Operations: []string{},
	}, nil
}

func mapGenericItem(item metadataclient.GenericSearchItem) (api.BaseSearchItem, error) {
	name, err := getField[string](item.Fields, "name")
	if err != nil {
		return api.BaseSearchItem{}, fmt.Errorf("TODO: %w", err)
	}

	baseItem := api.BaseSearchItem{
		ID:   item.Id,
		Name: name,
		// TODO: we have to build this in the right way, pointing to the details endpoint of the gateway, not the metadataclient
		Href: "TODO",
	}

	resourceType, err := fromMetadataResourceType(metadataclient.ResourceType(item.GetType()))
	if err != nil {
		return api.BaseSearchItem{}, err
	}

	baseItem.Type = resourceType

	return baseItem, nil
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
