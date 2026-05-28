package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"golang.org/x/sync/errgroup"
)

func mapOperation(ctx context.Context, detail *metadataclient.AccessDetailResponse, fetch accessDetailFetcher) (api.Operation, error) {
	reader := newPropertyReader(detail.GetProperties())

	iriTemplate, err := aggregateIriTemplate(ctx, reader, fetch)
	if err != nil {
		return api.Operation{}, fmt.Errorf("get operation iri template: %w", err)
	}

	apiDocumentation, err := aggregateAPIDocumentationByKeys(ctx, reader, fetch, "apiDocumentation", "api documentation", "documentation")
	if err != nil {
		return api.Operation{}, fmt.Errorf("get operation api documentation: %w", err)
	}

	returns, err := reader.strings("returns")
	if err != nil {
		return api.Operation{}, fmt.Errorf("get operation returns: %w", err)
	}

	return api.Operation{
		ID:               detail.GetId(),
		Method:           reader.optionalString("method"),
		Returns:          returns,
		IriTemplate:      iriTemplate,
		APIDocumentation: apiDocumentation,
	}, nil
}

func aggregateIriTemplate(ctx context.Context, reader propertyReader, fetch accessDetailFetcher) (*api.IriTemplate, error) {
	items, err := reader.itemsOneOf("pg_property", "property")
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	item := items[0]
	if item.ID == "" {
		template := firstNonEmpty(item.Value, item.Title)
		if template == "" {
			return nil, nil
		}

		return &api.IriTemplate{Template: template}, nil
	}

	iriTemplateDetail, err := fetch(ctx, "iritemplate", item.ID)
	if err != nil {
		return nil, err
	}

	iriTemplateReader := newPropertyReader(iriTemplateDetail.GetProperties())
	template, err := iriTemplateReader.requiredString("template")
	if err != nil {
		return nil, fmt.Errorf("get iri template template: %w", err)
	}

	mappings, err := aggregateIriTemplateMappings(ctx, iriTemplateReader, fetch)
	if err != nil {
		return nil, fmt.Errorf("get iri template mappings: %w", err)
	}

	return &api.IriTemplate{Template: template, Mappings: mappings}, nil
}

func aggregateIriTemplateMappings(ctx context.Context, reader propertyReader, fetch accessDetailFetcher) ([]api.IriTemplateMapping, error) {
	items, err := reader.items("mapping")
	if err != nil || len(items) == 0 {
		return nil, err
	}

	mappings := make([]api.IriTemplateMapping, len(items))
	g, groupCtx := errgroup.WithContext(ctx)

	for i, item := range items {
		idx := i
		it := item
		g.Go(func() error {
			mapping, err := mapIriTemplateMapping(groupCtx, it, fetch)
			if err != nil {
				return fmt.Errorf("mapping %d: %w", idx, err)
			}
			mappings[idx] = mapping
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("resolve iri template mappings: %w", err)
	}

	return mappings, nil
}

func mapIriTemplateMapping(ctx context.Context, item propertyItem, fetch accessDetailFetcher) (api.IriTemplateMapping, error) {
	if item.ID == "" {
		return api.IriTemplateMapping{}, fmt.Errorf("missing id")
	}

	detail, err := fetch(ctx, "iritemplatemapping", item.ID)
	if err != nil {
		return api.IriTemplateMapping{}, err
	}

	reader := newPropertyReader(detail.GetProperties())
	variable, err := reader.requiredString("variable")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping variable: %w", err)
	}

	label, err := reader.requiredStringOneOf("pg_label", "label")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping label: %w", err)
	}

	rangeValue, err := reader.requiredString("range")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping range: %w", err)
	}

	required, err := reader.requiredBool("required")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping required: %w", err)
	}

	paramValue, err := reader.strings("paramValue")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping param value: %w", err)
	}

	multipleValues, err := reader.optionalBool("multipleValues")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping multiple values: %w", err)
	}

	readonlyValue, err := reader.optionalBool("readonlyValue")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping readonly value: %w", err)
	}

	return api.IriTemplateMapping{
		Variable:       variable,
		Label:          label,
		Range:          rangeValue,
		Property:       reader.optionalString("property"),
		Required:       required,
		DefaultValue:   reader.optionalString("defaultValue"),
		MinValue:       reader.optionalString("minValue"),
		MaxValue:       reader.optionalString("maxValue"),
		ParamValue:     paramValue,
		MultipleValues: multipleValues != nil && *multipleValues,
		ReadonlyValue:  readonlyValue != nil && *readonlyValue,
		ValuePattern:   reader.optionalString("valuePattern"),
	}, nil
}

func aggregateAPIDocumentationByKeys(ctx context.Context, reader propertyReader, fetch accessDetailFetcher, keys ...string) (*api.APIDocumentation, error) {
	items, err := reader.itemsOneOf(keys...)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	item := items[0]
	if item.ID == "" {
		return nil, nil
	}

	detail, err := fetch(ctx, "apidocumentation", item.ID)
	if err != nil {
		return nil, err
	}

	return mapAPIDocumentation(detail)
}

func aggregateOperations(ctx context.Context, reader propertyReader, key string, fetch accessDetailFetcher) ([]api.Operation, error) {
	items, err := reader.items(key)
	if err != nil || len(items) == 0 {
		return nil, err
	}

	operations := make([]api.Operation, len(items))
	g, groupCtx := errgroup.WithContext(ctx)

	for i, item := range items {
		idx := i
		it := item
		g.Go(func() error {
			if it.ID == "" {
				return fmt.Errorf("operation %d: missing id", idx)
			}

			detail, err := fetch(groupCtx, "operation", it.ID)
			if err != nil {
				return fmt.Errorf("operation %d: %w", idx, err)
			}

			operation, err := mapOperation(groupCtx, detail, fetch)
			if err != nil {
				return fmt.Errorf("operation %d: %w", idx, err)
			}

			operations[idx] = operation
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("resolve operations: %w", err)
	}

	return operations, nil
}

func mapAPIDocumentation(detail *metadataclient.AccessDetailResponse) (*api.APIDocumentation, error) {
	docReader := newPropertyReader(detail.GetProperties())
	title, err := docReader.requiredString("title")
	if err != nil {
		return nil, fmt.Errorf("get api documentation title: %w", err)
	}

	description, err := docReader.requiredString("description")
	if err != nil {
		return nil, fmt.Errorf("get api documentation description: %w", err)
	}

	entrypoint, err := docReader.requiredString("entrypoint")
	if err != nil {
		return nil, fmt.Errorf("get api documentation entrypoint: %w", err)
	}

	return &api.APIDocumentation{
		Title:       title,
		Description: description,
		Entrypoint:  entrypoint,
	}, nil
}
