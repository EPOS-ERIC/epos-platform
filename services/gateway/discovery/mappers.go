package discovery

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/gatewayhref"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/metadataaccess"
	"golang.org/x/sync/errgroup"
)

func (s *Service) mapDatasetItem(ctx context.Context, item metadataclient.SearchItem) (api.DatasetItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.DatasetItem{}, err
	}

	themes, err := getStringSliceField(item.Fields, "theme")
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("get dataset concepts: %w", err)
	}

	publishers, err := getStringSliceField(item.Fields, "publisher")
	if err != nil {
		return api.DatasetItem{}, fmt.Errorf("get dataset providers: %w", err)
	}

	var (
		providers     []api.ResourceRef
		concepts      []api.ResourceRef
		distributions []api.DatasetDistributionSearchItem
	)

	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		mappedProviders, err := s.mapProviders(groupCtx, publishers)
		if err != nil {
			return fmt.Errorf("map dataset providers: %w", err)
		}
		providers = mappedProviders
		return nil
	})
	g.Go(func() error {
		mappedConcepts, err := s.mapConcepts(groupCtx, themes)
		if err != nil {
			return fmt.Errorf("map dataset concepts: %w", err)
		}
		concepts = mappedConcepts
		return nil
	})
	g.Go(func() error {
		mappedDistributions, err := s.mapDistributions(groupCtx, baseItem.ID)
		if err != nil {
			return fmt.Errorf("map dataset distributions: %w", err)
		}
		distributions = mappedDistributions
		return nil
	})

	if err := g.Wait(); err != nil {
		return api.DatasetItem{}, err
	}

	return api.DatasetItem{
		BaseSearchItem: baseItem,
		Providers:      providers,
		Concepts:       concepts,
		Distributions:  distributions,
	}, nil
}

func mapOrganizationItem(item metadataclient.SearchItem) (api.OrganizationItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.OrganizationItem{}, err
	}

	// TODO: is it correct that sometimes an organization item from the metadataclient doesn't have a legal name? what do we put here in those cases?
	name, err := getField[string](item.Fields, "legal name")
	if err != nil {
		name = ""
	}
	baseItem.Name = name

	return api.OrganizationItem{BaseSearchItem: baseItem}, nil
}

func (s *Service) mapSoftwareApplicationItem(ctx context.Context, item metadataclient.SearchItem) (api.SoftwareApplicationItem, error) {
	baseItem, err := mapBaseItem(item)
	if err != nil {
		return api.SoftwareApplicationItem{}, err
	}

	publishers, err := getStringSliceField(item.Fields, "publishers")
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
		return api.SoftwareSourceCodeItem{}, err
	}

	publishers, err := getStringSliceField(item.Fields, "publishers")
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
		return api.WebServiceItem{}, err
	}

	publishers, err := getField[string](item.Fields, "provider")
	if err != nil {
		return api.WebServiceItem{}, fmt.Errorf("get webservice providers: %w", err)
	}

	themes, err := getStringSliceField(item.Fields, "theme")
	if err != nil {
		return api.WebServiceItem{}, fmt.Errorf("get webservice concepts: %w", err)
	}

	var (
		providers []api.ResourceRef
		concepts  []api.ResourceRef
	)

	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		mappedProviders, err := s.mapProviders(groupCtx, []string{publishers})
		if err != nil {
			return fmt.Errorf("map webservice providers: %w", err)
		}
		providers = mappedProviders
		return nil
	})
	g.Go(func() error {
		mappedConcepts, err := s.mapConcepts(groupCtx, themes)
		if err != nil {
			return fmt.Errorf("map webservice concepts: %w", err)
		}
		concepts = mappedConcepts
		return nil
	})

	if err := g.Wait(); err != nil {
		return api.WebServiceItem{}, err
	}

	return api.WebServiceItem{
		BaseSearchItem: baseItem,
		Provider:       providers,
		Concepts:       concepts,
	}, nil
}

func mapBaseItem(item metadataclient.SearchItem) (api.BaseSearchItem, error) {
	name, err := getField[string](item.Fields, "name")
	if err != nil {
		return api.BaseSearchItem{}, fmt.Errorf("get item name: %w", err)
	}

	resourceType, err := fromMetadataResourceType(item.GetType())
	if err != nil {
		return api.BaseSearchItem{}, err
	}

	return api.BaseSearchItem{
		Type: resourceType,
		ID:   item.Id,
		Name: name,
		Href: gatewayhref.Build(resourceType, item.Id),
	}, nil
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
				Href: gatewayhref.Build(api.OrganizationType, providerID),
			}

			detail, err := metadataaccess.FetchDetail(groupCtx, s.metadataClient, string(api.OrganizationType), providerID)
			if err != nil {
				s.log.Error("provider details fetch failed", "provider_id", providerID, "error", err)
				return fmt.Errorf("fetch provider '%s' details: %w", providerID, err)
			}

			names, err := getStringSliceField(detail.GetProperties(), "legalName")
			if err != nil {
				s.log.Warn("provider legal name extraction failed", "provider_id", providerID, "error", err)
				return fmt.Errorf("extract provider '%s' legal name: %w", providerID, err)
			}
			if len(names) > 0 {
				provider.Name = names[0]
			}

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
	distributions := []api.DatasetDistributionSearchItem{}
	detail, err := metadataaccess.FetchDetail(ctx, s.metadataClient, string(api.DatasetType), distributionID)
	if err != nil {
		return nil, fmt.Errorf("fetch dataset '%s' details: %w", distributionID, err)
	}

	refs, err := getLinkedRefsField(detail.GetProperties(), "distribution", api.DistributionType)
	if err != nil {
		return nil, fmt.Errorf("extract dataset '%s' distribution refs: %w", distributionID, err)
	}
	if len(refs) == 0 {
		return []api.DatasetDistributionSearchItem{}, nil
	}

	for _, distribution := range refs {
		distributions = append(distributions, api.DatasetDistributionSearchItem{
			ID:   distribution.ID,
			Name: distribution.Name,
			Href: gatewayhref.Build(api.DistributionType, distribution.ID),
		})
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
				Href: gatewayhref.Build(api.ConceptType, conceptID),
			}

			detail, err := metadataaccess.FetchDetail(groupCtx, s.metadataClient, string(api.ConceptType), conceptID)
			if err != nil {
				return fmt.Errorf("fetch concept '%s' details: %w", conceptID, err)
			}

			labels, err := getStringSliceField(detail.GetProperties(), "prefLabel")
			if err != nil {
				return fmt.Errorf("extract concept '%s' prefLabel: %w", conceptID, err)
			}
			if len(labels) > 0 {
				concept.Name = labels[0]
			}

			concepts[i] = concept
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("map concepts: %w", err)
	}

	return concepts, nil
}

func getField[T any](fields map[string]any, key string) (T, error) {
	var zero T
	raw, ok := fields[key]
	if !ok {
		return zero, fmt.Errorf("missing field '%s'", key)
	}

	val, ok := raw.(T)
	if !ok {
		return zero, fmt.Errorf("field '%s': expected %T, got %T", key, zero, raw)
	}

	return val, nil
}

func getStringSliceField(fields map[string]any, key string) ([]string, error) {
	raw, ok := fields[key]
	if !ok {
		return nil, fmt.Errorf("missing field '%s'", key)
	}

	if val, ok := raw.([]string); ok {
		return val, nil
	}

	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("field '%s': expected []string or []interface{}, got %T", key, raw)
	}

	result := make([]string, 0, len(values))
	for i, value := range values {
		strValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("field '%s[%d]': expected string, got %T", key, i, value)
		}

		result = append(result, strValue)
	}

	return result, nil
}

func getLinkedRefsField(fields map[string]any, key string, resourceType api.ResourceType) ([]api.ResourceRef, error) {
	raw, ok := fields[key]
	if !ok {
		return nil, fmt.Errorf("missing field '%s'", key)
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("field '%s': expected []interface{}, got %T", key, raw)
	}
	refs := make([]api.ResourceRef, 0, len(values))
	for i, v := range values {
		obj, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("field '%s[%d]': expected object, got %T", key, i, v)
		}
		id, ok := obj["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("field '%s[%d].id': missing/invalid", key, i)
		}

		name := ""
		nameField, ok := obj["name"].(string)
		if ok && nameField != "" {
			name = nameField
		}

		// Keep gateway-style href for consistency
		refs = append(refs, api.ResourceRef{
			ID:   id,
			Name: name,
			Href: gatewayhref.Build(resourceType, id),
		})
	}
	return refs, nil
}
