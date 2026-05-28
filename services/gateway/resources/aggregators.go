package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"golang.org/x/sync/errgroup"
)

const propertyValueShape = "propertyvalue"

func aggregateIdentifiers(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) ([]api.Identifier, error) {
	// TODO: fix/semplify this once the metadata-lifecycle fixes the identifiers in all the entities
	reader := newPropertyReader(properties)
	items, err := reader.items("identifier")
	if err != nil || len(items) == 0 {
		return nil, err
	}

	identifiers := make([]api.Identifier, len(items))
	g, groupCtx := errgroup.WithContext(ctx)

	for i, item := range items {
		idx := i
		it := item
		g.Go(func() error {
			identifier, err := mapIdentifierItem(groupCtx, it, fetch)
			if err != nil {
				return fmt.Errorf("identifier %d: %w", idx, err)
			}
			identifiers[idx] = identifier
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("resolve identifiers: %w", err)
	}

	result := make([]api.Identifier, 0, len(identifiers))
	for _, identifier := range identifiers {
		if identifier.Type == "" && identifier.Value == "" {
			continue
		}
		result = append(result, identifier)
	}

	return result, nil
}

func mapIdentifierItem(ctx context.Context, item propertyItem, fetch accessDetailFetcher) (api.Identifier, error) {
	if item.Value != "" {
		return api.Identifier{Value: item.Value}, nil
	}

	if item.ID == "" {
		return api.Identifier{}, nil
	}

	shape := firstNonEmpty(item.Type, "identifier")
	detail, err := fetch(ctx, shape, item.ID)
	if err != nil {
		return api.Identifier{}, err
	}

	properties := detail.GetProperties()
	reader := newPropertyReader(properties)
	if shape == propertyValueShape {
		values, err := reader.items("pg_value")
		if err != nil {
			return api.Identifier{}, fmt.Errorf("get property value identifier value: %w", err)
		}

		value := ""
		if len(values) > 0 {
			value = firstNonEmpty(values[0].Value, values[0].ID, values[0].Title)
		}

		return api.Identifier{
			Type:  reader.optionalString("propertyID"),
			Value: value,
		}, nil
	}

	return api.Identifier{
		Type:  reader.optionalString("scheme agency"),
		Value: reader.optionalString("notation"),
	}, nil
}

func aggregateTemporal(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) (*api.TemporalExtent, error) {
	reader := newPropertyReader(properties)
	items, err := reader.items("temporal")
	if err != nil || len(items) == 0 {
		return nil, err
	}

	item := items[0]
	if item.ID == "" {
		return nil, nil
	}

	detail, err := fetch(ctx, "periodoftime", item.ID)
	if err != nil {
		return nil, err
	}

	temporalProperties := detail.GetProperties()
	temporalReader := newPropertyReader(temporalProperties)
	start, err := temporalReader.optionalTime("start date")
	if err != nil {
		return nil, fmt.Errorf("get temporal start date: %w", err)
	}

	end, err := temporalReader.optionalTime("end date")
	if err != nil {
		return nil, fmt.Errorf("get temporal end date: %w", err)
	}

	if start == nil && end == nil {
		return nil, nil
	}

	return &api.TemporalExtent{Start: start, End: end}, nil
}

func aggregateAnnotationBody(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) (string, error) {
	reader := newPropertyReader(properties)
	items, err := reader.items("has quality annotation")
	if err != nil || len(items) == 0 {
		return "", err
	}

	item := items[0]
	if item.Value != "" {
		return item.Value, nil
	}

	if item.ID == "" {
		return "", nil
	}

	detail, err := fetch(ctx, "annotation", item.ID)
	if err != nil {
		return "", err
	}

	return newPropertyReader(detail.GetProperties()).optionalString("has body"), nil
}

func aggregateSpatial(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) (string, error) {
	reader := newPropertyReader(properties)
	items, err := reader.items("spatial")
	if err != nil || len(items) == 0 {
		return "", err
	}

	item := items[0]
	if item.Value != "" {
		return item.Value, nil
	}

	if item.ID == "" {
		return "", nil
	}

	detail, err := fetch(ctx, "location", item.ID)
	if err != nil {
		return "", err
	}

	return newPropertyReader(detail.GetProperties()).optionalString("geometry"), nil
}
