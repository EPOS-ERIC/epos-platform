// Package lifecycle contains resource detail domain use-cases.
package lifecycle

import "github.com/EPOS-ERIC/epos-platform/services/gateway/clients"

type Service struct {
	backoffice *clients.Backoffice
}

// NewService builds lifecycle service for Backoffice lifecycle operations.
func NewService(backoffice *clients.Backoffice) *Service {
	return &Service{backoffice: backoffice}
}
