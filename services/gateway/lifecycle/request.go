package lifecycle

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
)

func (s *Service) Request(ctx context.Context, method, path, actorID string, body []byte) (clients.BackofficeResponse, error) {
	if s == nil || s.backoffice == nil {
		return clients.BackofficeResponse{}, fmt.Errorf("backoffice lifecycle client is not configured")
	}
	response, err := s.backoffice.Request(ctx, method, path, body, actorID)
	if err != nil {
		return clients.BackofficeResponse{}, fmt.Errorf("request backoffice lifecycle: %w", err)
	}
	return response, nil
}
