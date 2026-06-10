package lifecycle

import (
	"context"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func (s *Service) SubmitResource(ctx context.Context, resourceType api.ResourceType, id, agent, group string) (api.WriteResult, error) {
	req := s.metadataClient.LifecycleAPI.ApiV2ResourceTypeIdSubmitPost(ctx, string(resourceType), id).XAgent(agent).XGroup(group)
	resp, err := s.metadataClient.LifecycleAPI.ApiV2ResourceTypeIdSubmitPostExecute(req)
	if err != nil {
		return api.WriteResult{}, fmt.Errorf("submit resource request: %w", err)
	}

	if resp == nil {
		return api.WriteResult{}, fmt.Errorf("submit resource request failed: empty response")
	}

	if resp.StatusCode != http.StatusOK {
		return api.WriteResult{}, fmt.Errorf("submit resource request failed: unexpected status %s", resp.Status)
	}

	return api.WriteResult{
		ResourceType: resourceType,
		ID:           id,
		Status:       api.SubmittedStatus,
	}, nil
}
