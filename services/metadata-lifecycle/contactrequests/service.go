// Package contactrequests prepares emails for messages sent to resource contacts.
package contactrequests

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type targetReader interface {
	Resource(context.Context, domain.ResourceType, uuid.UUID) (string, []string, error)
	Distribution(context.Context, uuid.UUID) (string, []string, error)
}

type userReader interface {
	GetByID(context.Context, domain.Actor, string) (domain.User, error)
}

type Service struct {
	targets targetReader
	users   userReader
}

func New(targets targetReader, users userReader) *Service {
	return &Service{
		targets: targets,
		users:   users,
	}
}

type Target struct {
	ResourceType   domain.ResourceType
	ResourceID     uuid.UUID
	DistributionID uuid.UUID
}

type PreparedEmail struct {
	TargetEmail string
	Subject     string
	Body        string
}

func (s *Service) Prepare(ctx context.Context, actor domain.Actor, target Target, message string) ([]PreparedEmail, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	message = strings.TrimSpace(message)
	if message == "" {
		return nil, apperror.Field("message", "Message is required.", nil)
	}

	resourceTarget := target.ResourceType != "" || target.ResourceID != uuid.Nil
	distributionTarget := target.DistributionID != uuid.Nil
	if resourceTarget == distributionTarget {
		return nil, apperror.Field("target", "Select one resource or distribution target.", nil)
	}

	title, contactPoints, err := s.contactTarget(ctx, target)
	if err != nil {
		return nil, err
	}

	if len(contactPoints) == 0 {
		return nil, apperror.New(apperror.CodeNotFound, "No contact points are available for this resource.")
	}

	user, err := s.users.GetByID(ctx, actor, string(actor))
	if err != nil {
		return nil, fmt.Errorf("get contact request sender: %w", err)
	}

	view := emailView{
		ResourceTitle: title,
		Message:       message,
		SenderName:    strings.TrimSpace(user.GivenName + " " + user.FamilyName),
		SenderEmail:   user.Email,
	}
	body, err := renderBody(view)
	if err != nil {
		return nil, fmt.Errorf("render contact request email: %w", err)
	}

	result := make([]PreparedEmail, 0, len(contactPoints))
	for _, contactPoint := range contactPoints {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result = append(result, PreparedEmail{
			TargetEmail: contactPoint,
			Subject:     "Contact request: " + title,
			Body:        body,
		})
	}

	return result, nil
}

func (s *Service) contactTarget(ctx context.Context, target Target) (string, []string, error) {
	if target.DistributionID != uuid.Nil {
		title, emails, err := s.targets.Distribution(ctx, target.DistributionID)
		if err != nil {
			return "", nil, fmt.Errorf("get published web-service distribution: %w", err)
		}

		return title, emails, nil
	}

	if target.ResourceID == uuid.Nil {
		return "", nil, apperror.Field("resource_id", "Resource ID must be a UUID.", nil)
	}

	switch target.ResourceType {
	case domain.Dataset, domain.SoftwareApplication, domain.SoftwareSourceCode:
	default:
		return "", nil, apperror.Field("resource_type", "Unknown resource type.", nil)
	}

	title, emails, err := s.targets.Resource(ctx, target.ResourceType, target.ResourceID)
	if err != nil {
		return "", nil, fmt.Errorf("get published resource: %w", err)
	}

	return title, emails, nil
}

type emailView struct {
	ResourceTitle string
	Message       string
	SenderName    string
	SenderEmail   string
}

//go:embed contact_request.html
var contactRequestTemplate embed.FS

func renderBody(view emailView) (string, error) {
	parsed, err := template.ParseFS(contactRequestTemplate, "contact_request.html")
	if err != nil {
		return "", fmt.Errorf("parse contact request template: %w", err)
	}

	var body strings.Builder
	if err := parsed.ExecuteTemplate(&body, "contact_request.html", view); err != nil {
		return "", fmt.Errorf("execute contact request template: %w", err)
	}

	return body.String(), nil
}
