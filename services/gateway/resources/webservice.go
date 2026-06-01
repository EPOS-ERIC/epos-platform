package resources

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadatareader"
	"golang.org/x/sync/errgroup"
)

func (s *Service) GetWebserviceByID(ctx context.Context, id string) (api.Webservice, error) {
	fetchDetail := func(ctx context.Context, shape, id string) (*metadataclient.AccessDetailResponse, error) {
		return metadataaccess.FetchDetail(ctx, s.metadataClient, shape, id)
	}

	detail, err := fetchDetail(ctx, string(api.WebserviceType), id)
	if err != nil {
		return api.Webservice{}, fmt.Errorf("fetch webservice '%s' detail: %w", id, err)
	}

	webservice, err := mapWebservice(ctx, detail, fetchDetail)
	if err != nil {
		return api.Webservice{}, fmt.Errorf("map webservice '%s' detail: %w", id, err)
	}

	return webservice, nil
}

func mapWebservice(ctx context.Context, detail *metadataclient.AccessDetailResponse, fetch accessDetailFetcher) (api.Webservice, error) {
	properties := detail.GetProperties()
	reader := metadatareader.New(properties)

	name, err := reader.RequiredStringOneOf("name", "title")
	if err != nil {
		return api.Webservice{}, fmt.Errorf("get webservice name: %w", err)
	}

	issued, err := reader.OptionalTimeOneOf("datePublished", "issued")
	if err != nil {
		return api.Webservice{}, fmt.Errorf("get webservice issued: %w", err)
	}

	modified, err := reader.OptionalTimeOneOf("dateModified", "modified")
	if err != nil {
		return api.Webservice{}, fmt.Errorf("get webservice modified: %w", err)
	}

	identifiers, temporal, spatial, operations, apiDocumentation, err := aggregateWebserviceEntities(ctx, properties, fetch)
	if err != nil {
		return api.Webservice{}, err
	}

	contactPoints, err := reader.PassthroughRefs("contactPoint")
	if err != nil {
		return api.Webservice{}, fmt.Errorf("get webservice contact points: %w", err)
	}

	providers, err := reader.ResourceRefs("provider", api.OrganizationType)
	if err != nil {
		return api.Webservice{}, fmt.Errorf("get webservice provider: %w", err)
	}
	provider := api.ResourceRef{}
	if len(providers) > 0 {
		provider = providers[0]
	}

	concepts, err := reader.ResourceRefs("theme", api.ConceptType)
	if err != nil {
		return api.Webservice{}, fmt.Errorf("get webservice concepts: %w", err)
	}

	keywords, err := reader.Strings("keywords")
	if err != nil {
		return api.Webservice{}, fmt.Errorf("get webservice keywords: %w", err)
	}

	return api.Webservice{
		BaseDetailResource: api.BaseDetailResource{
			ID:          detail.GetId(),
			Name:        name,
			Description: reader.OptionalString("description"),
		},
		Identifiers:      identifiers,
		Entrypoint:       reader.OptionalString("entrypoint"),
		ContactPoints:    contactPoints,
		Provider:         provider,
		Issued:           issued,
		Modified:         modified,
		Spatial:          spatial,
		Temporal:         temporal,
		Keywords:         keywords,
		Concepts:         concepts,
		License:          reader.OptionalString("license"),
		Operations:       operations,
		APIDocumentation: apiDocumentation,
	}, nil
}

func aggregateWebserviceEntities(ctx context.Context, properties map[string]any, fetch accessDetailFetcher) ([]api.Identifier, *api.TemporalExtent, string, []api.Operation, *api.APIDocumentation, error) {
	reader := metadatareader.New(properties)

	var (
		identifiers      []api.Identifier
		temporal         *api.TemporalExtent
		spatial          string
		operations       []api.Operation
		apiDocumentation *api.APIDocumentation
	)

	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		mappedIdentifiers, err := aggregateIdentifiers(groupCtx, properties, fetch)
		if err != nil {
			return fmt.Errorf("get webservice identifiers: %w", err)
		}
		identifiers = mappedIdentifiers
		return nil
	})
	g.Go(func() error {
		mappedTemporal, err := aggregateTemporal(groupCtx, properties, fetch)
		if err != nil {
			return fmt.Errorf("get webservice temporal: %w", err)
		}
		temporal = mappedTemporal
		return nil
	})
	g.Go(func() error {
		mappedSpatial, err := aggregateSpatial(groupCtx, properties, fetch)
		if err != nil {
			return fmt.Errorf("get webservice spatial: %w", err)
		}
		spatial = mappedSpatial
		return nil
	})
	g.Go(func() error {
		mappedOperations, err := aggregateOperations(groupCtx, reader, "supportedOperation", fetch)
		if err != nil {
			return fmt.Errorf("get webservice operations: %w", err)
		}
		operations = mappedOperations
		return nil
	})
	g.Go(func() error {
		mappedAPIDocumentation, err := aggregateAPIDocumentationByKeys(groupCtx, reader, fetch, "conformsTo")
		if err != nil {
			return fmt.Errorf("get webservice api documentation: %w", err)
		}
		apiDocumentation = mappedAPIDocumentation
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, nil, "", nil, nil, fmt.Errorf("aggregate webservice entities: %w", err)
	}

	return identifiers, temporal, spatial, operations, apiDocumentation, nil
}
