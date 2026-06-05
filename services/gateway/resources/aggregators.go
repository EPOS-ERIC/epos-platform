package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadatareader"
	"golang.org/x/sync/errgroup"
)

const propertyValueShape = "propertyvalue"

func aggregateIdentifiers(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) ([]api.Identifier, error) {
	reader := metadatareader.New(properties)
	items, err := reader.ItemsOneOf("adms:identifier", "schema:identifier")
	if err != nil {
		return nil, fmt.Errorf("get identifiers: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	identifiers := make([]api.Identifier, len(items))
	g, groupCtx := errgroup.WithContext(ctx)

	for i, item := range items {
		g.Go(func() error {
			identifier, err := mapIdentifierItem(groupCtx, item, fetch)
			if err != nil {
				return fmt.Errorf("identifier %d: %w", i, err)
			}
			identifiers[i] = identifier
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("resolve identifiers: %w", err)
	}

	result := make([]api.Identifier, 0, len(identifiers))
	for _, identifier := range identifiers {
		if identifier.Type == "" || identifier.Value == "" {
			continue
		}
		result = append(result, identifier)
	}

	return result, nil
}

func mapIdentifierItem(ctx context.Context, item metadatareader.Item, fetch accessDetailFetcher) (api.Identifier, error) {
	if item.Value != "" {
		return api.Identifier{Type: "", Value: item.Value}, nil
	}

	if item.ID == "" {
		return api.Identifier{Type: "", Value: ""}, nil
	}

	shape := metadatareader.FirstNonEmpty(item.Type, "identifier")
	detail, err := fetch(ctx, shape, item.ID)
	if err != nil {
		return api.Identifier{}, err
	}

	properties := detail.GetProperties()
	reader := metadatareader.New(properties)
	if shape == propertyValueShape {
		values, err := reader.ItemsOneOf("schema:value", "schema:pg_value", "hydra:pg_value", "pg_value")
		if err != nil {
			return api.Identifier{}, fmt.Errorf("get property value identifier value: %w", err)
		}

		value := ""
		if len(values) > 0 {
			value = metadatareader.FirstNonEmpty(values[0].Value, values[0].ID, values[0].Title)
		}

		return api.Identifier{
			Type:  reader.OptionalString("schema:propertyID"),
			Value: value,
		}, nil
	}

	return api.Identifier{
		Type:  reader.OptionalStringOneOf("adms:schemeAgency", "epos:schemeAgency"),
		Value: reader.OptionalString("skos:notation"),
	}, nil
}

func aggregateTemporal(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) (*api.TemporalExtent, error) {
	reader := metadatareader.New(properties)
	items, err := reader.Items("dct:temporal")
	if err != nil {
		return nil, fmt.Errorf("get temporal items: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
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
	temporalReader := metadatareader.New(temporalProperties)
	start, err := temporalReader.OptionalTime("schema:startDate")
	if err != nil {
		return nil, fmt.Errorf("get temporal start date: %w", err)
	}

	end, err := temporalReader.OptionalTime("schema:endDate")
	if err != nil {
		return nil, fmt.Errorf("get temporal end date: %w", err)
	}

	if start == nil && end == nil {
		return nil, nil
	}

	return &api.TemporalExtent{Start: start, End: end}, nil
}

func aggregateAnnotationBody(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) (string, error) {
	reader := metadatareader.New(properties)
	items, err := reader.Items("dqv:hasQualityAnnotation")
	if err != nil {
		return "", fmt.Errorf("get quality annotation items: %w", err)
	}
	if len(items) == 0 {
		return "", nil
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

	return metadatareader.New(detail.GetProperties()).OptionalString("oa:hasBody"), nil
}

func aggregateSpatial(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) (string, error) {
	reader := metadatareader.New(properties)
	items, err := reader.Items("dct:spatial")
	if err != nil {
		return "", fmt.Errorf("get spatial items: %w", err)
	}
	if len(items) == 0 {
		return "", nil
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

	return metadatareader.New(detail.GetProperties()).OptionalString("locn:geometry"), nil
}
