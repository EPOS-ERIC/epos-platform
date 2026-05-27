package metadataaccess

import (
	"context"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/gen/metadataclient"
)

func FetchDetail(ctx context.Context, client *metadataclient.APIClient, shape, id string) (*metadataclient.AccessDetailResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("metadata client is not configured")
	}

	resp, httpResp, err := client.AccessAPI.ApiV2AccessShapeIdGet(ctx, shape, id).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return nil, domain.ErrNotFound
		}

		return nil, fmt.Errorf("fetch access detail for %s %q: %w", shape, id, err)
	}

	if resp == nil {
		return nil, fmt.Errorf("fetch access detail for %s %q: empty response", shape, id)
	}

	if resp.AccessDetailResponse == nil {
		if resp.AccessListResponse != nil {
			return nil, fmt.Errorf("fetch access detail for %s %q: unexpected list response", shape, id)
		}

		return nil, fmt.Errorf("fetch access detail for %s %q: empty typed response", shape, id)
	}

	return resp.AccessDetailResponse, nil
}
