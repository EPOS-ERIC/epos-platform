package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadatareader"
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
	reader := metadatareader.New(properties)

	name, err := reader.RequiredString("title")
	if err != nil {
		return api.Distribution{}, fmt.Errorf("get distribution title: %w", err)
	}

	issued, err := reader.OptionalTime("issued")
	if err != nil {
		return api.Distribution{}, fmt.Errorf("get distribution issued: %w", err)
	}

	modified, err := reader.OptionalTime("modified")
	if err != nil {
		return api.Distribution{}, fmt.Errorf("get distribution modified: %w", err)
	}

	distributionType, err := mapDistributionKind(reader)
	if err != nil {
		return api.Distribution{}, fmt.Errorf("get distribution type: %w", err)
	}

	webservice, err := reader.OptionalResourceRef("conformsTo", api.WebserviceType)
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
			Description: reader.OptionalString("description"),
		},
		DownloadURL: downloadURL,
		Operation:   operation,
		Webservice:  webservice,
		Format:      reader.OptionalString("format"),
		Issued:      issued,
		Modified:    modified,
		License:     reader.OptionalString("license"),
		Type:        distributionType,
	}, nil
}

func mapDistributionKind(reader metadatareader.Reader) (api.DistributionKind, error) {
	types, err := reader.Strings("type")
	if err != nil {
		return "", fmt.Errorf("get distribution types: %w", err)
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
	reader := metadatareader.New(properties)
	items, err := reader.Items("accessURL")
	if err != nil {
		return nil, "", fmt.Errorf("get access url items: %w", err)
	}
	if len(items) == 0 {
		return nil, "", nil
	}

	item := items[0]
	if distributionType == api.DistributionKindDownloadableFile {
		return nil, metadatareader.FirstNonEmpty(item.Value, item.Href, item.ID), nil
	}

	if item.ID == "" {
		return nil, item.Value, nil
	}

	operations, err := aggregateOperations(ctx, reader, "accessURL", fetch)
	if err != nil {
		return nil, "", err
	}
	if len(operations) == 0 {
		return nil, "", nil
	}

	return &operations[0], "", nil
}
