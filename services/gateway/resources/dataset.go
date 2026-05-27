package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"golang.org/x/sync/errgroup"
)

type accessDetailFetcher func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error)

func (s *Service) GetDatasetByID(ctx context.Context, id string) (api.Dataset, error) {
	fetchDetail := func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error) {
		return metadataaccess.FetchDetail(ctx, s.metadataClient, shape, id)
	}

	detail, err := fetchDetail(ctx, string(api.DatasetType), id)
	if err != nil {
		return api.Dataset{}, fmt.Errorf("fetch dataset %q detail: %w", id, err)
	}

	dataset, err := mapDataset(ctx, detail, fetchDetail)
	if err != nil {
		return api.Dataset{}, fmt.Errorf("map dataset %q detail: %w", id, err)
	}

	return dataset, nil
}

func mapDataset(ctx context.Context, detail *metadataclient.AccessDetailResponse, fetch accessDetailFetcher) (api.Dataset, error) {
	properties := detail.GetProperties()
	reader := newPropertyReader(properties)

	name, err := reader.requiredString("title")
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset title: %w", err)
	}

	status, err := reader.optionalVersioningStatus("status")
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset status: %w", err)
	}

	created, err := reader.optionalTime("created")
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset created: %w", err)
	}

	modified, err := reader.optionalTime("modified")
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset modified: %w", err)
	}

	issued, err := reader.optionalTime("issued")
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset issued: %w", err)
	}

	identifiers, temporal, spatial, qualityAnnotation, err := aggregateDatasetEntities(ctx, properties, fetch)
	if err != nil {
		return api.Dataset{}, err
	}

	contactPoints, err := reader.passthroughRefs("contact point")
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset contact points: %w", err)
	}

	concepts, err := reader.resourceRefs("theme", api.ConceptType)
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset concepts: %w", err)
	}

	publisher, err := reader.resourceRefs("publisher", api.OrganizationType)
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset publisher: %w", err)
	}

	distributions, err := reader.resourceRefs("distribution", api.DistributionType)
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset distributions: %w", err)
	}

	isVersionOf, err := reader.optionalResourceRef("is version of", api.DatasetType)
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset is version of: %w", err)
	}

	keywords, err := reader.strings("keyword")
	if err != nil {
		return api.Dataset{}, fmt.Errorf("get dataset keywords: %w", err)
	}

	return api.Dataset{
		BaseDetailResource: api.BaseDetailResource{
			ID:          detail.GetId(),
			Name:        name,
			Description: reader.optionalString("description"),
		},
		Created:            created,
		Modified:           modified,
		VersionInfo:        reader.optionalString("version info"),
		ContactPoints:      contactPoints,
		Keywords:           keywords,
		Concepts:           concepts,
		Publisher:          publisher,
		Issued:             issued,
		Spatial:            spatial,
		Identifiers:        identifiers,
		Temporal:           temporal,
		Status:             status,
		Distributions:      distributions,
		AccessRights:       reader.optionalString("access rights"),
		AccrualPeriodicity: reader.optionalString("accrual periodicity"),
		IsVersionOf:        isVersionOf,
		QualityAnnotation:  qualityAnnotation,
		Version:            reader.optionalString("version"),
	}, nil
}

func aggregateDatasetEntities(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) ([]api.Identifier, *api.TemporalExtent, string, string, error) {
	var (
		identifiers       []api.Identifier
		temporal          *api.TemporalExtent
		spatial           string
		qualityAnnotation string
	)

	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		mappedIdentifiers, err := aggregateIdentifiers(groupCtx, properties, fetch)
		if err != nil {
			return fmt.Errorf("get dataset identifiers: %w", err)
		}
		identifiers = mappedIdentifiers
		return nil
	})
	g.Go(func() error {
		mappedTemporal, err := aggregateTemporal(groupCtx, properties, fetch)
		if err != nil {
			return fmt.Errorf("get dataset temporal: %w", err)
		}
		temporal = mappedTemporal
		return nil
	})
	g.Go(func() error {
		mappedQualityAnnotation, err := aggregateAnnotationBody(groupCtx, properties, fetch)
		if err != nil {
			return fmt.Errorf("get dataset quality annotation: %w", err)
		}
		qualityAnnotation = mappedQualityAnnotation
		return nil
	})
	g.Go(func() error {
		mappedSpatial, err := aggregateSpatial(groupCtx, properties, fetch)
		if err != nil {
			return fmt.Errorf("get dataset spatial: %w", err)
		}
		spatial = mappedSpatial
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, nil, "", "", fmt.Errorf("aggregate dataset entities: %w", err)
	}

	return identifiers, temporal, spatial, qualityAnnotation, nil
}
