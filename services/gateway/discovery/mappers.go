package discovery

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/gatewayhref"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadatareader"
	"golang.org/x/sync/errgroup"
)

var errMissingField = errors.New("missing field")

func (s *Service) mapDatasetItem(ctx context.Context, item metadataclient.SearchItem) (api.DatasetItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("map dataset base item: %w", err)
	}

	detail, err := metadataaccess.FetchDetail(ctx, s.metadataClient, string(api.DatasetType), baseItem.ID)
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("fetch dataset '%s' details: %w", baseItem.ID, err)
	}

	reader := metadatareader.New(detail.GetProperties())

	concepts, err := reader.ResourceRefs("dcat:theme", api.ConceptType)
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("get dataset concepts: %w", err)
	}

	publishers, err := reader.ResourceRefs("dct:publisher", api.OrganizationType)
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("get dataset providers: %w", err)
	}

	providerIDs := resourceRefIDs(publishers)
	providers, err := s.mapProviders(ctx, providerIDs)
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("map dataset providers: %w", err)
	}

	distributions, err := s.mapDistributions(ctx, baseItem.ID)
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("map dataset distributions: %w", err)
	}

	return api.DatasetItem{
		BaseSearchItem: baseItem,
		Providers:      providers,
		Concepts:       concepts,
		Distributions:  distributions,
	}, nil
}

func mapOrganizationItem(item metadataclient.SearchItem) (api.OrganizationItem, error) {
	// TODO: is it correct that sometimes an organization item from the metadataclient doesn't have a legal name? what do we put here in those cases?
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.OrganizationItem{}, fmt.Errorf("map organization base item: %w", err)
	}

	return api.OrganizationItem{BaseSearchItem: baseItem}, nil
}

func (s *Service) mapSoftwareApplicationItem(ctx context.Context, item metadataclient.SearchItem) (api.SoftwareApplicationItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.SoftwareApplicationItem{}, fmt.Errorf("map software application base item: %w", err)
	}

	publishers, err := s.getSoftwareCreators(ctx, string(api.SoftwareApplicationType), baseItem.ID)
	if err != nil {
		return api.SoftwareApplicationItem{}, fmt.Errorf("get software application providers: %w", err)
	}

	providers, err := s.mapProviders(ctx, publishers)
	if err != nil {
		return api.SoftwareApplicationItem{}, fmt.Errorf("map software application providers: %w", err)
	}

	return api.SoftwareApplicationItem{
		BaseSearchItem: baseItem,
		Provider:       providers,
	}, nil
}

func (s *Service) mapSoftwareSourceCodeItem(ctx context.Context, item metadataclient.SearchItem) (api.SoftwareSourceCodeItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.SoftwareSourceCodeItem{}, fmt.Errorf("map software source code base item: %w", err)
	}

	publishers, err := s.getSoftwareCreators(ctx, string(api.SoftwareSourceCodeType), baseItem.ID)
	if err != nil {
		return api.SoftwareSourceCodeItem{}, fmt.Errorf("get software source code providers: %w", err)
	}

	providers, err := s.mapProviders(ctx, publishers)
	if err != nil {
		return api.SoftwareSourceCodeItem{}, fmt.Errorf("map software source code providers: %w", err)
	}

	return api.SoftwareSourceCodeItem{
		BaseSearchItem: baseItem,
		Provider:       providers,
	}, nil
}

func (s *Service) mapWebServiceItem(ctx context.Context, item metadataclient.SearchItem) (api.WebServiceItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.WebServiceItem{}, fmt.Errorf("map webservice base item: %w", err)
	}

	detail, err := metadataaccess.FetchDetail(ctx, s.metadataClient, string(api.WebserviceType), baseItem.ID)
	if err != nil {
		return api.WebServiceItem{}, fmt.Errorf("fetch webservice '%s' details: %w", baseItem.ID, err)
	}

	reader := metadatareader.New(detail.GetProperties())

	publishers, err := reader.ResourceRefs("schema:provider", api.OrganizationType)
	if err != nil {
		return api.WebServiceItem{}, fmt.Errorf("get webservice providers: %w", err)
	}

	providerRefs, err := s.mapProviders(ctx, resourceRefIDs(publishers))
	if err != nil {
		return api.WebServiceItem{}, fmt.Errorf("map webservice providers: %w", err)
	}

	themes, err := getLinkedIDs(reader, "dcat:theme")
	if err != nil {
		if !errors.Is(err, errMissingField) {
			return api.WebServiceItem{}, fmt.Errorf("get webservice concepts: %w", err)
		}

		themes = nil
	}

	var concepts []api.ResourceRef

	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		mappedConcepts, err := s.mapConcepts(groupCtx, themes)
		if err != nil {
			return fmt.Errorf("map webservice concepts: %w", err)
		}
		concepts = mappedConcepts
		return nil
	})

	if err := g.Wait(); err != nil {
		return api.WebServiceItem{}, fmt.Errorf("map webservice item dependencies: %w", err)
	}

	provider := api.ResourceRef{ID: "", Name: "", Href: ""}
	if len(providerRefs) > 0 {
		provider = providerRefs[0]
	}

	return api.WebServiceItem{
		BaseSearchItem: baseItem,
		Provider:       provider,
		Concepts:       concepts,
	}, nil
}

func mapBaseItem(item metadataclient.SearchItem) (api.BaseSearchItem, error) {
	reader := metadatareader.New(item.Fields)

	nameKey, err := getSearchItemNameKey(item.GetType())
	if err != nil {
		return api.BaseSearchItem{}, fmt.Errorf("resolve name key for metadata resource type '%s': %w", item.GetType(), err)
	}

	name, err := reader.RequiredString(nameKey)
	if err != nil {
		return api.BaseSearchItem{}, fmt.Errorf("get item name: %w", err)
	}

	resourceType, err := fromMetadataResourceType(item.GetType())
	if err != nil {
		return api.BaseSearchItem{}, fmt.Errorf("map metadata resource type '%s': %w", item.GetType(), err)
	}

	return api.BaseSearchItem{
		Type: resourceType,
		ID:   item.Id,
		Name: name,
		Href: gatewayhref.Build(resourceType, item.Id),
	}, nil
}

func resourceRefIDs(refs []api.ResourceRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.ID != "" {
			ids = append(ids, ref.ID)
		}
	}
	return ids
}

func getSearchItemNameKey(resourceType metadataclient.ResourceType) (string, error) {
	switch resourceType {
	case metadataclient.DATASET:
		return "dct:title", nil
	case metadataclient.ORGANIZATION:
		return "schema:legalName", nil
	case metadataclient.SOFTWAREAPPLICATION, metadataclient.SOFTWARESOURCECODE, metadataclient.WEBSERVICE:
		return "schema:name", nil
	default:
		return "", fmt.Errorf("unsupported metadata resource type: '%s'", resourceType)
	}
}

func (s *Service) mapProviders(ctx context.Context, providerIDs []string) ([]api.ResourceRef, error) {
	s.log.Debug("mapping providers", "providers count", len(providerIDs))
	providers := make([]api.ResourceRef, len(providerIDs))
	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.fanOutLimit)

	for i, providerID := range providerIDs {
		g.Go(func() error {
			provider := api.ResourceRef{
				ID:   providerID,
				Name: "",
				Href: gatewayhref.Build(api.OrganizationType, providerID),
			}

			detail, err := metadataaccess.FetchDetail(groupCtx, s.metadataClient, string(api.OrganizationType), providerID)
			if err != nil {
				s.log.Error("provider details fetch failed", "provider_id", providerID, "error", err)
				return fmt.Errorf("fetch provider '%s' details: %w", providerID, err)
			}

			name, err := metadatareader.New(detail.GetProperties()).RequiredString("schema:legalName")
			if err != nil {
				s.log.Warn("provider legal name extraction failed", "provider_id", providerID, "error", err)
				return fmt.Errorf("extract provider '%s' legal name: %w", providerID, err)
			}
			provider.Name = name

			providers[i] = provider
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("map providers: %w", err)
	}

	return providers, nil
}

func (s *Service) mapDistributions(ctx context.Context, distributionID string) ([]api.DatasetDistributionSearchItem, error) {
	detail, err := metadataaccess.FetchDetail(ctx, s.metadataClient, string(api.DatasetType), distributionID)
	if err != nil {
		return nil, fmt.Errorf("fetch dataset '%s' details: %w", distributionID, err)
	}

	refs, err := metadatareader.New(detail.GetProperties()).ResourceRefs("dcat:distribution", api.DistributionType)
	if err != nil {
		return nil, fmt.Errorf("extract dataset '%s' distribution refs: %w", distributionID, err)
	}
	if len(refs) == 0 {
		return []api.DatasetDistributionSearchItem{}, nil
	}

	distributions := make([]api.DatasetDistributionSearchItem, len(refs))
	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.fanOutLimit)

	for i, distribution := range refs {
		g.Go(func() error {
			name := distribution.Name
			if name == "" {
				distributionDetail, err := metadataaccess.FetchDetail(groupCtx, s.metadataClient, string(api.DistributionType), distribution.ID)
				if err != nil {
					s.log.Warn("distribution details fetch failed", "distribution_id", distribution.ID, "error", err)
				} else {
					name, err = metadatareader.New(distributionDetail.GetProperties()).RequiredString("dct:title")
					if err != nil {
						s.log.Warn("distribution title extraction failed", "distribution_id", distribution.ID, "error", err)
						name = ""
					}
				}
			}

			distributions[i] = api.DatasetDistributionSearchItem{
				ID:   distribution.ID,
				Name: name,
				Href: gatewayhref.Build(api.DistributionType, distribution.ID),
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("map dataset '%s' distributions: %w", distributionID, err)
	}

	return distributions, nil
}

func (s *Service) mapConcepts(ctx context.Context, conceptIDs []string) ([]api.ResourceRef, error) {
	concepts := make([]api.ResourceRef, len(conceptIDs))
	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.fanOutLimit)

	for i, conceptID := range conceptIDs {
		g.Go(func() error {
			concept := api.ResourceRef{
				ID:   conceptID,
				Name: "",
				Href: gatewayhref.Build(api.ConceptType, conceptID),
			}

			detail, err := metadataaccess.FetchDetail(groupCtx, s.metadataClient, string(api.ConceptType), conceptID)
			if err != nil {
				return fmt.Errorf("fetch concept '%s' details: %w", conceptID, err)
			}

			label, err := metadatareader.New(detail.GetProperties()).RequiredString("skos:prefLabel")
			if err != nil {
				return fmt.Errorf("extract concept '%s' prefLabel: %w", conceptID, err)
			}
			concept.Name = label

			concepts[i] = concept
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("map concepts: %w", err)
	}

	return concepts, nil
}

func (s *Service) getSoftwareCreators(ctx context.Context, softwareType, softwareID string) ([]string, error) {
	detail, err := metadataaccess.FetchDetail(ctx, s.metadataClient, softwareType, softwareID)
	if err != nil {
		return nil, fmt.Errorf("fetch software '%s' details: %w", softwareID, err)
	}

	creators, err := getLinkedIDs(metadatareader.New(detail.GetProperties()), "schema:creator")
	if err != nil {
		if errors.Is(err, errMissingField) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("extract software '%s' creators: %w", softwareID, err)
	}

	return creators, nil
}

func getLinkedIDs(reader metadatareader.Reader, key string) ([]string, error) {
	items, err := reader.Items(key)
	if err != nil {
		return nil, fmt.Errorf("read linked ids for '%s': %w", key, err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%w '%s'", errMissingField, key)
	}

	ids := make([]string, 0, len(items))
	for i, item := range items {
		id := metadatareader.FirstNonEmpty(item.ID, item.Value)
		if id == "" {
			return nil, fmt.Errorf("field '%s[%d]': missing id/value", key, i)
		}
		ids = append(ids, id)
	}

	return ids, nil
}
