package lifecycle

import (
	"context"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func (s *Service) DiscardResource(ctx context.Context, resourceType api.ResourceType, id, agent, group string) (api.WriteResult, error) {
	req := s.metadataClient.LifecycleAPI.ApiV2ResourceTypeIdDiscardPost(ctx, string(resourceType), id).XAgent(agent).XGroup(group)
	resp, err := s.metadataClient.LifecycleAPI.ApiV2ResourceTypeIdDiscardPostExecute(req)
	if err != nil {
		return api.WriteResult{}, fmt.Errorf("discard resource request: %w", err)
	}

	if resp == nil {
		return api.WriteResult{}, fmt.Errorf("discard resource request failed: empty response")
	}

	if resp.StatusCode != http.StatusOK {
		return api.WriteResult{}, fmt.Errorf("discard resource request failed: unexpected status %s", resp.Status)
	}

	return api.WriteResult{
		ResourceType: resourceType,
		ID:           id,
		Status:       api.DiscardedStatus,
	}, nil
}
