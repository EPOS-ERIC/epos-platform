package resources

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadatareader"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/resourcehref"
)

func (s *Service) GetSoftwareApplicationByID(ctx context.Context, id string) (api.SoftwareApplication, error) {
	fetchDetail := func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error) {
		return metadataaccess.FetchDetail(ctx, s.metadataClient, shape, id)
	}

	detail, err := fetchDetail(ctx, string(api.SoftwareApplicationType), id)
	if err != nil {
		return api.SoftwareApplication{}, fmt.Errorf("fetch software application '%s' detail: %w", id, err)
	}

	softwareApplication, err := mapSoftwareApplication(ctx, detail, fetchDetail, s.log)
	if err != nil {
		return api.SoftwareApplication{}, fmt.Errorf("map software application '%s' detail: %w", id, err)
	}

	return softwareApplication, nil
}

func mapSoftwareApplication(ctx context.Context, detail *metadataclient.AccessDetailResponse, fetch accessDetailFetcher, log *slog.Logger) (api.SoftwareApplication, error) {
	properties := detail.GetProperties()
	reader := metadatareader.New(properties)

	name, err := reader.RequiredString("schema:name")
	if err != nil {
		return api.SoftwareApplication{}, fmt.Errorf("get software application name: %w", err)
	}

	identifiers, err := aggregateIdentifiers(ctx, properties, fetch)
	if err != nil {
		return api.SoftwareApplication{}, fmt.Errorf("get software application identifiers: %w", err)
	}

	concepts, err := reader.ResourceRefs("dcat:theme", api.ConceptType)
	if err != nil {
		return api.SoftwareApplication{}, fmt.Errorf("get software application concepts: %w", err)
	}

	creators, err := mapSoftwareCreators(ctx, reader, log, "software application")
	if err != nil {
		return api.SoftwareApplication{}, fmt.Errorf("get software application creators: %w", err)
	}

	keywords, err := mapSoftwareKeywords(reader)
	if err != nil {
		return api.SoftwareApplication{}, fmt.Errorf("get software application keywords: %w", err)
	}

	return api.SoftwareApplication{
		BaseDetailResource: api.BaseDetailResource{
			ID:          detail.GetId(),
			Name:        name,
			Description: reader.OptionalString("schema:description"),
		},
		MainEntityOfPage: reader.OptionalString("schema:mainEntityOfPage"),
		License:          reader.OptionalString("schema:license"),
		Version:          reader.OptionalStringOneOf("schema:softwareVersion", "version"),
		Keywords:         keywords,
		Concepts:         concepts,
		DownloadURL:      reader.OptionalString("schema:downloadUrl"),
		Requirements:     reader.OptionalString("schema:softwareRequirements"),
		Creator:          creators,
		Identifiers:      identifiers,
		OperatingSystem:  reader.OptionalString("schema:operatingSystem"),
	}, nil
}

func (s *Service) GetSoftwareSourceCodeByID(ctx context.Context, id string) (api.SoftwareSourceCode, error) {
	fetchDetail := func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error) {
		return metadataaccess.FetchDetail(ctx, s.metadataClient, shape, id)
	}

	detail, err := fetchDetail(ctx, string(api.SoftwareSourceCodeType), id)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("fetch software source code '%s' detail: %w", id, err)
	}

	softwareSourceCode, err := mapSoftwareSourceCode(ctx, detail, fetchDetail, s.log)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("map software source code '%s' detail: %w", id, err)
	}

	return softwareSourceCode, nil
}

func mapSoftwareSourceCode(ctx context.Context, detail *metadataclient.AccessDetailResponse, fetch accessDetailFetcher, log *slog.Logger) (api.SoftwareSourceCode, error) {
	properties := detail.GetProperties()
	reader := metadatareader.New(properties)

	name, err := reader.RequiredString("schema:name")
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("get software source code name: %w", err)
	}

	identifiers, err := aggregateIdentifiers(ctx, properties, fetch)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("get software source code identifiers: %w", err)
	}

	concepts, err := reader.ResourceRefs("dcat:theme", api.ConceptType)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("get software source code concepts: %w", err)
	}

	creators, err := mapSoftwareCreators(ctx, reader, log, "software source code")
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("get software source code creators: %w", err)
	}

	keywords, err := mapSoftwareKeywords(reader)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("get software source code keywords: %w", err)
	}

	return api.SoftwareSourceCode{
		BaseDetailResource: api.BaseDetailResource{
			ID:          detail.GetId(),
			Name:        name,
			Description: reader.OptionalString("schema:description"),
		},
		ProgrammingLanguage:  reader.OptionalString("schema:programmingLanguage"),
		Keywords:             keywords,
		Concepts:             concepts,
		Repository:           reader.OptionalString("schema:codeRepository"),
		License:              reader.OptionalString("schema:license"),
		SoftwareRequirements: reader.OptionalString("schema:softwareRequirements"),
		RuntimePlatform:      reader.OptionalString("schema:runtimePlatform"),
		Creator:              creators,
		MainEntityOfPage:     reader.OptionalString("schema:mainEntityOfPage"),
		Version:              reader.OptionalString("version"),
		Identifiers:          identifiers,
	}, nil
}

func mapSoftwareCreators(ctx context.Context, reader metadatareader.Reader, log *slog.Logger, resourceName string) ([]api.ResourceRef, error) {
	items, err := reader.Items("schema:creator")
	if err != nil {
		return nil, fmt.Errorf("get creator items: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	creators := make([]api.ResourceRef, 0, len(items))
	for i, item := range items {
		if item.ID == "" {
			return nil, fmt.Errorf("item %d: missing id", i)
		}

		if item.Type != "" && item.Type != string(api.OrganizationType) {
			if log != nil {
				log.WarnContext(ctx, "skip creator with unsupported type", "resource_type", resourceName, "creator_id", item.ID, "creator_type", item.Type)
			}
			continue
		}

		creators = append(creators, api.ResourceRef{
			ID:   item.ID,
			Name: item.Title,
			Href: resourcehref.Build(api.OrganizationType, item.ID),
		})
	}

	return creators, nil
}

func mapSoftwareKeywords(reader metadatareader.Reader) ([]string, error) {
	rawKeywords, err := reader.Strings("schema:keywords")
	if err != nil {
		return nil, fmt.Errorf("get keyword strings: %w", err)
	}
	if len(rawKeywords) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(rawKeywords))
	keywords := make([]string, 0, len(rawKeywords))
	for _, rawKeyword := range rawKeywords {
		for keyword := range strings.SplitSeq(rawKeyword, ",") {
			trimmed := strings.TrimSpace(keyword)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			keywords = append(keywords, trimmed)
		}
	}

	return keywords, nil
}
