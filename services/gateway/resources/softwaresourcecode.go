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
)

func (s *Service) GetSoftwareSourceCodeByID(ctx context.Context, id string) (api.SoftwareSourceCode, error) {
	fetchDetail := func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error) {
		return metadataaccess.FetchDetail(ctx, s.metadataClient, shape, id)
	}

	detail, err := fetchDetail(ctx, string(api.SoftwareSourceCodeType), id)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("fetch software source code '%s' detail: %w", id, err)
	}

	softwareSourceCode, err := mapSoftwareSourceCode(ctx, detail, s.log)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("map software source code '%s' detail: %w", id, err)
	}

	return softwareSourceCode, nil
}

func mapSoftwareSourceCode(ctx context.Context, detail *metadataclient.AccessDetailResponse, log *slog.Logger) (api.SoftwareSourceCode, error) {
	properties := detail.GetProperties()
	reader := metadatareader.New(properties)

	name, err := reader.RequiredString("schema:name")
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("get software source code name: %w", err)
	}

	contactPoints, err := reader.PassthroughRefs("schema:contactPoint")
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("get software source code contact points: %w", err)
	}

	concepts, err := reader.ResourceRefs("dcat:theme", api.ConceptType)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("get software source code concepts: %w", err)
	}

	creators, err := mapSoftwareSourceCodeCreators(ctx, reader, log)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("get software source code creators: %w", err)
	}

	keywords, err := mapSoftwareSourceCodeKeywords(reader)
	if err != nil {
		return api.SoftwareSourceCode{}, fmt.Errorf("get software source code keywords: %w", err)
	}

	return api.SoftwareSourceCode{
		BaseDetailResource: api.BaseDetailResource{
			ID:          detail.GetId(),
			Name:        name,
			Description: reader.OptionalString("schema:description"),
		},
		ContactPoints:        contactPoints,
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
	}, nil
}

func mapSoftwareSourceCodeCreators(ctx context.Context, reader metadatareader.Reader, log *slog.Logger) ([]api.ResourceRef, error) {
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
				log.WarnContext(ctx, "skip software source code creator with unsupported type", "creator_id", item.ID, "creator_type", item.Type)
			}
			continue
		}

		creators = append(creators, api.ResourceRef{
			ID:   item.ID,
			Name: item.Title,
			Href: metadatareader.FirstNonEmpty(item.Href, item.ID),
		})
	}

	return creators, nil
}

func mapSoftwareSourceCodeKeywords(reader metadatareader.Reader) ([]string, error) {
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
		for _, keyword := range strings.Split(rawKeyword, ",") {
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
