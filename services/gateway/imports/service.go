// Package imports contains metadata import use-cases.
package imports

import (
	"context"
	"fmt"
	"io"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Service struct {
	metadataLifecycle *clients.MetadataLifecycle
}

func NewService(metadataLifecycle *clients.MetadataLifecycle) *Service {
	return &Service{metadataLifecycle: metadataLifecycle}
}

func (s *Service) Turtle(ctx context.Context, actorID string, body io.Reader) (*domain.ImportResult, error) {
	content, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read import body: %w: %v", domain.ErrBadRequest, err)
	}

	result, err := s.metadataLifecycle.ImportTurtle(ctx, actorID, content)
	if err != nil {
		return nil, fmt.Errorf("import Turtle metadata: %w", err)
	}

	return result, nil
}
