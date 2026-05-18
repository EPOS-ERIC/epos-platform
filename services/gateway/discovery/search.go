package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
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

func (p *SearchParams) buildMetadataSearchRequest(ctx context.Context, client *metadataclient.APIClient) (metadataclient.ApiSearchMetadataRequest, error) {
	versioningStatus := make([]metadataclient.VersioningStatus, len(p.VersioningStatus))
	for i, v := range p.VersioningStatus {
		versioningStatus[i] = metadataclient.VersioningStatus(v)
	}

	types := make([]metadataclient.ResourceType, len(p.Types))
	for i, t := range p.Types {
		metadataType, err := toMetadataResourceType(t)
		if err != nil {
			return metadataclient.ApiSearchMetadataRequest{}, fmt.Errorf("convert resource type: %w", err)
		}

		types[i] = metadataType
	}

	req := client.SearchAPI.SearchMetadata(ctx)

	if p.Query != "" {
		req = req.Q(p.Query)
	}
	if p.PageNumber > 0 {
		req = req.Page(p.PageNumber)
	}
	if p.PageSize > 0 {
		req = req.ResultsPerPage(p.PageSize)
	}
	if len(types) > 0 {
		req = req.Type_(types)
	}
	if p.SpatialFilter != "" {
		req = req.SpatialFilter(p.SpatialFilter)
	}
	if !p.DateFrom.IsZero() {
		req = req.DateFrom(p.DateFrom)
	}
	if !p.DateTo.IsZero() {
		req = req.DateTo(p.DateTo)
	}
	if len(p.ServiceProviders) > 0 {
		req = req.ServiceProvider(p.ServiceProviders)
	}
	if len(versioningStatus) > 0 {
		req = req.VersioningStatus(versioningStatus)
	}
	if len(p.Keywords) > 0 {
		req = req.Keywords(p.Keywords)
	}

	return req, nil
}

func (s *Service) Search(ctx context.Context, params SearchParams) (api.SearchResponse, error) {
	req, err := params.buildMetadataSearchRequest(ctx, s.metadataClient)
	if err != nil {
		return api.SearchResponse{}, fmt.Errorf("build metadata search request: %w", err)
	}

	resp, _, err := req.Execute()
	if err != nil {
		return api.SearchResponse{}, fmt.Errorf("search metadata: %w", err)
	}

	searchResp := api.SearchResponse{
		Total: int(resp.Total),
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
		return api.DatasetItem{
			BaseSearchItem: api.BaseSearchItem{
				Type: api.DatasetType,
				ID:   typed.GetId(),
				Name: typed.GetName(),
				Href: typed.GetHref(),
			},
			DataProviders: typed.GetServiceProviders(), // TODO: these should be data providers in the metadata lifecycle, not service providers
			Concepts:      typed.GetCategories(),
			Operations:    []string{}, // TODO the aggregation
		}, nil
	case metadataclient.GenericSearchItem:
		resourceType, err := fromMetadataResourceType(metadataclient.ResourceType(typed.GetType()))
		if err != nil {
			return nil, err
		}

		baseItem := api.BaseSearchItem{
			Type: resourceType,
			ID:   typed.GetId(),
			Name: typed.GetName(),
			Href: typed.GetHref(),
		}

		// as of now there is no distinction between these types in the metadata lifecycle, when there will be these can be implemented
		switch resourceType {
		case api.SoftwareApplicationType:
			return api.SoftwareApplicationItem{BaseSearchItem: baseItem}, nil
		case api.SoftwareSourceCodeType:
			return api.SoftwareSourceCodeItem{BaseSearchItem: baseItem}, nil
		// TODO: add organizations
		default:
			return nil, fmt.Errorf("unsupported gateway resource type: %q", resourceType)
		}
	default:
		return nil, fmt.Errorf("unsupported metadata search item type %T", actual)
	}
}
