package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"golang.org/x/sync/errgroup"
)

const genericDistributionTypeIRI = "http://www.w3.org/ns/dcat#Distribution"

func (s *Service) GetDistributionByID(ctx context.Context, id string) (api.Distribution, error) {
	fetchDetail := func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error) {
		return metadataaccess.FetchDetail(ctx, s.metadataClient, shape, id)
	}

	detail, err := fetchDetail(ctx, string(api.DistributionType), id)
	if err != nil {
		return api.Distribution{}, fmt.Errorf("fetch distribution '%s' detail: %w", id, err)
	}

	distribution, err := mapDistribution(ctx, detail, fetchDetail)
	if err != nil {
		return api.Distribution{}, fmt.Errorf("map distribution '%s' detail: %w", id, err)
	}

	return distribution, nil
}

func mapDistribution(ctx context.Context, detail *metadataclient.AccessDetailResponse, fetch accessDetailFetcher) (api.Distribution, error) {
	properties := detail.GetProperties()
	reader := newPropertyReader(properties)

	name, err := reader.requiredString("title")
	if err != nil {
		return api.Distribution{}, fmt.Errorf("get distribution title: %w", err)
	}

	issued, err := reader.optionalTime("issued")
	if err != nil {
		return api.Distribution{}, fmt.Errorf("get distribution issued: %w", err)
	}

	modified, err := reader.optionalTime("modified")
	if err != nil {
		return api.Distribution{}, fmt.Errorf("get distribution modified: %w", err)
	}

	distributionType, err := mapDistributionKind(reader)
	if err != nil {
		return api.Distribution{}, fmt.Errorf("get distribution type: %w", err)
	}

	webservice, err := reader.optionalResourceRef("conformsTo", api.WebserviceType)
	if err != nil {
		return api.Distribution{}, fmt.Errorf("get distribution webservice: %w", err)
	}
	if distributionType == api.DistributionKindDownloadableFile {
		webservice = nil
	}

	operation, downloadURL, err := aggregateDistributionOperation(ctx, properties, fetch, distributionType)
	if err != nil {
		return api.Distribution{}, fmt.Errorf("get distribution operation: %w", err)
	}

	return api.Distribution{
		BaseDetailResource: api.BaseDetailResource{
			ID:          detail.GetId(),
			Name:        name,
			Description: reader.optionalString("description"),
		},
		DownloadURL: downloadURL,
		Operation:   operation,
		Webservice:  webservice,
		Format:      reader.optionalString("format"),
		Issued:      issued,
		Modified:    modified,
		License:     reader.optionalString("license"),
		Type:        distributionType,
	}, nil
}

func mapDistributionKind(reader propertyReader) (api.DistributionKind, error) {
	types, err := reader.strings("type")
	if err != nil {
		return "", err
	}
	if len(types) == 0 {
		return "", nil
	}

	var distributionType api.DistributionKind
	for _, value := range types {
		if value == "" || value == genericDistributionTypeIRI {
			continue
		}

		parsedType, err := api.NewDistributionKindFromValue(value)
		if err != nil {
			return "", fmt.Errorf("parse distribution kind '%s': %w", value, err)
		}

		if distributionType != "" && distributionType != parsedType {
			return "", fmt.Errorf("multiple distribution kinds: %s and %s", distributionType, parsedType)
		}

		distributionType = parsedType
	}

	return distributionType, nil
}

func aggregateDistributionOperation(ctx context.Context, properties map[string]any, fetch accessDetailFetcher, distributionType api.DistributionKind) (*api.Operation, string, error) {
	reader := newPropertyReader(properties)
	items, err := reader.items("accessURL")
	if err != nil {
		return nil, "", err
	}
	if len(items) == 0 {
		return nil, "", nil
	}

	item := items[0]
	if distributionType == api.DistributionKindDownloadableFile {
		return nil, firstNonEmpty(item.Value, item.Href, item.ID), nil
	}

	if item.ID == "" {
		return nil, item.Value, nil
	}

	operationDetail, err := fetch(ctx, "operation", item.ID)
	if err != nil {
		return nil, "", err
	}

	operation, err := mapOperation(ctx, operationDetail, fetch)
	if err != nil {
		return nil, "", err
	}

	return &operation, "", nil
}

func mapOperation(ctx context.Context, detail *metadataclient.AccessDetailResponse, fetch accessDetailFetcher) (api.Operation, error) {
	reader := newPropertyReader(detail.GetProperties())

	iriTemplate, err := aggregateIriTemplate(ctx, reader, fetch)
	if err != nil {
		return api.Operation{}, fmt.Errorf("get operation iri template: %w", err)
	}

	apiDocumentation, err := aggregateAPIDocumentation(ctx, reader, fetch)
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
	items, err := firstItems(reader, "pg_property", "property")
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

	label, err := requiredStringOneOf(reader, "pg_label", "label")
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

func aggregateAPIDocumentation(ctx context.Context, reader propertyReader, fetch accessDetailFetcher) (*api.APIDocumentation, error) {
	items, err := firstItems(reader, "apiDocumentation", "api documentation", "documentation")
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

func firstItems(reader propertyReader, keys ...string) ([]propertyItem, error) {
	for _, key := range keys {
		items, err := reader.items(key)
		if err != nil {
			return nil, err
		}
		if len(items) > 0 {
			return items, nil
		}
	}

	return nil, nil
}

func requiredStringOneOf(reader propertyReader, keys ...string) (string, error) {
	for _, key := range keys {
		value := reader.optionalString(key)
		if value != "" {
			return value, nil
		}
	}

	return "", fmt.Errorf("missing fields %v", keys)
}
