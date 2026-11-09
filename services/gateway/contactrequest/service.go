// Package contactrequest orchestrates preparation and synchronous delivery of contact emails.
package contactrequest

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/google/uuid"
)

type metadataLifecycle interface {
	PrepareResourceContactRequestEmails(context.Context, string, domain.ResourceType, uuid.UUID, string) ([]domain.ContactRequestEmail, error)
	PrepareDistributionContactRequestEmails(context.Context, string, uuid.UUID, string) ([]domain.ContactRequestEmail, error)
}

type EmailSender interface {
	SendContactRequest(context.Context, domain.ContactRequestEmail) error
}

type Service struct {
	metadataLifecycle metadataLifecycle
	sender            EmailSender
}

func New(metadataLifecycle metadataLifecycle, sender EmailSender) *Service {
	return &Service{
		metadataLifecycle: metadataLifecycle,
		sender:            sender,
	}
}

func (s *Service) RequestResource(ctx context.Context, actorID string, resourceType domain.ResourceType, resourceID uuid.UUID, message string) error {
	emails, err := s.metadataLifecycle.PrepareResourceContactRequestEmails(ctx, actorID, resourceType, resourceID, message)
	if err != nil {
		return fmt.Errorf("prepare resource contact request emails: %w", err)
	}

	return s.send(ctx, emails)
}

func (s *Service) RequestDistribution(ctx context.Context, actorID string, distributionID uuid.UUID, message string) error {
	emails, err := s.metadataLifecycle.PrepareDistributionContactRequestEmails(ctx, actorID, distributionID, message)
	if err != nil {
		return fmt.Errorf("prepare distribution contact request emails: %w", err)
	}

	return s.send(ctx, emails)
}

func (s *Service) send(ctx context.Context, emails []domain.ContactRequestEmail) error {
	if len(emails) == 0 {
		return apperror.New(apperror.CodeNotFound, "No contact points are available for this resource.")
	}

	sent := false
	for _, email := range emails {
		if s.sender == nil {
			continue
		}

		if err := s.sender.SendContactRequest(ctx, email); err != nil {
			continue
		}

		sent = true
	}

	if sent {
		return nil
	}

	return apperror.New(apperror.CodeUpstreamUnavailable, "Unable to send email to contact points.")
}
