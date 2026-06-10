package lifecycle

import (
	"context"
	"fmt"
	"net/http"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/api"
)

func (s *Service) DraftResource(ctx context.Context, resourceType api.ResourceType, id, agent, group string) (api.WriteResult, error) {
	// TODO: limit only to one draft per user
	// TODO: do i have to check that the current status of the entity is "draftable"? or is it done in the metadata lifecycle?
	// TODO: implement the whole user/group thing
	req := s.metadataClient.LifecycleAPI.ApiV2ResourceTypeIdDraftPost(ctx, string(resourceType), id).XAgent(agent).XGroup(group)
	resp, err := s.metadataClient.LifecycleAPI.ApiV2ResourceTypeIdDraftPostExecute(req)
	if err != nil {
		return api.WriteResult{}, fmt.Errorf("draft resource request: %w", err)
	}

	if resp == nil {
		return api.WriteResult{}, fmt.Errorf("draft resource request failed: empty response")
	}

	if resp.StatusCode != http.StatusCreated {
		return api.WriteResult{}, fmt.Errorf("draft resource request failed: unexpected status %s", resp.Status)
	}

	return api.WriteResult{
		ResourceType: resourceType,
		ID:           id,
		Status:       api.DraftStatus,
	}, nil
}
