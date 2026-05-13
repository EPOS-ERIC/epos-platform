package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/integrations/metadatalifecycle"
)

type SearchParams struct {
	Query            string
	PageSize         int32
	PageNumber       int32
	Types            []api.ResourceType
	SpatialFilter    string
	DateFrom         time.Time
	DateTo           time.Time
	ServiceProviders []string
	VersioningStatus []api.VersioningStatus
	Keywords         []string
}

func (p *SearchParams) ToMetadatalifecycleSearchParams() (metadatalifecycle.SearchParams, error) {
	versioningStatus := make([]metadataclient.VersioningStatus, len(p.VersioningStatus))
	for i, v := range p.VersioningStatus {
		versioningStatus[i] = metadataclient.VersioningStatus(v)
	}

	types := make([]metadataclient.ResourceType, len(p.Types))
	for i, t := range p.Types {
		metadataType, err := toMetadataResourceType(t)
		if err != nil {
			return metadatalifecycle.SearchParams{}, fmt.Errorf("TODO: %w", err)
		}

		types[i] = metadataType
	}

	dateFrom := &p.DateFrom
	if dateFrom.IsZero() {
		dateFrom = nil
	}
	dateTo := &p.DateTo
	if dateTo.IsZero() {
		dateTo = nil
	}

	return metadatalifecycle.SearchParams{
		Query:            p.Query,
		Page:             &p.PageNumber,
		ResultsPerPage:   &p.PageSize,
		Type:             types,
		SpatialFilter:    p.SpatialFilter,
		DateFrom:         dateFrom,
		DateTo:           dateTo,
		ServiceProvider:  p.ServiceProviders,
		VersioningStatus: versioningStatus,
		Keywords:         p.Keywords,
	}, nil
}

func (s *Service) Search(ctx context.Context, params SearchParams) (api.SearchResponse, error) {
	metadataParams, err := params.ToMetadatalifecycleSearchParams()
	if err != nil {
		return api.SearchResponse{}, fmt.Errorf("TODO: %w", err)
	}

	resp, err := s.metadataClient.Search(ctx, metadataParams)
	if err != nil {
		return api.SearchResponse{}, fmt.Errorf("TODO: %w", err)
	}

	searchResp := api.SearchResponse{
		Total: 0,
		Items: []api.SearchItem{},
	}
	for _, item := range resp.Items {
		searchItem, err := toAPISearchItem(item)
		if err != nil {
			log.Warn("error parsing metadata client search item", "error", err)
			continue
		}

		searchResp.Items = append(searchResp.Items, searchItem)
	}

	searchResp.Total = int32(len(searchResp.Items))

	return searchResp, nil
}

// enumMapping defines the canonical source-of-truth for ResourceType conversions.
// To add a new type, only this map needs to change.
var resourceTypeMatrix = map[api.ResourceType]metadataclient.ResourceType{
	api.DatasetType:             metadataclient.DATASET,
	api.DistributionType:        metadataclient.DISTRIBUTION,
	api.SoftwareApplicationType: metadataclient.SOFTWAREAPPLICATION,
	api.SoftwareSourceCodeType:  metadataclient.SOFTWARESOURCECODE,
}

func toMetadataResourceType(t api.ResourceType) (metadataclient.ResourceType, error) {
	v, ok := resourceTypeMatrix[t]
	if !ok {
		return "", fmt.Errorf("unsupported gateway resource type: %q", t)
	}
	return v, nil
}

func fromMetadataResourceType(t metadataclient.ResourceType) (api.ResourceType, error) {
	for gw, meta := range resourceTypeMatrix {
		if meta == t {
			return gw, nil
		}
	}
	return "", fmt.Errorf("unsupported metadata resource type: %q", t)
}

func toAPISearchItem(item metadataclient.SearchItem) (api.SearchItem, error) {
	actual := item.GetActualInstanceValue()
	switch typed := actual.(type) {
	case metadataclient.DatasetSearchItem:
		return api.SearchItem{
			Type:             api.DatasetType,
			ID:               typed.GetId(),
			Name:             typed.GetName(),
			Href:             typed.GetHref(),
			ServiceProviders: typed.GetServiceProviders(),
			Concepts:         typed.GetCategories(),
		}, nil
	case metadataclient.GenericSearchItem:
		resourceType, err := fromMetadataResourceType(metadataclient.ResourceType(typed.GetType()))
		if err != nil {
			return api.SearchItem{}, err
		}

		return api.SearchItem{
			Type: resourceType,
			ID:   typed.GetId(),
			Name: typed.GetName(),
			Href: typed.GetHref(),
		}, nil
	default:
		return api.SearchItem{}, fmt.Errorf("unsupported metadata search item type %T", actual)
	}
}
