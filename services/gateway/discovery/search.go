package discovery

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	Providers        []string
	VersioningStatus []api.VersioningStatus
	Keywords         []string
}

func (p *SearchParams) buildMetadataSearchRequest(ctx context.Context, client *metadataclient.APIClient) (metadataclient.ApiApiV2AccessSearchGetRequest, error) {
	versioningStatus := make([]metadataclient.VersioningStatus, len(p.VersioningStatus))
	for i, v := range p.VersioningStatus {
		versioningStatus[i] = metadataclient.VersioningStatus(v)
	}

	supportedTypes := map[metadataclient.ResourceType]struct{}{
		metadataclient.DATASET:             {},
		metadataclient.ORGANIZATION:        {},
		metadataclient.SOFTWAREAPPLICATION: {},
		metadataclient.SOFTWARESOURCECODE:  {},
		metadataclient.WEBSERVICE:          {},
	}

	// intersect the types we support with the types asked for in the query
	types := []metadataclient.ResourceType{}
	if len(p.Types) == 0 {
		for t := range supportedTypes {
			types = append(types, t)
		}
	} else {
		for _, t := range p.Types {
			metadataType, err := toMetadataResourceType(t)
			if err != nil {
				return metadataclient.ApiApiV2AccessSearchGetRequest{}, fmt.Errorf("convert resource type: %w", err)
			}

			if _, ok := supportedTypes[metadataType]; ok {
				types = append(types, metadataType)
			}
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
			log.Error("metadata search request failed", "method", requestMethod, "url", requestURL, "error", err)
		} else {
			log.Error("metadata search request failed", "error", err)
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return api.SearchResponse{}, fmt.Errorf("search metadata timeout: %w", context.DeadlineExceeded)
		}

		return api.SearchResponse{}, fmt.Errorf("search metadata request failed")
	}

	log.Debug("metadata search response received", "total", resp.Total, "items_count", len(resp.Items), "page", resp.Page)

	searchResp := api.SearchResponse{
		PageNumber: int(params.PageNumber),
		PageSize:   int(params.PageSize),
		Total:      int(resp.Total),
		TotalPages: totalPages(int(resp.Total), int(params.PageSize)),
		Items:      []api.SearchItem{},
	}
	for i, item := range resp.Items {
		searchItem, err := toAPISearchItem(item)
		if err != nil {
			log.Warn("metadata search item parse failed", "index", i, "item_id", item.Id, "item_type", item.GetType(), "error", err)
			continue
		}

		searchResp.Items = append(searchResp.Items, searchItem)
	}

	return searchResp, nil
}

func totalPages(total, pageSize int) int {
	if total == 0 {
		return 0
	}

	return int(math.Ceil(float64(total) / float64(pageSize)))
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
	switch item.GetType() {
	case metadataclient.DATASET:
		return mapDatasetItem(item)
	case metadataclient.ORGANIZATION:
		return mapOrganizationItem(item)
	case metadataclient.SOFTWAREAPPLICATION:
		return mapSoftwareApplicationItem(item)
	case metadataclient.SOFTWARESOURCECODE:
		return mapSoftwareSourceCodeItem(item)
	case metadataclient.WEBSERVICE:
		return mapWebServiceItem(item)
	default:
		return nil, fmt.Errorf("unsupported metadata resource type: %q", item.GetType())
	}
}
