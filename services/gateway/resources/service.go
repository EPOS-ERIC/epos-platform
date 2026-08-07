// Package resources contains resource detail domain use-cases.
package resources

import (
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/packages/go/logging"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

type Service struct {
	log       *slog.Logger
	resources *clients.Resources
}

func resourceID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, domain.ErrBadRequest
	}
	return id, nil
}

// NewService builds resources service for canonical resource lookups.
func NewService(resources *clients.Resources) *Service {
	return &Service{
		log:       logging.Get("resources-service"),
		resources: resources,
	}
}
