package discovery

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
	"golang.org/x/sync/errgroup"
)

var supportedSearchTypes = []api.ResourceType{
	api.DatasetType,
	api.OrganizationType,
	api.SoftwareApplicationType,
	api.SoftwareSourceCodeType,
	api.WebserviceType,
}

var (
	ErrUpstreamTimeout       = errors.New("upstream timeout")
	ErrUpstreamRequestFailed = errors.New("upstream request failed")
)

func ValidateSearchTypes(types []api.ResourceType) error {
	for _, resourceType := range types {
		if !slices.Contains(supportedSearchTypes, resourceType) {
			return fmt.Errorf("unsupported search type '%s'; allowed values: %s", resourceType, supportedSearchTypes)
		}
	}

	return nil
}

type SearchParams struct {
	Query            string
	PageSize         int32
	PageNumber       int32
	Types            []api.ResourceType
	SpatialFilter    string
	DateFrom         time.Time
	DateTo           time.Time
	Providers        []string
	VersioningStatus []api.VersioningStatus
	Keywords         []string
}

func (p *SearchParams) buildMetadataSearchRequest(ctx context.Context, client *metadataclient.APIClient) (metadataclient.ApiApiV2AccessSearchGetRequest, error) {
	if err := ValidateSearchTypes(p.Types); err != nil {
		return metadataclient.ApiApiV2AccessSearchGetRequest{}, err
	}

	versioningStatus := make([]metadataclient.VersioningStatus, len(p.VersioningStatus))
	for i, v := range p.VersioningStatus {
		versioningStatus[i] = metadataclient.VersioningStatus(v)
	}

	types := []metadataclient.ResourceType{}
	if len(p.Types) == 0 {
		for _, resourceType := range supportedSearchTypes {
			metadataType, err := toMetadataResourceType(resourceType)
			if err != nil {
				return metadataclient.ApiApiV2AccessSearchGetRequest{}, fmt.Errorf("convert resource type: %w", err)
			}

			types = append(types, metadataType)
		}
	} else {
		for _, t := range p.Types {
			metadataType, err := toMetadataResourceType(t)
			if err != nil {
				return metadataclient.ApiApiV2AccessSearchGetRequest{}, fmt.Errorf("convert resource type: %w", err)
			}

			types = append(types, metadataType)
		}
	}

	req := client.AccessAPI.ApiV2AccessSearchGet(ctx)

	if p.Query != "" {
		req = req.Q(p.Query)
	}
	if p.PageNumber > 0 {
		req = req.Page(p.PageNumber)
	}
	if p.PageSize > 0 {
		req = req.ResultsPerPage(p.PageSize)
	}
	req = req.Type_(types)
	if p.SpatialFilter != "" {
		req = req.SpatialFilter(p.SpatialFilter)
	}
	if !p.DateFrom.IsZero() {
		req = req.DateFrom(p.DateFrom)
	}
	if !p.DateTo.IsZero() {
		req = req.DateTo(p.DateTo)
	}
	if len(p.Providers) > 0 {
		req = req.ServiceProvider(p.Providers)
	}
	if len(versioningStatus) > 0 {
		req = req.VersioningStatus(versioningStatus)
	}
	if len(p.Keywords) > 0 {
		req = req.Keywords(p.Keywords)
	}
	req = req.IncludeConstraints(true)

	return req, nil
}

func (s *Service) Search(ctx context.Context, params SearchParams) (api.SearchResponse, error) {
	req, err := params.buildMetadataSearchRequest(ctx, s.metadataClient)
	if err != nil {
		return api.SearchResponse{}, fmt.Errorf("build metadata search request: %w", err)
	}

	resp, httpResp, err := req.Execute()
	if err != nil {
		requestMethod := ""
		requestURL := ""
		if httpResp != nil && httpResp.Request != nil {
			requestMethod = httpResp.Request.Method
			if httpResp.Request.URL != nil {
				requestURL = httpResp.Request.URL.String()
			}
		}

		if requestURL != "" {
			s.log.Error("metadata search request failed", "method", requestMethod, "url", requestURL, "error", err)
		} else {
			s.log.Error("metadata search request failed", "error", err)
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return api.SearchResponse{}, fmt.Errorf("search metadata timeout: %w", ErrUpstreamTimeout)
		}

		if errors.Is(err, context.Canceled) {
			return api.SearchResponse{}, fmt.Errorf("search metadata canceled: %w", ErrUpstreamTimeout)
		}

		return api.SearchResponse{}, fmt.Errorf("search metadata request failed: %w", ErrUpstreamRequestFailed)
	}

	s.log.Debug("metadata search response received", "total", resp.Total, "items_count", len(resp.Items), "page", resp.Page)

	searchResp := api.SearchResponse{
		PageNumber: int(resp.Page),
		PageSize:   len(resp.Items),
		Total:      int(resp.Total),
		TotalPages: int(resp.TotalPages),
		Items:      []api.SearchItem{},
	}
	mappedItems := make([]api.SearchItem, len(resp.Items))

	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.fanOutLimit)
	for i, item := range resp.Items {
		g.Go(func() error {
			searchItem, err := s.toAPISearchItem(groupCtx, item)
			if err != nil {
				s.log.Warn("metadata search item parse failed", "index", i, "item id", item.Id, "item type", item.GetType(), "error", err)
				return nil
			}

			mappedItems[i] = searchItem
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return api.SearchResponse{}, fmt.Errorf("map search items: %w", err)
	}

	for i := range resp.Items {
		if mappedItems[i] != nil {
			searchResp.Items = append(searchResp.Items, mappedItems[i])
		}
	}

	return searchResp, nil
}

// enumMapping defines the canonical source-of-truth for ResourceType conversions.
// To add a new type, only this map needs to change.
var resourceTypeMatrix = map[api.ResourceType]metadataclient.ResourceType{
	api.DatasetType:             metadataclient.DATASET,
	api.SoftwareApplicationType: metadataclient.SOFTWAREAPPLICATION,
	api.SoftwareSourceCodeType:  metadataclient.SOFTWARESOURCECODE,
	api.OrganizationType:        metadataclient.ORGANIZATION,
	api.WebserviceType:          metadataclient.WEBSERVICE,
}

func toMetadataResourceType(t api.ResourceType) (metadataclient.ResourceType, error) {
	v, ok := resourceTypeMatrix[t]
	if !ok {
		return "", fmt.Errorf("unsupported gateway resource type: '%s'", t)
	}
	return v, nil
}

func fromMetadataResourceType(t metadataclient.ResourceType) (api.ResourceType, error) {
	for gw, meta := range resourceTypeMatrix {
		if meta == t {
			return gw, nil
		}
	}
	return "", fmt.Errorf("unsupported metadata resource type: '%s'", t)
}

func (s *Service) toAPISearchItem(ctx context.Context, item metadataclient.SearchItem) (api.SearchItem, error) {
	switch item.GetType() {
	case metadataclient.DATASET:
		return s.mapDatasetItem(ctx, item)
	case metadataclient.ORGANIZATION:
		return mapOrganizationItem(item)
	case metadataclient.SOFTWAREAPPLICATION:
		return s.mapSoftwareApplicationItem(ctx, item)
	case metadataclient.SOFTWARESOURCECODE:
		return s.mapSoftwareSourceCodeItem(ctx, item)
	case metadataclient.WEBSERVICE:
		return s.mapWebServiceItem(ctx, item)
	default:
		return nil, fmt.Errorf("unsupported metadata resource type: '%s'", item.GetType())
	}
}
