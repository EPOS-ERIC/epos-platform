// Package discovery contains resource discovery use-cases.
package discovery

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type ResourceSearcher interface {
	SearchResources(context.Context, string, domain.DiscoveryFilter) ([]domain.DiscoveryItem, error)
}

type Service struct {
	searcher ResourceSearcher
}

func NewService(backoffice *clients.Backoffice) *Service {
	return &Service{searcher: backoffice}
}

func (s *Service) Search(ctx context.Context, actorID string, filter domain.DiscoveryFilter) ([]domain.DiscoveryItem, error) {
	items, err := s.searcher.SearchResources(ctx, actorID, filter)
	if err != nil {
		return nil, fmt.Errorf("search discovery resources: %w", err)
	}
	return items, nil
}
