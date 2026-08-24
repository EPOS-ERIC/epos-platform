// Package canonical contains Gateway canonical metadata use-cases.
package canonical

import (
	"context"
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/clients"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

type Service struct {
	backoffice *clients.Backoffice
}

func NewService(backoffice *clients.Backoffice) *Service {
	return &Service{backoffice: backoffice}
}

func (s *Service) ListContactPoints(ctx context.Context, actorID string) ([]domain.ContactPoint, error) {
	if strings.TrimSpace(actorID) == "" {
		return nil, fmt.Errorf("invalid contact-point actor: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return nil, fmt.Errorf("backoffice canonical client is not configured")
	}
	points, err := s.backoffice.ListContactPoints(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("list contact points through backoffice: %w", err)
	}
	return points, nil
}

func (s *Service) GetContactPoint(ctx context.Context, actorID, id string) (domain.ContactPoint, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(id) == "" {
		return domain.ContactPoint{}, fmt.Errorf("invalid contact-point request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return domain.ContactPoint{}, fmt.Errorf("backoffice canonical client is not configured")
	}
	point, err := s.backoffice.GetContactPoint(ctx, actorID, id)
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("get contact point through backoffice: %w", err)
	}
	return point, nil
}

func (s *Service) CreateContactPoint(ctx context.Context, actorID string, input domain.CreateContactPointInput) (domain.ContactPoint, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Role) == "" {
		return domain.ContactPoint{}, fmt.Errorf("invalid contact-point request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return domain.ContactPoint{}, fmt.Errorf("backoffice canonical client is not configured")
	}
	point, err := s.backoffice.CreateContactPoint(ctx, actorID, input)
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("create contact point through backoffice: %w", err)
	}
	return point, nil
}

func (s *Service) ReplaceContactPoint(ctx context.Context, actorID, id string, input domain.ReplaceContactPointInput) (domain.ContactPoint, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(id) == "" || strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Role) == "" {
		return domain.ContactPoint{}, fmt.Errorf("invalid contact-point request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return domain.ContactPoint{}, fmt.Errorf("backoffice canonical client is not configured")
	}
	point, err := s.backoffice.ReplaceContactPoint(ctx, actorID, id, input)
	if err != nil {
		return domain.ContactPoint{}, fmt.Errorf("replace contact point through backoffice: %w", err)
	}
	return point, nil
}

func (s *Service) DeleteContactPoint(ctx context.Context, actorID, id string) error {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(id) == "" {
		return fmt.Errorf("invalid contact-point request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice canonical client is not configured")
	}
	if err := s.backoffice.DeleteContactPoint(ctx, actorID, id); err != nil {
		return fmt.Errorf("delete contact point through backoffice: %w", err)
	}
	return nil
}

func (s *Service) ListOrganizations(ctx context.Context) ([]domain.CanonicalOrganization, error) {
	if s == nil || s.backoffice == nil {
		return nil, fmt.Errorf("backoffice canonical client is not configured")
	}
	organizations, err := s.backoffice.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations through backoffice: %w", err)
	}
	return organizations, nil
}

func (s *Service) GetOrganization(ctx context.Context, id string) (domain.CanonicalOrganization, error) {
	if strings.TrimSpace(id) == "" {
		return domain.CanonicalOrganization{}, fmt.Errorf("invalid organization id: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("backoffice canonical client is not configured")
	}
	organization, err := s.backoffice.GetOrganization(ctx, id)
	if err != nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("get organization through backoffice: %w", err)
	}
	return organization, nil
}

func (s *Service) CreateOrganization(ctx context.Context, actorID string, input domain.CreateOrganizationInput) (domain.CanonicalOrganization, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(input.Name) == "" {
		return domain.CanonicalOrganization{}, fmt.Errorf("invalid organization request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("backoffice canonical client is not configured")
	}
	organization, err := s.backoffice.CreateOrganization(ctx, actorID, input)
	if err != nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("create organization through backoffice: %w", err)
	}
	return organization, nil
}

func (s *Service) ReplaceOrganization(ctx context.Context, actorID, id string, input domain.ReplaceOrganizationInput) (domain.CanonicalOrganization, error) {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(id) == "" || strings.TrimSpace(input.Name) == "" {
		return domain.CanonicalOrganization{}, fmt.Errorf("invalid organization request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("backoffice canonical client is not configured")
	}
	organization, err := s.backoffice.ReplaceOrganization(ctx, actorID, id, input)
	if err != nil {
		return domain.CanonicalOrganization{}, fmt.Errorf("replace organization through backoffice: %w", err)
	}
	return organization, nil
}

func (s *Service) DeleteOrganization(ctx context.Context, actorID, id string) error {
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(id) == "" {
		return fmt.Errorf("invalid organization request: %w", domain.ErrBadRequest)
	}
	if s == nil || s.backoffice == nil {
		return fmt.Errorf("backoffice canonical client is not configured")
	}
	if err := s.backoffice.DeleteOrganization(ctx, actorID, id); err != nil {
		return fmt.Errorf("delete organization through backoffice: %w", err)
	}
	return nil
}
