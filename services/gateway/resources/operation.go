package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadatareader"
	"golang.org/x/sync/errgroup"
)

func mapOperation(ctx context.Context, detail *metadataclient.AccessDetailResponse, fetch accessDetailFetcher) (api.Operation, error) {
	reader := metadatareader.New(detail.GetProperties())

	iriTemplate, err := aggregateIriTemplate(ctx, reader, fetch)
	if err != nil {
		return api.Operation{}, fmt.Errorf("get operation iri template: %w", err)
	}

	apiDocumentation, err := aggregateAPIDocumentationByKeys(ctx, reader, fetch, "dct:conformsTo", "schema:documentation")
	if err != nil {
		return api.Operation{}, fmt.Errorf("get operation api documentation: %w", err)
	}

	returns, err := reader.Strings("hydra:returns")
	if err != nil {
		return api.Operation{}, fmt.Errorf("get operation returns: %w", err)
	}

	return api.Operation{
		ID:               detail.GetId(),
		Method:           reader.OptionalString("hydra:method"),
		Returns:          returns,
		IriTemplate:      iriTemplate,
		APIDocumentation: apiDocumentation,
	}, nil
}

func aggregateIriTemplate(ctx context.Context, reader metadatareader.Reader, fetch accessDetailFetcher) (*api.IriTemplate, error) {
	items, err := reader.ItemsOneOf("hydra:pg_property", "hydra:property")
	if err != nil {
		return nil, fmt.Errorf("get iri template items: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	item := items[0]
	if item.ID == "" {
		template := metadatareader.FirstNonEmpty(item.Value, item.Title)
		if template == "" {
			return nil, nil
		}

		return &api.IriTemplate{Template: template}, nil
	}

	iriTemplateDetail, err := fetch(ctx, "iritemplate", item.ID)
	if err != nil {
		return nil, err
	}

	iriTemplateReader := metadatareader.New(iriTemplateDetail.GetProperties())
	template, err := iriTemplateReader.RequiredString("template")
	if err != nil {
		return nil, fmt.Errorf("get iri template template: %w", err)
	}

	mappings, err := aggregateIriTemplateMappings(ctx, iriTemplateReader, fetch)
	if err != nil {
		return nil, fmt.Errorf("get iri template mappings: %w", err)
	}

	return &api.IriTemplate{Template: template, Mappings: mappings}, nil
}

func aggregateIriTemplateMappings(ctx context.Context, reader metadatareader.Reader, fetch accessDetailFetcher) ([]api.IriTemplateMapping, error) {
	items, err := reader.Items("mapping")
	if err != nil {
		return nil, fmt.Errorf("get iri template mappings: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	mappings := make([]api.IriTemplateMapping, len(items))
	g, groupCtx := errgroup.WithContext(ctx)

	for i, item := range items {
		g.Go(func() error {
			mapping, err := mapIriTemplateMapping(groupCtx, item, fetch)
			if err != nil {
				return fmt.Errorf("mapping %d: %w", i, err)
			}
			mappings[i] = mapping
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("resolve iri template mappings: %w", err)
	}

	return mappings, nil
}

func mapIriTemplateMapping(ctx context.Context, item metadatareader.Item, fetch accessDetailFetcher) (api.IriTemplateMapping, error) {
	if item.ID == "" {
		return api.IriTemplateMapping{}, fmt.Errorf("missing id")
	}

	detail, err := fetch(ctx, "iritemplatemapping", item.ID)
	if err != nil {
		return api.IriTemplateMapping{}, err
	}

	reader := metadatareader.New(detail.GetProperties())
	variable, err := reader.RequiredString("variable")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping variable: %w", err)
	}

	label, err := reader.RequiredStringOneOf("pg_label", "label")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping label: %w", err)
	}

	rangeValue, err := reader.RequiredString("range")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping range: %w", err)
	}

	required, err := reader.RequiredBool("required")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping required: %w", err)
	}

	paramValue, err := reader.Strings("paramValue")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping param value: %w", err)
	}

	multipleValues, err := reader.OptionalBool("multipleValues")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping multiple values: %w", err)
	}

	readonlyValue, err := reader.OptionalBool("readonlyValue")
	if err != nil {
		return api.IriTemplateMapping{}, fmt.Errorf("get iri template mapping readonly value: %w", err)
	}

	return api.IriTemplateMapping{
		Variable:       variable,
		Label:          label,
		Range:          rangeValue,
		Property:       reader.OptionalString("property"),
		Required:       required,
		DefaultValue:   reader.OptionalString("defaultValue"),
		MinValue:       reader.OptionalString("minValue"),
		MaxValue:       reader.OptionalString("maxValue"),
		ParamValue:     paramValue,
		MultipleValues: multipleValues != nil && *multipleValues,
		ReadonlyValue:  readonlyValue != nil && *readonlyValue,
		ValuePattern:   reader.OptionalString("valuePattern"),
	}, nil
}

func aggregateAPIDocumentationByKeys(ctx context.Context, reader metadatareader.Reader, fetch accessDetailFetcher, keys ...string) (*api.APIDocumentation, error) {
	items, err := reader.ItemsOneOf(keys...)
	if err != nil {
		return nil, fmt.Errorf("get api documentation refs: %w", err)
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

func aggregateOperations(ctx context.Context, reader metadatareader.Reader, key string, fetch accessDetailFetcher) ([]api.Operation, error) {
	items, err := reader.Items(key)
	if err != nil {
		return nil, fmt.Errorf("get operations: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	operations := make([]api.Operation, len(items))
	g, groupCtx := errgroup.WithContext(ctx)

	for i, item := range items {
		g.Go(func() error {
			if item.ID == "" {
				return fmt.Errorf("operation %d: missing id", i)
			}

			detail, err := fetch(groupCtx, "operation", item.ID)
			if err != nil {
				return fmt.Errorf("operation %d: %w", i, err)
			}

			operation, err := mapOperation(groupCtx, detail, fetch)
			if err != nil {
				return fmt.Errorf("operation %d: %w", i, err)
			}

			operations[i] = operation
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("resolve operations: %w", err)
	}

	return operations, nil
}

func mapAPIDocumentation(detail *metadataclient.AccessDetailResponse) (*api.APIDocumentation, error) {
	docReader := metadatareader.New(detail.GetProperties())
	title, err := docReader.RequiredString("hydra:title")
	if err != nil {
		return nil, fmt.Errorf("get api documentation title: %w", err)
	}

	description, err := docReader.RequiredString("hydra:description")
	if err != nil {
		return nil, fmt.Errorf("get api documentation description: %w", err)
	}

	entrypoint, err := docReader.RequiredString("hydra:entrypoint")
	if err != nil {
		return nil, fmt.Errorf("get api documentation entrypoint: %w", err)
	}

	return &api.APIDocumentation{
		Title:       title,
		Description: description,
		Entrypoint:  entrypoint,
	}, nil
}
